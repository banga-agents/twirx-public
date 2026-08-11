// twirx-archive-acquire is an operator-only restricted network helper. It can
// contact only the official Common Crawl index and data hosts under a sealed,
// human-approved work order. It is not linked into the public runtime.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/typed-web-commons/typed-web/internal/archiveacquire"
	"github.com/typed-web-commons/typed-web/internal/archiveimport"
)

const usage = `usage:
  twirx-archive-acquire run --work-orders DIR --id ID --out NEW_DIR
  twirx-archive-acquire verify --root DIR

run derives every network request from one completed human policy decision and
sealed work order. No URL, host, range, collection or redirect is accepted from
the command line.`

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(arguments []string, stdout, stderr io.Writer) error {
	if len(arguments) == 0 {
		return errors.New(usage)
	}
	switch arguments[0] {
	case "run":
		flags := flag.NewFlagSet("run", flag.ContinueOnError)
		flags.SetOutput(stderr)
		orders := flags.String("work-orders", "", "sealed work-order directory")
		id := flags.String("id", "", "sealed work-order identity")
		output := flags.String("out", "", "new immutable acquisition directory")
		if err := flags.Parse(arguments[1:]); err != nil {
			return err
		}
		if *orders == "" || *id == "" || *output == "" || flags.NArg() != 0 {
			return errors.New("twirx-archive-acquire: run requires --work-orders, --id, and --out")
		}
		loaded, err := archiveimport.LoadWorkOrder(*orders, *id)
		if err != nil {
			return err
		}
		runner, err := archiveacquire.NewRunner()
		if err != nil {
			return err
		}
		manifest, err := runner.Acquire(context.Background(), loaded, *output)
		if err != nil {
			return err
		}
		return writeJSON(stdout, manifest)
	case "verify":
		flags := flag.NewFlagSet("verify", flag.ContinueOnError)
		flags.SetOutput(stderr)
		root := flags.String("root", "", "immutable acquisition directory")
		if err := flags.Parse(arguments[1:]); err != nil {
			return err
		}
		if *root == "" || flags.NArg() != 0 {
			return errors.New("twirx-archive-acquire: verify requires --root")
		}
		manifest, err := archiveacquire.Verify(*root)
		if err != nil {
			return err
		}
		return writeJSON(stdout, manifest)
	default:
		return errors.New(usage)
	}
}

func writeJSON(output io.Writer, value any) error {
	encoder := json.NewEncoder(output)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
