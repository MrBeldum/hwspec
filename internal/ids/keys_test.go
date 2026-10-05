package ids

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// People write IDs in many ways; every reasonable spelling must reach the
// same database entry, in overrides and in lookups alike.
func TestEquivalentSpellingsResolveTheSame(t *testing.T) {
	isolate(t)
	cases := []struct {
		kind      Kind
		spellings []string
		want      string
	}{
		{USB, []string{"046d", "46d", "0x046D", " 046D "}, "Logitech, Inc."},
		{PCI, []string{"8086", "0x8086"}, "Intel Corporation"},
		{BT, []string{"2", "0002", "0x0002"}, "Intel Corp."},
		{JEDEC, []string{"1:4E", "1:CE", "80CE", "80ce"}, "Samsung"},
		{AMDGPU, []string{"1114:c2", "1114:C2", "0x1114:0xc2"}, "AMD Radeon 860M Graphics"},
		{CPU, []string{"intel:6:9e:10", "INTEL:0x6:9E:10"}, "Coffee Lake | Skylake"},
		{CPU, []string{"intel:6:9e:99"}, "Kaby Lake | Skylake"}, // unknown stepping: model entry
		{OUI, []string{"04:0E:3C", "04-0e-3c-12-34-56", "040e3c"}, "HP Inc."},
		{PNP, []string{"DEL", "del"}, "Dell Inc."},
		{PCI, []string{"class 0300", "class 030000"}, "VGA compatible controller"},
	}
	for _, c := range cases {
		for _, s := range c.spellings {
			got, _, err := Lookup(c.kind, s)
			if err != nil || got != c.want {
				t.Errorf("Lookup(%s, %q) = %q, %v; want %q", c.kind, s, got, err, c.want)
			}
		}
	}
}

func TestOverridesAcceptEverySpelling(t *testing.T) {
	isolate(t)
	overridesPath = write(t, filepath.Join(t.TempDir(), "overrides.ids"), strings.Join([]string{
		"usb 46d = Short USB",
		"bluetooth 2 = Short BT",
		"jedec 1:CE = Parity JEDEC",
		"pci 8086:3E92 = Upper PCI",
	}, "\n"))
	checks := map[string][2]string{
		"usb":   {USBVendor("046d"), "Short USB"},
		"bt":    {BluetoothCompany(2), "Short BT"},
		"jedec": {first(MemoryManufacturer("80CE")), "Parity JEDEC"},
		"pci":   {PCIDevice("8086", "3e92"), "Upper PCI"},
	}
	for name, c := range checks {
		if c[0] != c[1] {
			t.Errorf("%s override = %q, want %q", name, c[0], c[1])
		}
	}
	if err := OverridesError(); err != nil {
		t.Errorf("valid overrides reported as errors: %v", err)
	}
}

func TestMalformedIDsAreRejectedWithAReason(t *testing.T) {
	bad := []struct {
		kind Kind
		id   string
	}{
		{USB, "12345"}, {USB, "zz"}, {USB, "1:2:3"}, {PCI, "1:2:3"}, {USB, "class 0300"},
		{PNP, "DE"}, {PNP, "D3L"}, {OUI, "zz:zz"}, {JEDEC, "0:4E"}, {JEDEC, "1:7F"}, {JEDEC, "Samsung"},
		{AMDGPU, "1114"}, {AMDGPU, "1114:zz"}, {BT, "10000"}, {CPU, "via:6:1"}, {CPU, "intel:x:1"},
		{CPU, "intel:6:9e:-1"}, {"nope", "1"},
	}
	for _, c := range bad {
		if key, err := NormalizeKey(c.kind, c.id); err == nil {
			t.Errorf("NormalizeKey(%s, %q) = %q, want an error", c.kind, c.id, key)
		}
	}
}

// Names come from upstream files, devices and captures; none may carry
// terminal escape sequences to the screen.
func TestNamesNeverCarryControlCharacters(t *testing.T) {
	for in, want := range map[string]string{
		"Intel\x1b]52;c;ZWNobw==\x07 Corp": "Intel]52;c;ZWNobw== Corp",
		"plain":                            "plain",
		"tab\there":                        "tabhere",
		"bad \xff utf8":                    "bad � utf8",
		"c1 \u009b31m":                     "c1 31m",
	} {
		if got := CleanName(in); got != want {
			t.Errorf("CleanName(%q) = %q, want %q", in, got, want)
		}
	}
	isolate(t)
	overridesPath = write(t, filepath.Join(t.TempDir(), "overrides.ids"), "pci 8086 = Evil\x1b[2J Corp\n")
	if got := PCIVendor("8086"); got != "Evil[2J Corp" {
		t.Errorf("override name not cleaned: %q", got)
	}
}

func TestUpdateRefusesBundlesOlderThanBuiltInData(t *testing.T) {
	isolate(t)
	syncedDir = filepath.Join(t.TempDir(), "ids")
	b := newBundle(t)
	b.publish(embeddedManifest().GeneratedAt.Add(-time.Hour), b.key)
	if _, err := b.update(UpdateOptions{}); err == nil || !strings.Contains(err.Error(), "built into hwspec") {
		t.Errorf("err = %v, want refusal of a bundle older than the built-in data", err)
	}
	if _, err := os.Stat(syncedDir); !os.IsNotExist(err) {
		t.Error("files were installed from a refused bundle")
	}
	if _, err := b.update(UpdateOptions{AllowOlder: true}); err != nil {
		t.Errorf("AllowOlder: %v", err)
	}
}

func TestUpdateRefusesWhenTheInstalledManifestIsUnreadable(t *testing.T) {
	isolate(t)
	syncedDir = filepath.Join(t.TempDir(), "ids")
	b := newBundle(t)
	write(t, filepath.Join(syncedDir, "manifest.json"), "{broken")
	if _, err := b.update(UpdateOptions{}); err == nil || !strings.Contains(err.Error(), "--allow-older") {
		t.Errorf("err = %v, want a refusal that names --allow-older", err)
	}
}

func TestInstallErrorSaysWhatWasReplaced(t *testing.T) {
	cause := errors.New("disk full")
	none := &InstallError{Err: cause}
	some := &InstallError{Written: []string{"pci.ids.gz", "usb.ids.gz"}, Err: cause}
	if !strings.Contains(none.Error(), "before any file was replaced") {
		t.Errorf("none: %q", none.Error())
	}
	if !strings.Contains(some.Error(), "pci.ids.gz, usb.ids.gz") || !errors.Is(some, cause) {
		t.Errorf("some: %q", some.Error())
	}
}

func TestInstallFailureIsReportedAsPartial(t *testing.T) {
	isolate(t)
	dir := t.TempDir()
	syncedDir = filepath.Join(dir, "ids")
	b := newBundle(t)
	// A directory where a file must go makes that rename fail midway.
	if err := os.MkdirAll(filepath.Join(syncedDir, "usb.ids.gz", "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := b.update(UpdateOptions{})
	var ie *InstallError
	if !errors.As(err, &ie) || len(ie.Written) == 0 {
		t.Fatalf("err = %v, want an InstallError listing what was written", err)
	}
}
