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
	"syscall"
	"time"

	"github.com/typed-web-commons/typed-web/internal/atlas"
	"github.com/typed-web-commons/typed-web/internal/atlasapi"
	"github.com/typed-web-commons/typed-web/internal/atlasstress"
	"github.com/typed-web-commons/typed-web/internal/atomicfile"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("subcommand required: validate, metrics, plan, stress, or serve")
	}
	switch args[0] {
	case "validate":
		return runValidate(args[1:], stdout, stderr)
	case "metrics":
		return runMetrics(args[1:], stdout, stderr)
	case "plan":
		return runPlan(args[1:], stdout, stderr)
	case "stress":
		return runStress(ctx, args[1:], stdout, stderr)
	case "serve":
		return runServe(ctx, args[1:], stderr)
	default:
		return fmt.Errorf("unknown subcommand %q", args[0])
	}
}

func runStress(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("stress", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".", "repository root")
	atText := fs.String("at", "", "required canonical UTC planning time")
	rounds := fs.Int("rounds", 10, "complete 500-origin lookup rounds")
	workers := fs.Int("workers", 32, "bounded concurrent workers")
	base := fs.String("base", "", "optional separately running literal-loopback Atlas HTTP origin")
	out := fs.String("out", "", "optional atomic JSON report path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	at, err := parseCanonicalTime(*atText, "stress")
	if err != nil {
		return err
	}
	selection, policies, registry, err := artifacts(*root)
	if err != nil {
		return err
	}
	plan, err := atlas.BuildDryRunFrontier(selection, registry, policies, at)
	if err != nil {
		return err
	}
	handler, err := atlasapi.New(selection, registry, policies)
	if err != nil {
		return err
	}
	config := atlasstress.Config{Rounds: *rounds, Workers: *workers}
	var report atlasstress.Report
	if *base == "" {
		report, err = atlasstress.Run(ctx, handler, selection, plan, config)
	} else {
		report, err = atlasstress.RunLoopback(ctx, *base, selection, plan, config)
	}
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if *out != "" {
		path := *out
		if !filepath.IsAbs(path) {
			path = filepath.Join(*root, path)
		}
		if err := atomicfile.Write(path, data, 1<<20, 0o640); err != nil {
			return err
		}
	}
	_, err = stdout.Write(data)
	return err
}

func artifacts(root string) (*atlas.Selection, *atlas.PolicySet, *atlas.Registry, error) {
	selection, err := atlas.LoadSelection(filepath.Join(root, "atlas", "genesis-500", "selection.json"))
	if err != nil {
		return nil, nil, nil, err
	}
	policies, err := atlas.LoadPolicySet(filepath.Join(root, "atlas", "policies.json"), selection)
	if err != nil {
		return nil, nil, nil, err
	}
	registry, err := atlas.LoadRegistry(filepath.Join(root, "atlas", "registry.json"), selection, policies)
	if err != nil {
		return nil, nil, nil, err
	}
	return selection, policies, registry, nil
}

func runValidate(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".", "repository root")
	if err := fs.Parse(args); err != nil {
		return err
	}
	selection, policies, registry, err := artifacts(*root)
	if err != nil {
		return err
	}
	metrics, err := atlas.BuildMetrics(selection, registry, policies)
	if err != nil {
		return err
	}
	return writeJSON(stdout, map[string]any{
		"status": "valid", "selection_digest": selection.DigestReference(), "policy_set_digest": policies.DigestReference(),
		"selected_candidates": metrics.Atlas.SelectedCandidates, "policy_records": metrics.Atlas.PolicyRecords,
		"cataloged":        metrics.Atlas.CatalogState[string(atlas.CatalogCataloged)],
		"policy_completed": metrics.Atlas.PolicyReviewState[string(atlas.PolicyCompleted)], "network_access": "disabled",
	})
}

func runMetrics(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("metrics", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".", "repository root")
	out := fs.String("out", "", "optional generated metrics path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	selection, policies, registry, err := artifacts(*root)
	if err != nil {
		return err
	}
	metrics, err := atlas.BuildMetrics(selection, registry, policies)
	if err != nil {
		return err
	}
	if *out != "" {
		if err := atlas.WriteMetrics(*out, metrics); err != nil {
			return err
		}
		return writeJSON(stdout, map[string]any{"status": "generated", "path": *out, "selection_digest": metrics.SelectionDigest})
	}
	data, err := atlas.MarshalMetrics(metrics)
	if err != nil {
		return err
	}
	_, err = stdout.Write(data)
	return err
}

func runPlan(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("plan", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".", "repository root")
	atText := fs.String("at", "", "required canonical UTC planning time")
	if err := fs.Parse(args); err != nil {
		return err
	}
	at, err := parseCanonicalTime(*atText, "plan")
	if err != nil {
		return err
	}
	selection, policies, registry, err := artifacts(*root)
	if err != nil {
		return err
	}
	plan, err := atlas.BuildDryRunFrontier(selection, registry, policies, at)
	if err != nil {
		return err
	}
	return writeJSON(stdout, plan)
}

func parseCanonicalTime(value, command string) (time.Time, error) {
	if value == "" {
		return time.Time{}, fmt.Errorf("%s: --at is required for deterministic output", command)
	}
	at, err := time.Parse(time.RFC3339Nano, value)
	if err != nil || at.Location() != time.UTC || at.Format(time.RFC3339Nano) != value {
		return time.Time{}, fmt.Errorf("%s: --at must be canonical UTC RFC3339", command)
	}
	return at, nil
}

func runServe(ctx context.Context, args []string, stderr io.Writer) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".", "repository root")
	listenAddress := fs.String("listen", "127.0.0.1:8092", "loopback listen address")
	if err := fs.Parse(args); err != nil {
		return err
	}
	host, _, err := net.SplitHostPort(*listenAddress)
	if err != nil {
		return fmt.Errorf("serve: invalid listen address: %w", err)
	}
	address, err := netip.ParseAddr(host)
	if err != nil || !address.IsLoopback() {
		return errors.New("serve: --listen must use a literal loopback address")
	}
	selection, policies, registry, err := artifacts(*root)
	if err != nil {
		return err
	}
	handler, err := atlasapi.New(selection, registry, policies)
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", *listenAddress)
	if err != nil {
		return fmt.Errorf("serve: listen: %w", err)
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	stopped := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = server.Shutdown(shutdown)
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
