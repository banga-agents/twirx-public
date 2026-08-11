// Package bindings derives E2 JSON Schema, OpenAPI, MCP, and CLI reference
// material from the canonical TWIR contract set.
package bindings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/typed-web-commons/typed-web/internal/atomicfile"
	"github.com/typed-web-commons/typed-web/internal/origincatalog"
	"github.com/typed-web-commons/typed-web/internal/twircontract"
)

const MaxGeneratedBytes = 2 << 20

type Tool struct {
	Name        string          `json:"name"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	InputSchema map[string]any  `json:"inputSchema"`
	Annotations ToolAnnotations `json:"annotations"`
}
type ToolAnnotations struct {
	ReadOnlyHint    bool `json:"readOnlyHint"`
	DestructiveHint bool `json:"destructiveHint"`
	IdempotentHint  bool `json:"idempotentHint"`
	OpenWorldHint   bool `json:"openWorldHint"`
}

func Tools(set *twircontract.Set) ([]Tool, error) {
	tools := make([]Tool, 0, len(set.Operations))
	for i := range set.Operations {
		op := &set.Operations[i]
		schemaBytes, err := twircontract.JSONSchema(op)
		if err != nil {
			return nil, err
		}
		var schema map[string]any
		if err := json.Unmarshal(schemaBytes, &schema); err != nil {
			return nil, err
		}
		tools = append(tools, Tool{Name: op.ID, Title: op.Title, Description: op.Description, InputSchema: schema, Annotations: ToolAnnotations{ReadOnlyHint: true, DestructiveHint: false, IdempotentHint: true, OpenWorldHint: false}})
	}
	return tools, nil
}

func OpenAPI(set *twircontract.Set) ([]byte, error) {
	components := make(map[string]any, len(set.Operations))
	oneOf := make([]any, 0, len(set.Operations))
	for i := range set.Operations {
		op := &set.Operations[i]
		schemaBytes, err := twircontract.JSONSchema(op)
		if err != nil {
			return nil, err
		}
		var inputSchema map[string]any
		if err := json.Unmarshal(schemaBytes, &inputSchema); err != nil {
			return nil, err
		}
		name := componentName(op.ID) + "Input"
		components[name] = inputSchema
		requestName := componentName(op.ID) + "Request"
		components[requestName] = map[string]any{"type": "object", "additionalProperties": false, "required": []string{"origin_id", "operation_id", "input"}, "properties": map[string]any{"origin_id": map[string]any{"const": op.OriginID}, "operation_id": map[string]any{"const": op.ID}, "mode": map[string]any{"const": "replay", "default": "replay", "description": "The deployable HTTP Lab is replay-only."}, "input": map[string]any{"$ref": "#/components/schemas/" + name}}}
		oneOf = append(oneOf, map[string]any{"$ref": "#/components/schemas/" + requestName})
	}
	spec := map[string]any{"openapi": "3.1.0", "info": map[string]any{"title": "TWIRX Live Provenance Lab", "version": "0.1.0-e2", "description": "Catalog-only, read-only replay invocation with downloadable provenance bundles. Fresh-origin execution is not exposed by this HTTP surface."}, "servers": []any{map[string]any{"url": "https://lab.twirx.org"}}, "paths": map[string]any{"/api/v1/invoke": map[string]any{"post": map[string]any{"operationId": "invoke", "summary": "Invoke one admitted replay operation", "requestBody": map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"oneOf": oneOf}}}}, "responses": map[string]any{"200": map[string]any{"description": "Canonical typed result and provenance view"}, "400": map[string]any{"description": "Bounded typed request rejected"}, "429": map[string]any{"description": "Rate or concurrency limit reached"}}}}}, "components": map[string]any{"schemas": components}}
	return marshal(spec)
}

func MCPToolsJSON(set *twircontract.Set) ([]byte, error) {
	tools, err := Tools(set)
	if err != nil {
		return nil, err
	}
	return marshal(map[string]any{"protocolVersion": "2025-11-25", "tools": tools})
}

func CLIReference(set *twircontract.Set) []byte {
	var out strings.Builder
	out.WriteString("TWIRX E2 generated CLI operations\n\n")
	for _, op := range set.Operations {
		out.WriteString(op.ID + " — " + op.Title + "\n  origin: " + op.OriginID + "\n  effect: read\n")
		if len(op.Input) == 0 {
			out.WriteString("  input: none\n")
		} else {
			out.WriteString("  input:\n")
			for _, field := range op.Input {
				required := "optional"
				if field.Required {
					required = "required"
				}
				out.WriteString(fmt.Sprintf("    %s (%s, %s)\n", field.ID, field.Type, required))
			}
		}
		out.WriteByte('\n')
	}
	return []byte(strings.TrimRight(out.String(), "\n") + "\n")
}

func Write(dir string, set *twircontract.Set) error {
	if err := os.MkdirAll(filepath.Join(dir, "json-schema"), 0o750); err != nil {
		return err
	}
	for i := range set.Operations {
		op := &set.Operations[i]
		data, err := twircontract.JSONSchema(op)
		if err != nil {
			return err
		}
		data = append(data, '\n')
		if err := atomicfile.Write(filepath.Join(dir, "json-schema", op.ID+".json"), data, MaxGeneratedBytes, 0o640); err != nil {
			return err
		}
	}
	openapi, err := OpenAPI(set)
	if err != nil {
		return err
	}
	mcp, err := MCPToolsJSON(set)
	if err != nil {
		return err
	}
	files := map[string][]byte{"openapi.json": append(openapi, '\n'), "mcp-tools.json": append(mcp, '\n'), "cli.txt": CLIReference(set)}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := atomicfile.Write(filepath.Join(dir, name), files[name], MaxGeneratedBytes, 0o640); err != nil {
			return err
		}
	}
	return nil
}

func WritePublicProof(dir string, set *twircontract.Set, catalog *origincatalog.Catalog) error {
	operations := make([]string, 0, len(set.Operations))
	for _, operation := range set.Operations {
		operations = append(operations, operation.ID)
	}
	origins := make([]string, 0, len(catalog.Origins))
	for _, origin := range catalog.Origins {
		origins = append(origins, origin.ID)
	}
	value := map[string]any{
		"format": "tw.public-proof-data/0.1", "release_label": "Genesis Preview",
		"engineering_gate": "E2", "gate_status": "implementation_candidate",
		"origin_count": len(origins), "origins": origins, "operation_count": len(operations), "operations": operations,
		"bindings":              []string{"cli", "json-schema", "local-mcp", "openapi"},
		"independent_verifiers": []string{"go", "restricted-c"}, "result_format": "tw.result/0.2",
		"catalog_only": true, "read_only": true, "arbitrary_url_input": false,
		"claim": "Records an origin representation and declared derivation; does not assert objective truth.",
	}
	data, err := marshal(value)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return atomicfile.Write(filepath.Join(dir, "public-proof.json"), data, MaxGeneratedBytes, 0o640)
}

func marshal(value any) ([]byte, error) { return json.MarshalIndent(value, "", "  ") }

func componentName(id string) string {
	parts := strings.FieldsFunc(id, func(r rune) bool { return r == '.' || r == '-' || r == ':' })
	for i := range parts {
		if parts[i] != "" {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}
