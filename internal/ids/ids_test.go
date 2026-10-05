package ids

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// isolate makes lookups use only the embedded data (as on NixOS, which has
// no hwdata path), plus whatever the test sets up, and restores afterwards.
func isolate(t *testing.T) {
	t.Helper()
	oldSys, oldSynced, oldOv := systemEnabled, syncedDir, overridesPath
	oldSpecs := map[Kind]spec{}
	for k, v := range specs {
		oldSpecs[k] = v
	}
	systemEnabled, syncedDir, overridesPath = false, "", ""
	Reset()
	t.Cleanup(func() {
		systemEnabled, syncedDir, overridesPath = oldSys, oldSynced, oldOv
		specs = oldSpecs
		Reset()
	})
}

func TestEmbeddedLookups(t *testing.T) {
	isolate(t)
	cases := []struct {
		name, got, want string
	}{
		{"pci vendor", PCIVendor("8086"), "Intel Corporation"},
		{"pci vendor 0x", PCIVendor("0x10DE"), "NVIDIA Corporation"},
		{"pci class", PCIClass("030000"), "VGA compatible controller"},
		{"pci subclass fallback", PCIClass("0380ff"), "Display controller"},
		{"pci class only", PCIClass("02"), "Network controller"},
		{"usb vendor", USBVendor("046d"), "Logitech, Inc."},
		{"usb class", USBClass("03"), "Human Interface Device"},
		{"pnp", PNPVendor("del"), "Dell Inc."},
		{"amdgpu", AMDGPUName("1114", "C2"), "AMD Radeon 860M Graphics"},
		{"oui", MACVendor("04:0e:3c:91:34:f4"), "HP Inc."},
		{"oui local", MACVendor("02:0e:3c:91:34:f4"), ""},
		{"oui garbage", MACVendor("zz"), ""},
		{"missing", PCIDevice("8086", "zzzz"), ""},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.name, c.got, c.want)
		}
	}
	if PCIDevice("8086", "3e92") == "" {
		t.Error("PCIDevice(8086, 3e92) is empty")
	}
	for k, src := range Loaded() {
		if !strings.HasPrefix(src, "embedded") || strings.Contains(src, "+") {
			t.Errorf("%s loaded from %q, want embedded only", k, src)
		}
	}
}

func TestMemoryManufacturer(t *testing.T) {
	isolate(t)
	cases := map[string]string{
		"80CE":               "Samsung",
		"80AD":               "SK Hynix (former Hyundai Electronics)",
		"802C":               "Micron Technology",
		"2C00":               "Micron Technology",
		"CE00000000000000":   "Samsung",
		"00CE000000000000":   "Samsung",
		"859B":               "Crucial Technology",
		"0x859B":             "Crucial Technology",
		"7F7F7F7F7F9B":       "Crucial Technology",
		"Unknown - [0xF785]": "Avant Technology",
		"0198":               "Kingston",
		"04CB":               "A-DATA Technology",
	}
	for raw, want := range cases {
		got, _, ok := MemoryManufacturer(raw)
		if !ok || got != want {
			t.Errorf("MemoryManufacturer(%q) = %q, %v; want %q", raw, got, ok, want)
		}
	}
	for _, raw := range []string{"Samsung", "Micron Technology", "", "Unknown", "1234567", "12345678"} {
		if got, _, ok := MemoryManufacturer(raw); ok {
			t.Errorf("MemoryManufacturer(%q) = %q, want no match", raw, got)
		}
	}
}

