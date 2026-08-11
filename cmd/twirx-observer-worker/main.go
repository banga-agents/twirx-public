package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/typed-web-commons/typed-web/internal/observatoryworker"
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
		return errors.New("subcommand required: fetch or verify")
	}
	switch args[0] {
	case "fetch":
		fs := flag.NewFlagSet("fetch", flag.ContinueOnError)
		fs.SetOutput(stderr)
		jobPath := fs.String("job", "", "validated local-fixture job JSON")
		out := fs.String("out", "", "new or empty evidence output directory")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *jobPath == "" || *out == "" {
			return errors.New("fetch: --job and --out are required")
		}
		job, err := observatoryworker.LoadJob(*jobPath)
		if err != nil {
			return err
		}
		result, err := observatoryworker.Execute(ctx, job, *out)
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	case "verify":
		fs := flag.NewFlagSet("verify", flag.ContinueOnError)
		fs.SetOutput(stderr)
		out := fs.String("out", "", "existing evidence output directory")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *out == "" {
			return errors.New("verify: --out is required")
		}
		result, err := observatoryworker.Verify(*out)
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	default:
		return fmt.Errorf("unknown subcommand %q", args[0])
	}
}

func writeJSON(output io.Writer, value any) error {
	data, err := observatoryworker.MarshalJSON(value)
	if err != nil {
		return err
	}
	_, err = output.Write(data)
	return err
}
