package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/typed-web-commons/typed-web/internal/bindings"
	"github.com/typed-web-commons/typed-web/internal/labapi"
	"github.com/typed-web-commons/typed-web/internal/labengine"
	"github.com/typed-web-commons/typed-web/internal/mcpstdio"
	"github.com/typed-web-commons/typed-web/internal/twircontract"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("subcommand required: invoke, verify, schema, openapi, generate, mcp, or serve")
	}
	switch args[0] {
	case "invoke":
		return runInvoke(ctx, args[1:], stdout, stderr)
	case "verify":
		return runVerify(args[1:], stdout, stderr)
	case "schema":
		return runSchema(args[1:], stdout, stderr)
	case "openapi":
		return runOpenAPI(args[1:], stdout, stderr)
	case "generate":
		return runGenerate(args[1:], stdout, stderr)
	case "mcp":
		return runMCP(ctx, args[1:], stdin, stdout, stderr)
	case "serve":
		return runServe(ctx, args[1:], stderr)
	default:
		return fmt.Errorf("unknown subcommand %q", args[0])
	}
}

type inputsFlag map[string]string

func (values inputsFlag) String() string { return fmt.Sprint(map[string]string(values)) }
func (values inputsFlag) Set(raw string) error {
	key, value, ok := strings.Cut(raw, "=")
	if !ok || key == "" {
		return errors.New("input must be key=value")
	}
	if _, exists := values[key]; exists {
		return fmt.Errorf("duplicate input %q", key)
	}
	values[key] = value
	return nil
}

func common(fs *flag.FlagSet) (*string, *string) {
	root := fs.String("root", ".", "repository root")
	results := fs.String("results", "var/e2/results", "result bundle directory")
	return root, results
}

func runInvoke(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("invoke", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root, results := common(fs)
	origin := fs.String("origin", "", "catalog origin ID")
	operation := fs.String("operation", "", "operation ID")
	mode := fs.String("mode", labengine.ModeReplay, "fresh or replay")
	input := inputsFlag{}
	fs.Var(input, "input", "typed input key=value (repeatable)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *origin == "" || *operation == "" {
		return errors.New("invoke requires --origin and --operation")
	}
	engine, err := labengine.New(*root, *results)
	if err != nil {
		return err
	}
	invocation, err := engine.Invoke(ctx, labengine.Request{OriginID: *origin, OperationID: *operation, Mode: *mode, Input: input})
	if err != nil {
		return err
	}
	return writeJSON(stdout, labengine.View(invocation))
}

func runVerify(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root, results := common(fs)
	bundle := fs.String("bundle", "", "proof bundle directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *bundle == "" {
		return errors.New("verify requires --bundle")
	}
	engine, err := labengine.New(*root, *results)
	if err != nil {
		return err
	}
	result, publication, err := engine.Verify(*bundle)
	if err != nil {
		return err
	}
	return writeJSON(stdout, map[string]any{"status": "verified", "result_id": publication.ResultID, "result_digest": publication.ResultDigest, "bundle_id": publication.BundleID, "origin_id": result.OriginID, "operation_id": result.OperationID, "field_count": len(result.Fields)})
}

func runSchema(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("schema", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".", "repository root")
	operation := fs.String("operation", "", "operation ID")
	if err := fs.Parse(args); err != nil {
		return err
	}
	engine, err := labengine.New(*root, filepath.Join(os.TempDir(), "twirx-schema-unused"))
	if err != nil {
		return err
	}
	op, err := engine.Contracts.Find(*operation)
	if err != nil {
		return err
	}
	data, err := twircontract.JSONSchema(op)
	if err != nil {
		return err
	}
	_, err = stdout.Write(append(data, '\n'))
	return err
}

func runOpenAPI(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("openapi", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".", "repository root")
	if err := fs.Parse(args); err != nil {
		return err
	}
	engine, err := labengine.New(*root, filepath.Join(os.TempDir(), "twirx-openapi-unused"))
	if err != nil {
		return err
	}
	data, err := bindings.OpenAPI(engine.Contracts)
	if err != nil {
		return err
	}
	_, err = stdout.Write(append(data, '\n'))
	return err
}

func runGenerate(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".", "repository root")
	out := fs.String("out", "generated/e2", "output directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	engine, err := labengine.New(*root, filepath.Join(os.TempDir(), "twirx-generate-unused"))
	if err != nil {
		return err
	}
	if err := bindings.Write(*out, engine.Contracts); err != nil {
		return err
	}
	if err := bindings.WritePublicProof(*out, engine.Contracts, engine.Catalog); err != nil {
		return err
	}
	return writeJSON(stdout, map[string]any{"status": "generated", "directory": *out, "operations": len(engine.Contracts.Operations)})
}

func runMCP(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root, results := common(fs)
	mode := fs.String("mode", labengine.ModeReplay, "fresh or replay")
	if err := fs.Parse(args); err != nil {
		return err
	}
	engine, err := labengine.New(*root, *results)
	if err != nil {
		return err
	}
	server := mcpstdio.Server{Engine: engine, Mode: *mode}
	return server.Serve(ctx, stdin, stdout)
}

func runServe(ctx context.Context, args []string, stderr io.Writer) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root, results := common(fs)
	listenAddress := fs.String("listen", "127.0.0.1:8090", "loopback listen address")
	staticDir := fs.String("static", "lab/static", "Lab static asset directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	host, _, err := net.SplitHostPort(*listenAddress)
	if err != nil {
		return fmt.Errorf("serve: invalid listen address: %w", err)
	}
	address, err := netip.ParseAddr(host)
	if err != nil || !address.IsLoopback() {
		return errors.New("serve: --listen must use a literal loopback address; Caddy owns the public edge")
	}
	engine, err := labengine.New(*root, *results)
	if err != nil {
		return err
	}
	handler, err := labapi.New(labapi.Config{Engine: engine, StaticDir: *staticDir, AuditWriter: stderr})
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", *listenAddress)
	if err != nil {
		return fmt.Errorf("serve: listen: %w", err)
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 32 << 10}
	stopped := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = server.Shutdown(shutdownCtx)
		case <-stopped:
		}
	}()
	err = server.Serve(listener)
	close(stopped)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func writeJSON(output io.Writer, value any) error {
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
