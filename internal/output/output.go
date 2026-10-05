// Package output writes a report as JSON, YAML or a human-readable summary.
package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/jiegui2025/hwspec/internal/report"
)

var Formats = []string{"json", "yaml", "text"}

// FormatFromPath guesses the format from a file extension ("" if unknown).
func FormatFromPath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".txt":
		return "text"
	}
	return ""
}

func Write(w io.Writer, r *report.Report, format string) error {
	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(r)
	case "yaml":
		return writeYAML(w, r)
	case "text":
		return writeText(w, r)
	}
	return fmt.Errorf("unknown format %q (want one of %s)", format, strings.Join(Formats, ", "))
}

// writeYAML goes through JSON so the YAML uses the same field names and
// order as the JSON (JSON is valid YAML, and yaml.Node keeps key order).
func writeYAML(w io.Writer, r *report.Report) error {
	js, err := json.Marshal(r)
	if err != nil {
		return err
	}
	var node yaml.Node
	if err := yaml.Unmarshal(js, &node); err != nil {
		return err
	}
	blockStyle(&node)
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&node); err != nil {
		return err
	}
	_, err = w.Write(buf.Bytes())
	return err
}

// blockStyle drops the flow ({...}, "...") styling that parsing JSON leaves
// on every node. The encoder still quotes strings that would otherwise be
// read back as numbers or booleans.
func blockStyle(n *yaml.Node) {
	n.Style = 0
	for _, c := range n.Content {
		blockStyle(c)
	}
	// Keep empty collections readable as [] / {}.
	if (n.Kind == yaml.SequenceNode || n.Kind == yaml.MappingNode) && len(n.Content) == 0 {
		n.Style = yaml.FlowStyle
	}
}

// ErrNotCapture is returned for valid JSON/YAML that isn't a hwspec capture.
var ErrNotCapture = errors.New("not a hwspec capture (no tool.name \"hwspec\" and schema_version)")

// Read loads a report previously written as JSON or YAML.
func Read(data []byte) (*report.Report, error) {
	r, err := decode(data)
	if err != nil {
		return nil, err
	}
	if r.Tool.Name != "hwspec" || r.SchemaVersion < 1 {
		return nil, ErrNotCapture
	}
	return r, nil
}

func decode(data []byte) (*report.Report, error) {
	var r report.Report
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) > 0 && trimmed[0] == '{' {
		if err := json.Unmarshal(trimmed, &r); err != nil {
			return nil, err
		}
		return &r, nil
	}
	// YAML: decode generically, then reuse the JSON field names.
	var v any
	if err := yaml.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	js, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(js, &r); err != nil {
		return nil, err
	}
	return &r, nil
}
