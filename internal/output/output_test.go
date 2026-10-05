package output

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jiegui2025/hwspec/internal/report"
)

func sample() *report.Report {
	on := true
	return &report.Report{
		SchemaVersion: report.SchemaVersion,
		Tool:          report.Tool{Name: "hwspec", Version: "test"},
		CapturedAt:    time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC),
		Hostname:      "box",
		OS:            report.OS{PrettyName: "Test OS", SecureBoot: &on, Virtualization: "none"},
		// Strings that look like other YAML types must survive a round trip.
		Board:    report.Board{Product: "8595", Version: "true"},
		BIOS:     report.BIOS{Release: "27.0"},
		Memory:   report.Memory{TotalBytes: 32 << 30, Modules: []report.MemoryModule{}},
		Storage:  []report.Disk{{Name: "sda", Serial: "S1", Partitions: []report.Partition{{Name: "sda1", UUID: "u"}}}},
		Network:  []report.NIC{{Name: "eth0", MAC: "00:11:22:33:44:55"}},
		Warnings: []string{"x: needs root"},
	}
}

func TestRoundTrip(t *testing.T) {
	for _, format := range []string{"json", "yaml"} {
		var buf bytes.Buffer
		if err := Write(&buf, sample(), format); err != nil {
			t.Fatalf("%s: %v", format, err)
		}
		got, err := Read(buf.Bytes())
		if err != nil {
			t.Fatalf("%s: read back: %v", format, err)
		}
		if !reflect.DeepEqual(got, sample()) {
			t.Errorf("%s round trip changed the report:\n%s", format, buf.String())
		}
	}
}

func TestText(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, sample(), "text"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Test OS", "32 GiB usable", "x: needs root"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("text output lacks %q:\n%s", want, buf.String())
		}
	}
}

func TestRedact(t *testing.T) {
	r := sample()
	r.Redact()
	if r.Hostname != "" || r.Storage[0].Serial != "" || r.Storage[0].Partitions[0].UUID != "" || r.Network[0].MAC != "" || !r.Redacted {
		t.Errorf("redact left identifiers: %+v", r)
	}
}

func TestBytesStr(t *testing.T) {
	for n, want := range map[uint64]string{512: "512 B", 32 << 10: "32 KiB", 9 << 20: "9 MiB", 256060514304: "238.5 GiB"} {
		if got := bytesStr(n); got != want {
			t.Errorf("bytesStr(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestMachine(t *testing.T) {
	for _, tc := range []struct {
		name string
		sys  report.System
		want string
	}{
		{"product repeats the vendor", report.System{Vendor: "HP", Product: "HP EliteDesk 800 G5 Desktop Mini"}, "HP EliteDesk 800 G5 Desktop Mini"},
		{"repeat in another case", report.System{Vendor: "LENOVO", Product: "Lenovo ThinkCentre M720q"}, "Lenovo ThinkCentre M720q"},
		{"product is only the vendor", report.System{Vendor: "Framework", Product: "framework"}, "framework"},
		{"vendor not repeated", report.System{Vendor: "Dell Inc.", Product: "OptiPlex 7090", Version: "1.0"}, "Dell Inc. OptiPlex 7090 1.0"},
		{"product starts with a longer word", report.System{Vendor: "HP", Product: "HPE ProLiant"}, "HP HPE ProLiant"},
		{"padding around the vendor", report.System{Vendor: " HP ", Product: "HP Z2 G9"}, "HP Z2 G9"},
		{"no vendor", report.System{Product: "Z2 G9", Version: "1"}, "Z2 G9 1"},
		{"nothing known", report.System{}, "unknown"},
	} {
		if got := machine(tc.sys); got != tc.want {
			t.Errorf("%s: machine(%+v) = %q, want %q", tc.name, tc.sys, got, tc.want)
		}
	}
}

func TestTextMachineLineDoesNotRepeatVendor(t *testing.T) {
	r := sample()
	r.System = report.System{Vendor: "HP", Product: "HP EliteDesk 800 G5 Desktop Mini"}
	var buf bytes.Buffer
	if err := Write(&buf, r, "text"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Machine    HP EliteDesk 800 G5 Desktop Mini\n") {
		t.Errorf("Machine line repeats or drops the vendor:\n%s", buf.String())
	}
}
