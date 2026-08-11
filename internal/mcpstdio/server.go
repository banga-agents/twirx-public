// Package mcpstdio exposes the canonical E2 operation set over the local MCP
// stdio transport. It implements the 2025-11-25 lifecycle and tools subset;
// stdout contains protocol messages only.
package mcpstdio

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/typed-web-commons/typed-web/internal/bindings"
	"github.com/typed-web-commons/typed-web/internal/jsonbounded"
	"github.com/typed-web-commons/typed-web/internal/labengine"
	"github.com/typed-web-commons/typed-web/internal/twircontract"
)

const ProtocolVersion = "2025-11-25"
const MaxMessageBytes = 1 << 20

type Server struct {
	Engine      *labengine.Engine
	Mode        string
	initialized bool
}
type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}
type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *Server) Serve(ctx context.Context, input io.Reader, output io.Writer) error {
	if s.Engine == nil {
		return errors.New("mcpstdio: engine is required")
	}
	if s.Mode == "" {
		s.Mode = labengine.ModeReplay
	}
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 4096), MaxMessageBytes)
	writer := bufio.NewWriter(output)
	defer writer.Flush()
	for scanner.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		line := append([]byte(nil), scanner.Bytes()...)
		reply, respond := s.handle(ctx, line)
		if !respond {
			continue
		}
		encoded, err := json.Marshal(reply)
		if err != nil {
			return err
		}
		if len(encoded) > MaxMessageBytes {
			return errors.New("mcpstdio: response exceeds message limit")
		}
		if _, err := writer.Write(append(encoded, '\n')); err != nil {
			return err
		}
		if err := writer.Flush(); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("mcpstdio: read: %w", err)
	}
	return nil
}

func (s *Server) handle(ctx context.Context, line []byte) (response, bool) {
	var req request
	policy := jsonbounded.Policy{MaxBytes: MaxMessageBytes, MaxDepth: 32, MaxScalarBytes: 256 << 10, MaxContainerEntries: 4096, MaxTokens: 32768}
	if err := jsonbounded.Decode(line, &req, policy, true); err != nil {
		return errorResponse(nil, -32700, "Parse error"), true
	}
	if req.JSONRPC != "2.0" || req.Method == "" {
		return errorResponse(req.ID, -32600, "Invalid Request"), len(req.ID) > 0
	}
	notification := len(req.ID) == 0
	switch req.Method {
	case "initialize":
		if notification {
			return response{}, false
		}
		var params struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil || params.ProtocolVersion == "" {
			return errorResponse(req.ID, -32602, "Invalid initialize params"), true
		}
		s.initialized = true
		return success(req.ID, map[string]any{"protocolVersion": ProtocolVersion, "capabilities": map[string]any{"tools": map[string]any{"listChanged": false}}, "serverInfo": map[string]any{"name": "twirx-lab", "version": "0.1.0-e2"}, "instructions": "Catalog-only read operations. Results preserve native values and field provenance."}), true
	case "notifications/initialized":
		return response{}, false
	case "ping":
		if notification {
			return response{}, false
		}
		return success(req.ID, map[string]any{}), true
	}
	if !s.initialized {
		return errorResponse(req.ID, -32002, "Server not initialized"), !notification
	}
	switch req.Method {
	case "tools/list":
		if notification {
			return response{}, false
		}
		tools, err := bindings.Tools(s.Engine.Contracts)
		if err != nil {
			return errorResponse(req.ID, -32603, "Binding generation failed"), true
		}
		return success(req.ID, map[string]any{"tools": tools}), true
	case "tools/call":
		if notification {
			return response{}, false
		}
		return s.callTool(ctx, req)
	default:
		return errorResponse(req.ID, -32601, "Method not found"), !notification
	}
}

func (s *Server) callTool(ctx context.Context, req request) (response, bool) {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := decodeNumbers(req.Params, &params); err != nil || params.Name == "" {
		return errorResponse(req.ID, -32602, "Invalid tools/call params"), true
	}
	op, err := s.Engine.Contracts.Find(params.Name)
	if err != nil {
		return toolError(req.ID, "Unknown tool"), true
	}
	input, err := twircontract.NormalizeInput(op, params.Arguments)
	if err != nil {
		return toolError(req.ID, "Invocation input rejected: "+err.Error()), true
	}
	invocation, err := s.Engine.Invoke(ctx, labengine.Request{OriginID: op.OriginID, OperationID: op.ID, Input: input, Mode: s.Mode})
	if err != nil {
		return toolError(req.ID, "Invocation rejected: "+err.Error()), true
	}
	view := labengine.View(invocation)
	structured, err := toMap(view)
	if err != nil {
		return errorResponse(req.ID, -32603, "Result encoding failed"), true
	}
	text := fmt.Sprintf("%s returned %d provenance-bearing fields (%s)", op.ID, len(view.Fields), view.ResultID)
	return success(req.ID, map[string]any{"content": []any{map[string]any{"type": "text", "text": text}}, "structuredContent": structured, "isError": false}), true
}

func toMap(value any) (map[string]any, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	err = json.Unmarshal(encoded, &result)
	return result, err
}

func decodeNumbers(data []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("mcpstdio: trailing params")
	}
	return nil
}
func success(id json.RawMessage, result any) response {
	return response{JSONRPC: "2.0", ID: id, Result: result}
}
func errorResponse(id json.RawMessage, code int, message string) response {
	return response{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: message}}
}
func toolError(id json.RawMessage, message string) response {
	return success(id, map[string]any{"content": []any{map[string]any{"type": "text", "text": message}}, "isError": true})
}
