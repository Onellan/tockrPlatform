// Command platform-reconcile reads normalized fixture/adapter inventory JSON
// and emits a deterministic, read-only Platform mapping proposal report.
// It never connects to CTRL, IMS or the Platform database.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/Onellan/tockrplatform/internal/platform/reconciliation"
)

func main() {
	inputPath := flag.String("input", "", "normalized inventory JSON path; stdin when omitted")
	outputPath := flag.String("output", "", "report JSON path; stdout when omitted")
	flag.Parse()

	input, err := readInput(*inputPath)
	if err != nil {
		fatal(err)
	}
	defer input.Close()
	var inventory reconciliation.Inventory
	decoder := json.NewDecoder(input)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&inventory); err != nil {
		fatal(fmt.Errorf("decode inventory: %w", err))
	}
	report, err := reconciliation.MarshalReport(reconciliation.Build(inventory))
	if err != nil {
		fatal(fmt.Errorf("encode reconciliation report: %w", err))
	}
	if err := writeOutput(*outputPath, report); err != nil {
		fatal(err)
	}
}

func readInput(path string) (io.ReadCloser, error) {
	if path == "" {
		return io.NopCloser(os.Stdin), nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open inventory: %w", err)
	}
	return file, nil
}

func writeOutput(path string, data []byte) error {
	if path == "" {
		_, err := os.Stdout.Write(data)
		return err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	return nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
