// twirx-archive is an offline admission tool for sealed Common Crawl
// acquisition artifacts. It deliberately contains no HTTP client.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/typed-web-commons/typed-web/internal/archiveimport"
)

const usage = `usage:
  twirx-archive plan --work-orders DIR --id ID
  twirx-archive inspect-index --work-orders DIR --id ID --collection ID --route URL --response FILE
  twirx-archive import --work-orders DIR --id ID --collection ID --route URL --response FILE --capture N --warc FILE --http-status 206 --content-range VALUE --out DIR
  twirx-archive verify --spool DIR

This command never performs network access. plan prints the only official-host
requests permitted by a sealed, human-approved work order. import accepts only
bounded regular files acquired separately under that authority.`

type planEntry struct {
	CollectionID string `json:"collection_id"`
	Route        string `json:"route"`
	IndexURL     string `json:"index_url"`
}

type planOutput struct {
	Format              string      `json:"format"`
	WorkOrderID         string      `json:"work_order_id"`
	WorkOrderDigest     string      `json:"work_order_digest"`
	OriginID            string      `json:"origin_id"`
	NetworkRequestsMade uint64      `json:"network_requests_made"`
	Entries             []planEntry `json:"entries"`
}

type indexOutput struct {
	Format              string                  `json:"format"`
	WorkOrderID         string                  `json:"work_order_id"`
	WorkOrderDigest     string                  `json:"work_order_digest"`
	NetworkRequestsMade uint64                  `json:"network_requests_made"`
	Captures            []archiveimport.Capture `json:"captures"`
}

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
	case "plan":
		return runPlan(arguments[1:], stdout, stderr)
	case "inspect-index":
		return runInspect(arguments[1:], stdout, stderr)
	case "import":
		return runImport(arguments[1:], stdout, stderr)
	case "verify":
		return runVerify(arguments[1:], stdout, stderr)
	default:
		return errors.New(usage)
	}
}

func workOrderFlags(name string, arguments []string, stderr io.Writer) (*archiveimport.LoadedWorkOrder, *flag.FlagSet, error) {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("work-orders", "", "sealed work-order directory")
	id := flags.String("id", "", "sealed work-order ID")
	if err := flags.Parse(arguments); err != nil {
		return nil, flags, err
	}
	if *root == "" || *id == "" || flags.NArg() != 0 {
		return nil, flags, errors.New("twirx-archive: --work-orders and --id are required")
	}
	loaded, err := archiveimport.LoadWorkOrder(*root, *id)
	if err != nil {
		return nil, flags, err
	}
	return loaded, flags, nil
}

func runPlan(arguments []string, stdout, stderr io.Writer) error {
	loaded, _, err := workOrderFlags("plan", arguments, stderr)
	if err != nil {
		return err
	}
	output := planOutput{Format: "tw.archive-acquisition-plan/0.1", WorkOrderID: loaded.Order.ID, WorkOrderDigest: loaded.Digest, OriginID: loaded.Order.OriginID, Entries: make([]planEntry, 0, len(loaded.Order.CollectionIDs)*len(loaded.Order.PermittedRoutes))}
	for _, collection := range loaded.Order.CollectionIDs {
		for _, route := range loaded.Order.PermittedRoutes {
			indexURL, buildErr := archiveimport.BuildIndexURL(loaded.Order, collection, route)
			if buildErr != nil {
				return buildErr
			}
			output.Entries = append(output.Entries, planEntry{CollectionID: collection, Route: route, IndexURL: indexURL})
		}
	}
	return writeJSON(stdout, output)
}

func runInspect(arguments []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("inspect-index", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("work-orders", "", "sealed work-order directory")
	id := flags.String("id", "", "sealed work-order ID")
	collection := flags.String("collection", "", "sealed Common Crawl collection")
	route := flags.String("route", "", "exact sealed origin route")
	response := flags.String("response", "", "bounded index-response file")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if *root == "" || *id == "" || *collection == "" || *route == "" || *response == "" || flags.NArg() != 0 {
		return errors.New("twirx-archive: inspect-index requires --work-orders, --id, --collection, --route, and --response")
	}
	loaded, err := archiveimport.LoadWorkOrder(*root, *id)
	if err != nil {
		return err
	}
	captures, err := archiveimport.LoadIndexResponse(*response, loaded.Order, *collection, *route)
	if err != nil {
		return err
	}
	return writeJSON(stdout, indexOutput{Format: "tw.archive-index-inspection/0.1", WorkOrderID: loaded.Order.ID, WorkOrderDigest: loaded.Digest, Captures: captures})
}

func runImport(arguments []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("import", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("work-orders", "", "sealed work-order directory")
	id := flags.String("id", "", "sealed work-order ID")
	collection := flags.String("collection", "", "sealed Common Crawl collection")
	route := flags.String("route", "", "exact sealed origin route")
	response := flags.String("response", "", "bounded index-response file")
	captureIndex := flags.Uint64("capture", ^uint64(0), "zero-based capture index from inspect-index")
	warc := flags.String("warc", "", "bounded WARC range-response body")
	httpStatus := flags.Int("http-status", 0, "range-response HTTP status")
	contentRange := flags.String("content-range", "", "exact range-response Content-Range value")
	output := flags.String("out", "", "new immutable evidence-spool directory")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if *root == "" || *id == "" || *collection == "" || *route == "" || *response == "" || *captureIndex == ^uint64(0) || *warc == "" || *contentRange == "" || *output == "" || flags.NArg() != 0 {
		return errors.New("twirx-archive: import requires all sealed index, range-response, and output flags")
	}
	loaded, err := archiveimport.LoadWorkOrder(*root, *id)
	if err != nil {
		return err
	}
	captures, err := archiveimport.LoadIndexResponse(*response, loaded.Order, *collection, *route)
	if err != nil {
		return err
	}
	if *captureIndex >= uint64(len(captures)) {
		return errors.New("twirx-archive: capture index is outside the inspected response")
	}
	evidence, err := archiveimport.PublishCaptureFile(*output, loaded, captures[*captureIndex], *httpStatus, *contentRange, *warc)
	if err != nil {
		return err
	}
	return writeJSON(stdout, evidence)
}

func runVerify(arguments []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("verify", flag.ContinueOnError)
	flags.SetOutput(stderr)
	spool := flags.String("spool", "", "immutable evidence-spool directory")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if *spool == "" || flags.NArg() != 0 {
		return errors.New("twirx-archive: verify requires --spool")
	}
	evidence, err := archiveimport.VerifySpool(*spool)
	if err != nil {
		return err
	}
	return writeJSON(stdout, evidence)
}

func writeJSON(output io.Writer, value any) error {
	encoder := json.NewEncoder(output)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
