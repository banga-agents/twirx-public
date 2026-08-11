package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/typed-web-commons/typed-web/internal/atomicfile"
	"github.com/typed-web-commons/typed-web/internal/labstress"
)

const maxReportBytes = 2 << 20

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("twirx-stress", flag.ContinueOnError)
	fs.SetOutput(stderr)
	base := fs.String("base", "http://127.0.0.1:8090", "literal-loopback Lab base URL or https://lab.twirx.org")
	workloadPath := fs.String("workload", "stress/e2-replay-workload.json", "bounded replay workload")
	requests := fs.Int("requests", 100, "total invocations (1..1000)")
	concurrency := fs.Int("concurrency", 8, "concurrent invocations (1..64)")
	simulatedClients := fs.Int("simulated-clients", 100, "loopback-only simulated clients (0..1000)")
	proofSamples := fs.Int("proof-samples", 32, "maximum distinct proof bundles to validate (1..32)")
	timeout := fs.Duration("timeout", 30*time.Second, "per-request timeout (1s..60s)")
	out := fs.String("out", "", "optional atomic JSON report path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("positional arguments are not accepted")
	}
	workload, err := labstress.LoadWorkload(*workloadPath)
	if err != nil {
		return err
	}
	report, err := labstress.Run(ctx, labstress.Config{
		BaseURL:          *base,
		Requests:         *requests,
		Concurrency:      *concurrency,
		SimulatedClients: *simulatedClients,
		ProofSamples:     *proofSamples,
		Timeout:          *timeout,
		Workload:         workload,
	})
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("encode report: %w", err)
	}
	encoded = append(encoded, '\n')
	if len(encoded) > maxReportBytes {
		return errors.New("encoded report exceeds limit")
	}
	if _, err := stdout.Write(encoded); err != nil {
		return err
	}
	if *out != "" {
		if err := atomicfile.Write(*out, encoded, maxReportBytes, 0o640); err != nil {
			return err
		}
	}
	if !report.Pass {
		return errors.New("stress invariants failed; inspect the emitted report")
	}
	return nil
}
