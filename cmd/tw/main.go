package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/typed-web-commons/typed-web/internal/adapter"
	"github.com/typed-web-commons/typed-web/internal/atomicfile"
	"github.com/typed-web-commons/typed-web/internal/cas"
	"github.com/typed-web-commons/typed-web/internal/observation"
	"github.com/typed-web-commons/typed-web/internal/safefetch"
)

const version = "0.1.0-genesis"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		usage()
		return errors.New("a subcommand is required")
	}
	switch args[0] {
	case "observe":
		return runObserve(args[1:])
	case "verify":
		return runVerify(args[1:])
	case "extract":
		return runExtract(args[1:])
	case "version":
		fmt.Println(version)
		return nil
	case "help", "-h", "--help":
		usage()
		return nil
	default:
		usage()
		return fmt.Errorf("unknown subcommand %q", args[0])
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `tw — Typed Web Genesis CLI

Usage:
  tw observe --url URL --out DIR --cas DIR [--allow-loopback]
  tw verify --observation FILE --cas DIR
  tw extract --observation FILE --cas DIR --adapter FILE --out FILE
  tw version

Genesis is deliberately read-only. The --allow-loopback switch exists only for
controlled local fixtures and must not be used as a public-service policy.
`)
}

func runObserve(args []string) error {
	fs := flag.NewFlagSet("observe", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	rawURL := fs.String("url", "", "public URL to observe")
	outDir := fs.String("out", "var/run/latest", "bundle output directory")
	casDir := fs.String("cas", "var/cas", "content-addressed store directory")
	allowLoopback := fs.Bool("allow-loopback", false, "allow loopback and non-standard ports for controlled local fixtures")
	maxBody := fs.Int64("max-body", 2<<20, "maximum decompressed response bytes")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *rawURL == "" {
		return errors.New("observe: --url is required")
	}
	if err := validateBodyLimit(*maxBody); err != nil {
		return fmt.Errorf("observe: %w", err)
	}
	policy := safefetch.DefaultPolicy()
	policy.MaxBodyBytes = *maxBody
	if *allowLoopback {
		policy.ID = "tw.fetch.local-fixture-v0"
		policy.AllowLoopback = true
		policy.AllowNonStandardPorts = true
	}
	fetcher, err := safefetch.New(policy)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), policy.RequestTimeout+2*time.Second)
	defer cancel()
	result, err := fetcher.Fetch(ctx, *rawURL)
	if err != nil {
		return err
	}
	paths, err := observation.WriteBundle(*outDir, cas.New(*casDir), result, policy.ID)
	if err != nil {
		return err
	}
	return writeJSON(os.Stdout, map[string]any{
		"status":            "observed",
		"request_url":       result.RequestURL,
		"final_url":         result.FinalURL,
		"http_status":       result.Status,
		"media_type":        result.MediaType,
		"observation_cbor":  paths.CBORPath,
		"observation_json":  paths.JSONPath,
		"body_digest":       paths.BodyReference,
		"body_storage_path": paths.BodyStoragePath,
	})
}

func runVerify(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	observationPath := fs.String("observation", "", "path to observation.cbor")
	casDir := fs.String("cas", "var/cas", "content-addressed store directory")
	maxBody := fs.Int64("max-body", 2<<20, "maximum body bytes to verify")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *observationPath == "" {
		return errors.New("verify: --observation is required")
	}
	if err := validateBodyLimit(*maxBody); err != nil {
		return fmt.Errorf("verify: %w", err)
	}
	env, raw, err := observation.Load(*observationPath)
	if err != nil {
		return err
	}
	if err := observation.VerifyBody(env, cas.New(*casDir), *maxBody); err != nil {
		return err
	}
	return writeJSON(os.Stdout, map[string]any{
		"status":             "verified",
		"observation_digest": observation.EnvelopeDigest(raw),
		"body_digest":        env.BodyDigest(),
		"body_size":          env.BodySize,
		"request_url":        env.RequestURL,
		"final_url":          env.FinalURL,
	})
}

func runExtract(args []string) error {
	fs := flag.NewFlagSet("extract", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	observationPath := fs.String("observation", "", "path to observation.cbor")
	casDir := fs.String("cas", "var/cas", "content-addressed store directory")
	adapterPath := fs.String("adapter", "", "path to adapter manifest")
	outPath := fs.String("out", "var/run/latest/result.json", "typed result output path")
	maxBody := fs.Int64("max-body", 2<<20, "maximum body bytes to process")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *observationPath == "" || *adapterPath == "" {
		return errors.New("extract: --observation and --adapter are required")
	}
	if err := validateBodyLimit(*maxBody); err != nil {
		return fmt.Errorf("extract: %w", err)
	}
	env, raw, err := observation.Load(*observationPath)
	if err != nil {
		return err
	}
	store := cas.New(*casDir)
	if err := observation.VerifyBody(env, store, *maxBody); err != nil {
		return err
	}
	loaded, err := adapter.Load(*adapterPath)
	if err != nil {
		return err
	}
	result, err := adapter.Execute(env, raw, store, loaded, *maxBody)
	if err != nil {
		return err
	}
	encoded, err := adapter.MarshalResult(result)
	if err != nil {
		return err
	}
	if err := atomicfile.Write(*outPath, encoded, adapter.MaxResultBytes, 0o640); err != nil {
		return fmt.Errorf("extract: write result: %w", err)
	}
	return writeJSON(os.Stdout, map[string]any{
		"status":             "extracted",
		"result_path":        *outPath,
		"result_digest":      result.ResultDigest,
		"operation_id":       result.OperationID,
		"field_count":        len(result.Fields),
		"semantic_closure":   result.SemanticClosure,
		"observation_digest": result.ObservationDigest,
	})
}

func validateBodyLimit(limit int64) error {
	if limit <= 0 || limit > observation.MaxBodyBytes {
		return fmt.Errorf("--max-body must be between 1 and %d bytes", observation.MaxBodyBytes)
	}
	return nil
}

func writeJSON(file *os.File, value any) error {
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
