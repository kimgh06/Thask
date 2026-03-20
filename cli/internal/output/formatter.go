package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

type Format string

const (
	FormatJSON  Format = "json"
	FormatTable Format = "table"
	FormatQuiet Format = "quiet"
)

func ParseFormat(s string, pretty, quiet bool) Format {
	if quiet {
		return FormatQuiet
	}
	if pretty {
		return FormatTable
	}
	switch Format(s) {
	case FormatTable:
		return FormatTable
	case FormatQuiet:
		return FormatQuiet
	default:
		return FormatJSON
	}
}

// JSON prints raw JSON data to stdout
func JSON(data json.RawMessage) {
	var buf map[string]any
	if err := json.Unmarshal(data, &buf); err == nil {
		out, _ := json.MarshalIndent(buf, "", "  ")
		fmt.Println(string(out))
		return
	}
	var arr []any
	if err := json.Unmarshal(data, &arr); err == nil {
		out, _ := json.MarshalIndent(arr, "", "  ")
		fmt.Println(string(out))
		return
	}
	fmt.Println(string(data))
}

// Table prints data as an aligned table
func Table(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	fmt.Fprintln(w, strings.Repeat("-\t", len(headers)))
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	w.Flush()
}

// Quiet prints one value per line (typically IDs)
func Quiet(values []string) {
	for _, v := range values {
		fmt.Println(v)
	}
}

// ErrorJSON prints error as JSON to stderr
func ErrorJSON(msg string) {
	data, _ := json.Marshal(map[string]string{"error": msg})
	fmt.Fprintln(os.Stderr, string(data))
}
