package main

import (
	"bufio"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/typed-web-commons/typed-web/internal/e4vectors"
)

func main() {
	out := flag.String("out", "conformance/e4-ontology/vectors.tsv", "output path")
	flag.Parse()
	vectors, err := e4vectors.Corpus()
	if err != nil {
		fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fatal(err)
	}
	file, err := os.Create(*out)
	if err != nil {
		fatal(err)
	}
	writer := bufio.NewWriter(file)
	if _, err = fmt.Fprintln(writer, "name\tkind\texpect\treason\thex"); err == nil {
		for _, vector := range vectors {
			expect := "reject"
			if vector.Valid {
				expect = "accept"
			}
			reason := strings.ReplaceAll(vector.Reason, "\t", " ")
			_, err = fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n", vector.Name, vector.Kind, expect, reason, hex.EncodeToString(vector.Data))
			if err != nil {
				break
			}
		}
	}
	if flushErr := writer.Flush(); err == nil {
		err = flushErr
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
