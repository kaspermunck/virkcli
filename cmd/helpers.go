package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/kaspermunck/virkcli/virk"
)

// writePrettyJSON pretty-prints raw JSON to stdout, falling back to raw output
// if indentation fails. Used by --raw across all subcommands.
func writePrettyJSON(raw []byte) error {
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, raw, "", "  "); err != nil {
		_, _ = os.Stdout.Write(raw)
		fmt.Fprintln(os.Stdout)
		return nil
	}
	_, _ = os.Stdout.Write(pretty.Bytes())
	fmt.Fprintln(os.Stdout)
	return nil
}

func encodeJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func writeBytes(b []byte, err error) error {
	if err != nil {
		return err
	}
	_, _ = os.Stdout.Write(b)
	fmt.Fprintln(os.Stdout)
	return nil
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func addressLine1(a virk.Address) string {
	parts := []string{}
	if a.Street != "" {
		parts = append(parts, a.Street)
	}
	if a.Floor != "" || a.Door != "" {
		floorDoor := strings.TrimSpace(fmt.Sprintf("%s. %s", a.Floor, a.Door))
		parts = append(parts, floorDoor)
	}
	return strings.Join(parts, ", ")
}

func addressLine2(a virk.Address) string {
	line := strings.TrimSpace(fmt.Sprintf("%s %s", a.Postcode, a.City))
	if a.Country != "" && a.Country != "DK" {
		line = strings.TrimSpace(line + ", " + a.Country)
	}
	return line
}

func formatInt(n int64) string {
	s := fmt.Sprintf("%d", n)
	neg := false
	if s[0] == '-' {
		neg = true
		s = s[1:]
	}
	if len(s) <= 3 {
		if neg {
			return "-" + s
		}
		return s
	}
	var result []byte
	for i, c := range s {
		pos := len(s) - i
		if i > 0 && pos%3 == 0 {
			result = append(result, '.')
		}
		result = append(result, byte(c))
	}
	if neg {
		return "-" + string(result)
	}
	return string(result)
}

func dkkCell(v *int64) string {
	if v == nil {
		return "—"
	}
	return formatInt(*v)
}

func printField(label string, v *int64) {
	if v == nil {
		return
	}
	fmt.Fprintf(os.Stdout, "  %-18s %s DKK\n", label+":", formatInt(*v))
}
