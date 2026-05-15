package cmd

import (
	"testing"

	"github.com/kaspermunck/virkcli/virk"
)

func TestFormatInt(t *testing.T) {
	tests := []struct {
		in   int64
		want string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1.000"},
		{1234567, "1.234.567"},
		{-42, "-42"},
		{-1234567, "-1.234.567"},
	}
	for _, tt := range tests {
		if got := formatInt(tt.in); got != tt.want {
			t.Errorf("formatInt(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestTrunc(t *testing.T) {
	tests := []struct {
		in   string
		n    int
		want string
	}{
		{"hello", 10, "hello"},
		{"hello world", 6, "hello…"},
		{"ab", 2, "ab"},
	}
	for _, tt := range tests {
		if got := trunc(tt.in, tt.n); got != tt.want {
			t.Errorf("trunc(%q, %d) = %q, want %q", tt.in, tt.n, got, tt.want)
		}
	}
}

func TestAddressLine1(t *testing.T) {
	a := virk.Address{Street: "Vestergade 10", Floor: "2", Door: "th"}
	got := addressLine1(a)
	want := "Vestergade 10, 2. th"
	if got != want {
		t.Errorf("addressLine1 = %q, want %q", got, want)
	}
}

func TestAddressLine2(t *testing.T) {
	a := virk.Address{Postcode: "1000", City: "København K", Country: "DK"}
	got := addressLine2(a)
	want := "1000 København K"
	if got != want {
		t.Errorf("addressLine2 = %q, want %q", got, want)
	}

	a.Country = "SE"
	got = addressLine2(a)
	want = "1000 København K, SE"
	if got != want {
		t.Errorf("addressLine2 (foreign) = %q, want %q", got, want)
	}
}

func TestDkkCell(t *testing.T) {
	if got := dkkCell(nil); got != "—" {
		t.Errorf("dkkCell(nil) = %q", got)
	}
	v := int64(1234)
	if got := dkkCell(&v); got != "1.234" {
		t.Errorf("dkkCell(1234) = %q", got)
	}
}
