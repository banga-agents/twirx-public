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

	"github.com/typed-web-commons/typed-web/internal/egressworker"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("subcommand required: execute or verify")
	}
	switch args[0] {
	case "execute":
		fs := flag.NewFlagSet("execute", flag.ContinueOnError)
		fs.SetOutput(stderr)
		id := fs.String("id", "", "sealed work-order ID (never a URL)")
		orders := fs.String("work-orders", "/var/lib/twirx-egress/work-orders", "read-only sealed work-order directory")
		controlPath := fs.String("control", "/var/lib/twirx-egress/control.json", "read-only kill-switch and revocation artifact")
		spool := fs.String("spool", "/var/lib/twirx-egress/spool", "immutable evidence spool")
		state := fs.String("state", "/var/lib/twirx-egress/state", "bounded worker state directory")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *id == "" || fs.NArg() != 0 {
			return errors.New("execute requires exactly --id; caller-supplied URLs are not accepted")
		}
		order, err := egressworker.LoadWorkOrder(*orders, *id)
		if err != nil {
			return err
		}
		control, err := egressworker.LoadControl(*controlPath)
		if err != nil {
			return err
		}
		result, err := egressworker.Execute(context.Background(), order, control, *spool, *state, time.Now().UTC())
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	case "verify":
		fs := flag.NewFlagSet("verify", flag.ContinueOnError)
		fs.SetOutput(stderr)
		spool := fs.String("spool", "", "completed individual spool directory")
		maxBody := fs.Int64("max-body-bytes", egressworker.MaxBody, "maximum admitted representation size")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *spool == "" || fs.NArg() != 0 {
			return errors.New("verify requires exactly --spool")
		}
		result, err := egressworker.VerifySpool(*spool, *maxBody)
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	default:
		return fmt.Errorf("unknown subcommand %q", args[0])
	}
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
