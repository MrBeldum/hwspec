package output

import (
	"bytes"
	"errors"
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

func TestShowRejectsFilesThatAreNotCaptures(t *testing.T) {
	for _, in := range []string{`{"name":"left-pad","version":"1.0"}`, "name: x\nversion: 2\n", `{"tool":{"name":"hwspec"}}`} {
		if _, err := Read([]byte(in)); !errors.Is(err, ErrNotCapture) {
			t.Errorf("Read(%q) err = %v, want ErrNotCapture", in, err)
		}
	}
}

// Device strings come from hardware and shared files; they must not reach
// the terminal as escape sequences.
func TestTextOutputIsTerminalSafe(t *testing.T) {
	r := sample()
	r.CPU.Model = "Evil\x1b]52;c;ZXZpbA==\x07\x1b[2J CPU\x9b"
	var buf bytes.Buffer
	if err := Write(&buf, r, "text"); err != nil {
		t.Fatal(err)
	}
	for _, c := range buf.String() {
		if c != '\n' && (c < 0x20 || c == 0x7f || (c >= 0x80 && c < 0xa0)) {
			t.Fatalf("control character %U in text output", c)
		}
	}
	if !strings.Contains(buf.String(), "Evil]52;c;ZXZpbA==[2J CPU") {
		t.Errorf("visible text lost:\n%s", buf.String())
	}
}

func TestTextShowsOnlyWhatWasReported(t *testing.T) {
	r := sample()
	r.Bluetooth = []report.BluetoothController{{Name: "hci0", Manufacturer: "   "}}
	r.Batteries = []report.Battery{{Name: "BAT0", Model: "5B10", CapacityPercent: 80}}
	r.CPU.Microarchitecture = "Zen 4" // no codename for this model
	r.OS.BootMode = ""
	var buf bytes.Buffer
	if err := Write(&buf, r, "text"); err != nil { // must not panic on a blank manufacturer
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"5B10, 80% charged", "Cores are  Zen 4", "boot mode unknown"} {
		if !strings.Contains(out, want) {
			t.Errorf("text lacks %q:\n%s", want, out)
		}
	}
	for _, unwanted := range []string{"0.0 of 0.0 Wh", "0% health", "0 cycles"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("text shows a missing figure as zero (%q):\n%s", unwanted, out)
		}
	}
}
