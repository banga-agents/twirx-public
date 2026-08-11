package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/typed-web-commons/typed-web/internal/admission"
	"github.com/typed-web-commons/typed-web/internal/atlas"
	"github.com/typed-web-commons/typed-web/internal/atomicfile"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("subcommand required: validate, atlas-queue, review-queue, render, or check-canonical")
	}
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".", "repository root")
	admissions := fs.String("admissions", "atlas/admissions", "per-origin admission directory")
	fixtures := fs.String("fixtures", "atlas/fixtures", "controlled test-fixture source directory")
	out := fs.String("out", "generated/e3/admission", "generated dossier directory")
	version := fs.String("version", "2026-08-11", "artifact version")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	selection, err := atlas.LoadSelection(filepath.Join(*root, "atlas", "genesis-500", "selection.json"))
	if err != nil {
		return err
	}
	sources, err := admission.Load(filepath.Join(*root, *admissions), selection)
	if err != nil {
		return err
	}
	fixtureRecords, err := admission.LoadFixtures(filepath.Join(*root, *fixtures))
	if err != nil {
		return err
	}
	_, policyBytes, registry, registryBytes, batch, err := admission.Render(sources, fixtureRecords, selection, *version)
	if err != nil {
		return err
	}
	queue, err := admission.BuildWorkQueue(selection, sources, *version)
	if err != nil {
		return err
	}
	if err := registry.ValidateArtifactFiles(*root); err != nil {
		return err
	}
	switch args[0] {
	case "validate":
		return writeJSON(stdout, batch)
	case "atlas-queue":
		return writeJSON(stdout, queue)
	case "review-queue":
		queue := make([]admission.Dossier, 0, batch.PendingHumanReview)
		for _, dossier := range batch.OriginDossiers {
			if dossier.AdmissionReview == admission.ReviewPending {
				queue = append(queue, dossier)
			}
		}
		return writeJSON(stdout, map[string]any{"format": "tw.admission-review-queue/0.1", "pending": len(queue), "origins": queue})
	case "render":
		return render(filepath.Join(*root, *out), sources, policyBytes, registryBytes, batch, queue, stdout)
	case "check-canonical":
		for path, expected := range map[string][]byte{
			filepath.Join(*root, "atlas", "policies.json"): policyBytes,
			filepath.Join(*root, "atlas", "registry.json"): registryBytes,
		} {
			actual, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			if !bytes.Equal(actual, expected) {
				return fmt.Errorf("canonical artifact %s differs from per-origin sources", path)
			}
		}
		return writeJSON(stdout, map[string]any{"status": "canonical", "origins": batch.CanonicalAdmissions})
	default:
		return fmt.Errorf("unknown subcommand %q", args[0])
	}
}

func render(out string, sources []admission.Source, policies, registry []byte, batch admission.Batch, queue admission.WorkQueue, stdout io.Writer) error {
	if info, err := os.Lstat(out); err == nil && !info.IsDir() {
		return errors.New("admission output must be a directory, never a symlink or file")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Join(out, "dossiers"), 0o750); err != nil {
		return err
	}
	if err := rejectUnexpectedOutput(out, batch.OriginDossiers); err != nil {
		return err
	}
	if err := atomicfile.Write(filepath.Join(out, "policies.json"), policies, 8<<20, 0o640); err != nil {
		return err
	}
	if err := atomicfile.Write(filepath.Join(out, "registry.json"), registry, 8<<20, 0o640); err != nil {
		return err
	}
	batchBytes, err := admission.Marshal(batch)
	if err != nil {
		return err
	}
	if err := atomicfile.Write(filepath.Join(out, "batch.json"), batchBytes, 8<<20, 0o640); err != nil {
		return err
	}
	queueBytes, err := admission.Marshal(queue)
	if err != nil {
		return err
	}
	if err := atomicfile.Write(filepath.Join(out, "atlas-queue.json"), queueBytes, 8<<20, 0o640); err != nil {
		return err
	}
	for index, dossier := range batch.OriginDossiers {
		data, marshalErr := admission.Marshal(dossier)
		if marshalErr != nil {
			return marshalErr
		}
		if err := atomicfile.Write(filepath.Join(out, "dossiers", dossier.OriginID+".json"), data, admission.MaxArtifact, 0o640); err != nil {
			return err
		}
		if index >= len(sources) {
			return errors.New("rendered dossier count exceeds source count")
		}
	}
	return writeJSON(stdout, map[string]any{"status": "rendered", "canonical_admissions": batch.CanonicalAdmissions, "dossiers": batch.Dossiers, "output": out})
}

func rejectUnexpectedOutput(out string, dossiers []admission.Dossier) error {
	allowedRoot := map[string]struct{}{"atlas-queue.json": {}, "batch.json": {}, "dossiers": {}, "policies.json": {}, "registry.json": {}}
	entries, err := os.ReadDir(out)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if _, ok := allowedRoot[entry.Name()]; !ok {
			return fmt.Errorf("unexpected generated admission artifact %s", entry.Name())
		}
	}
	allowedDossiers := make(map[string]struct{}, len(dossiers))
	for _, dossier := range dossiers {
		allowedDossiers[dossier.OriginID+".json"] = struct{}{}
	}
	entries, err = os.ReadDir(filepath.Join(out, "dossiers"))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return fmt.Errorf("unexpected generated admission directory %s", entry.Name())
		}
		if _, ok := allowedDossiers[entry.Name()]; !ok {
			return fmt.Errorf("stale generated admission dossier %s", entry.Name())
		}
	}
	return nil
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