func write(t *testing.T, path, content string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestOverrides(t *testing.T) {
	isolate(t)
	overridesPath = write(t, filepath.Join(t.TempDir(), "overrides.ids"), `# test
pci 8086 = Intel (override)
pci 8086:3e92 = My iGPU
pci class 0300 = Graphics
usb 046d:c52b = Receiver
pnp DEL = Dell
oui 04-0E-3C = Mine
jedec F785 = Avant (override)
amdgpu 1114:C2 = Radeon
this line is wrong
pci = missing key
`)
	checks := map[string][2]string{
		"pci vendor":  {PCIVendor("8086"), "Intel (override)"},
		"pci device":  {PCIDevice("8086", "3e92"), "My iGPU"},
		"pci class":   {PCIClass("030000"), "Graphics"},
		"usb":         {USBProduct("046d", "c52b"), "Receiver"},
		"pnp":         {PNPVendor("DEL"), "Dell"},
		"oui":         {MACVendor("04:0e:3c:00:00:01"), "Mine"},
		"amdgpu":      {AMDGPUName("1114", "c2"), "Radeon"},
		"other pci":   {PCIVendor("10de"), "NVIDIA Corporation"},
		"other jedec": {first(MemoryManufacturer("80CE")), "Samsung"},
		"jedec":       {first(MemoryManufacturer("Unknown - [0xF785]")), "Avant (override)"},
	}
	for name, c := range checks {
		if c[0] != c[1] {
			t.Errorf("%s = %q, want %q", name, c[0], c[1])
		}
	}
	layers := Layers(PCI)
	last := layers[len(layers)-1]
	if last.Source != overridesPath || last.Entries != 3 {
		t.Errorf("last PCI layer = %+v, want overrides with 3 entries", last)
	}
	if !strings.Contains(last.Err, "line 10") || !strings.Contains(last.Err, "line 11") {
		t.Errorf("bad lines not reported: %q", last.Err)
	}
}

func first(name, _ string, _ bool) string { return name }

func TestLayerOrder(t *testing.T) {
	isolate(t)
	dir := t.TempDir()
	systemEnabled = true
	sys := filepath.Join(dir, "sys-pci.ids")
	s := specs[PCI]
	s.system = []string{sys}
	specs[PCI] = s

	// An older distro file must not override newer embedded names...
	write(t, sys, "#\tVersion: 2000.01.01\n8086  Old Intel\nffe0  Only In System\n")
	Reset()
	if got := PCIVendor("8086"); got != "Intel Corporation" {
		t.Errorf("older system file won: %q", got)
	}
	if got := PCIVendor("ffe0"); got != "Only In System" {
		t.Errorf("older system file's extra entry lost: %q", got)
	}
	if l := Layers(PCI); l[0].Source != sys || l[1].Source != "embedded" {
		t.Errorf("layers = %+v, want system then embedded", l)
	}

	// ...but a newer one does.
	write(t, sys, "#\tVersion: 2999.01.01\n8086  New Intel\n")
	Reset()
	if got := PCIVendor("8086"); got != "New Intel" {
		t.Errorf("newer system file lost: %q", got)
	}

	// A synced copy applies over both.
	syncedDir = filepath.Join(dir, "synced")
	write(t, filepath.Join(syncedDir, "pci.ids"), "8086  Synced Intel\n")
	Reset()
	if got := PCIVendor("8086"); got != "Synced Intel" {
		t.Errorf("synced file lost: %q", got)
	}
}

func TestLookup(t *testing.T) {
	isolate(t)
	cases := []struct {
		kind      Kind
		id, want  string
		wantError bool
	}{
		{PCI, "8086", "Intel Corporation", false},
		{PCI, "class 0300", "VGA compatible controller", false},
		{USB, "046d", "Logitech, Inc.", false},
		{JEDEC, "F785", "Avant Technology", false},
		{JEDEC, "6:77", "Avant Technology", false},
		{OUI, "04:0E:3C", "HP Inc.", false},
		{AMDGPU, "1114:c2", "AMD Radeon 860M Graphics", false},
		{JEDEC, "Samsung", "", true},
		{"nope", "1", "", true},
	}
	for _, c := range cases {
		got, _, err := Lookup(c.kind, c.id)
		if (err != nil) != c.wantError || got != c.want {
			t.Errorf("Lookup(%s, %q) = %q, %v; want %q (error %v)", c.kind, c.id, got, err, c.want, c.wantError)
		}
	}
}
