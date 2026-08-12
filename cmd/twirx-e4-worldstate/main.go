package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/typed-web-commons/typed-web/internal/worldstatepilot"
)

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("subcommand required: prepare, execute, compile, or verify-release")
	}
	switch args[0] {
	case "prepare":
		set := flag.NewFlagSet("prepare", flag.ContinueOnError)
		set.SetOutput(stderr)
		root := set.String("root", ".", "repository root")
		plan := set.String("plan", "atlas/e4-plans/world-bank-e2-matrix.json", "reviewed pilot plan")
		out := set.String("out", "", "new immutable prepared directory")
		if err := set.Parse(args[1:]); err != nil {
			return err
		}
		if *out == "" || set.NArg() != 0 {
			return errors.New("prepare requires --out and no positional arguments")
		}
		prepared, err := worldstatepilot.Prepare(*root, *plan, *out)
		if err != nil {
			return err
		}
		return writeJSON(stdout, prepared)
	case "execute":
		set := flag.NewFlagSet("execute", flag.ContinueOnError)
		set.SetOutput(stderr)
		root := set.String("root", ".", "repository root")
		plan := set.String("plan", "atlas/e4-plans/world-bank-e2-matrix.json", "reviewed pilot plan")
		prepared := set.String("prepared", "", "prepared order directory")
		spool := set.String("spool", "", "immutable spool root")
		state := set.String("state", "", "mutable circuit and lease state")
		if err := set.Parse(args[1:]); err != nil {
			return err
		}
		if *prepared == "" || *spool == "" || *state == "" || set.NArg() != 0 {
			return errors.New("execute requires --prepared, --spool, and --state")
		}
		summary, err := worldstatepilot.ExecutePrepared(ctx, *root, *plan, *prepared, *spool, *state, func() time.Time { return time.Now().UTC() }, time.Sleep)
		if err != nil {
			return err
		}
		return writeJSON(stdout, summary)
	case "compile":
		set := flag.NewFlagSet("compile", flag.ContinueOnError)
		set.SetOutput(stderr)
		root := set.String("root", ".", "repository root")
		prepared := set.String("prepared", "", "prepared and executed acquisition directory")
		spool := set.String("spool", "", "immutable spool root")
		out := set.String("out", "", "new immutable compiled release directory")
		compiledAt := set.String("compiled-at", "", "canonical UTC seconds")
		if err := set.Parse(args[1:]); err != nil {
			return err
		}
		if *prepared == "" || *spool == "" || *out == "" || *compiledAt == "" || set.NArg() != 0 {
			return errors.New("compile requires --prepared, --spool, --out, and --compiled-at")
		}
		release, err := worldstatepilot.Compile(*root, *prepared, *spool, *out, *compiledAt)
		if err != nil {
			return err
		}
		return writeJSON(stdout, release)
	case "verify-release":
		set := flag.NewFlagSet("verify-release", flag.ContinueOnError)
		set.SetOutput(stderr)
		root := set.String("root", ".", "repository root")
		releaseRoot := set.String("release", "", "manifest-last release directory")
		if err := set.Parse(args[1:]); err != nil {
			return err
		}
		if *releaseRoot == "" || set.NArg() != 0 {
			return errors.New("verify-release requires --release")
		}
		release, err := worldstatepilot.VerifyRelease(*root, *releaseRoot)
		if err != nil {
			return err
		}
		return writeJSON(stdout, release)
	default:
		return fmt.Errorf("unknown subcommand %q", args[0])
	}
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
