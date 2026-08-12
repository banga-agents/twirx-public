// Command twirx-e4-opportunity executes and verifies one founder-approved,
// manual-once Grants.gov bulk acquisition and its contact-free projection.
// It accepts no URL argument and has no scheduler, browser, model, action, or
// semantic-canon authority.
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

	"github.com/typed-web-commons/typed-web/internal/opportunitypilot"
	"github.com/typed-web-commons/typed-web/internal/opportunityrelease"
)

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, arguments []string, stdout, stderr io.Writer) error {
	if len(arguments) == 0 {
		return errors.New("subcommand required: acquire, verify-acquisition, project, verify-projection, build-release, verify-release, or export-c-sample")
	}
	switch arguments[0] {
	case "acquire":
		flags := flag.NewFlagSet("acquire", flag.ContinueOnError)
		flags.SetOutput(stderr)
		orderPath := flags.String("order", "", "exact founder-approved work order")
		root := flags.String("root", ".", "repository root containing exact policy authority")
		controlPath := flags.String("control", "", "manual kill-switch and revocation control")
		output := flags.String("out", "", "new immutable private acquisition directory")
		state := flags.String("state", "", "existing private one-shot execution-state directory")
		if err := flags.Parse(arguments[1:]); err != nil {
			return err
		}
		if *orderPath == "" || *controlPath == "" || *output == "" || *state == "" || flags.NArg() != 0 {
			return errors.New("acquire requires --order, --control, --out, --state, and no positional arguments")
		}
		loaded, err := opportunitypilot.LoadWorkOrder(*orderPath)
		if err != nil {
			return err
		}
		if err := opportunitypilot.VerifyAuthority(*root, loaded); err != nil {
			return err
		}
		control, err := opportunitypilot.LoadControl(*controlPath)
		if err != nil {
			return err
		}
		manifest, err := opportunitypilot.Acquire(ctx, loaded, control, *output, *state, func() time.Time { return time.Now().UTC() }, time.Sleep)
		if err != nil {
			return err
		}
		return writeJSON(stdout, manifest)
	case "verify-acquisition":
		flags := flag.NewFlagSet("verify-acquisition", flag.ContinueOnError)
		flags.SetOutput(stderr)
		orderPath := flags.String("order", "", "exact founder-approved work order")
		root := flags.String("root", ".", "repository root containing exact policy authority")
		acquisition := flags.String("acquisition", "", "manifest-complete private acquisition directory")
		if err := flags.Parse(arguments[1:]); err != nil {
			return err
		}
		if *orderPath == "" || *acquisition == "" || flags.NArg() != 0 {
			return errors.New("verify-acquisition requires --order, --acquisition, and no positional arguments")
		}
		loaded, err := opportunitypilot.LoadWorkOrder(*orderPath)
		if err != nil {
			return err
		}
		if err := opportunitypilot.VerifyAuthority(*root, loaded); err != nil {
			return err
		}
		manifest, err := opportunitypilot.VerifyAcquisition(*acquisition, loaded)
		if err != nil {
			return err
		}
		return writeJSON(stdout, manifest)
	case "project":
		flags := flag.NewFlagSet("project", flag.ContinueOnError)
		flags.SetOutput(stderr)
		orderPath := flags.String("order", "", "exact founder-approved work order")
		root := flags.String("root", ".", "repository root containing exact policy authority")
		acquisition := flags.String("acquisition", "", "manifest-complete private acquisition directory")
		output := flags.String("out", "", "new contact-free projection directory")
		if err := flags.Parse(arguments[1:]); err != nil {
			return err
		}
		if *orderPath == "" || *acquisition == "" || *output == "" || flags.NArg() != 0 {
			return errors.New("project requires --order, --acquisition, --out, and no positional arguments")
		}
		loaded, err := opportunitypilot.LoadWorkOrder(*orderPath)
		if err != nil {
			return err
		}
		if err := opportunitypilot.VerifyAuthority(*root, loaded); err != nil {
			return err
		}
		manifest, err := opportunitypilot.ProjectAcquisition(*acquisition, loaded, *output)
		if err != nil {
			return err
		}
		return writeJSON(stdout, manifest)
	case "verify-projection":
		flags := flag.NewFlagSet("verify-projection", flag.ContinueOnError)
		flags.SetOutput(stderr)
		orderPath := flags.String("order", "", "exact founder-approved work order")
		root := flags.String("root", ".", "repository root containing exact policy authority")
		acquisition := flags.String("acquisition", "", "manifest-complete private acquisition directory")
		projection := flags.String("projection", "", "manifest-complete approved projection directory")
		if err := flags.Parse(arguments[1:]); err != nil {
			return err
		}
		if *orderPath == "" || *acquisition == "" || *projection == "" || flags.NArg() != 0 {
			return errors.New("verify-projection requires --order, --acquisition, --projection, and no positional arguments")
		}
		loaded, err := opportunitypilot.LoadWorkOrder(*orderPath)
		if err != nil {
			return err
		}
		if err := opportunitypilot.VerifyAuthority(*root, loaded); err != nil {
			return err
		}
		manifest, _, err := opportunitypilot.VerifyProjection(*projection, *acquisition, loaded)
		if err != nil {
			return err
		}
		return writeJSON(stdout, manifest)
	case "build-release", "verify-release":
		flags := flag.NewFlagSet(arguments[0], flag.ContinueOnError)
		flags.SetOutput(stderr)
		root := flags.String("root", ".", "repository root containing exact policy authority")
		acquisition := flags.String("acquisition", "", "manifest-complete private acquisition directory")
		projection := flags.String("projection", "", "manifest-complete approved projection directory")
		worldRelease := flags.String("world-release", "", "verified E4.2 World State release")
		release := flags.String("release", "", "new or existing Opportunity release directory")
		if err := flags.Parse(arguments[1:]); err != nil {
			return err
		}
		if *acquisition == "" || *projection == "" || *worldRelease == "" || *release == "" || flags.NArg() != 0 {
			return fmt.Errorf("%s requires --acquisition, --projection, --world-release, --release, and no positional arguments", arguments[0])
		}
		if arguments[0] == "build-release" {
			manifest, err := opportunityrelease.Build(*root, *acquisition, *projection, *worldRelease, *release)
			if err != nil {
				return err
			}
			return writeJSON(stdout, manifest)
		}
		manifest, err := opportunityrelease.Verify(*root, *acquisition, *projection, *worldRelease, *release)
		if err != nil {
			return err
		}
		return writeJSON(stdout, manifest)
	case "export-c-sample":
		flags := flag.NewFlagSet("export-c-sample", flag.ContinueOnError)
		flags.SetOutput(stderr)
		release := flags.String("release", "", "verified immutable Opportunity release")
		releaseManifestDigest := flags.String("release-manifest-digest", "", "trusted SHA-256 identity of the release manifest")
		output := flags.String("out", "", "new immutable restricted-C sample directory")
		if err := flags.Parse(arguments[1:]); err != nil {
			return err
		}
		if *release == "" || *releaseManifestDigest == "" || *output == "" || flags.NArg() != 0 {
			return errors.New("export-c-sample requires --release, --release-manifest-digest, --out, and no positional arguments")
		}
		manifest, err := opportunityrelease.ExportVerifierSample(*release, *releaseManifestDigest, *output)
		if err != nil {
			return err
		}
		return writeJSON(stdout, manifest)
	default:
		return fmt.Errorf("unknown subcommand %q", arguments[0])
	}
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
