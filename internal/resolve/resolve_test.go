package resolve

import (
	"testing"

	"github.com/jiegui2025/hwspec/internal/report"
)

// A capture made before names were available (or with older databases)
// gets them from its raw IDs.
func TestNamesFromRawIDs(t *testing.T) {
	r := &report.Report{
		PCI: []report.PCIDevice{
			{Address: "0000:00:02.0", VendorID: "8086", DeviceID: "3e92", ClassCode: "030000"},
			{Address: "0000:03:00.0", VendorID: "1002", DeviceID: "1114", Revision: "c2", ClassCode: "030000"},
			{Address: "0000:00:1f.6", VendorID: "8086", DeviceID: "15bb", ClassCode: "020000"},
		},
		// Old-style GPU entries without IDs.
		GPUs: []report.GPU{{PCIAddress: "0000:00:02.0"}, {PCIAddress: "0000:03:00.0"}},
		USB: []report.USBDevice{
			{Path: "1-2", VendorID: "046d", ProductID: "ffff", Product: "Device's own name", ClassCode: "03"},
		},
		Network: []report.NIC{
			{Name: "eno1", Bus: "pci", BusAddress: "0000:00:1f.6", MAC: "04:0e:3c:00:00:01"},
			{Name: "usb0", Bus: "usb", BusAddress: "1-2", MAC: "02:00:00:00:00:01"},
		},
		Displays: []report.Display{{ManufacturerID: "DEL"}},
		Memory: report.Memory{Modules: []report.MemoryModule{
			{Manufacturer: "Unknown - [0xF785]"},
			{Manufacturer: "Samsung"},
		}},
	}
	Names(r)

	checks := []struct{ name, got, want string }{
		{"pci vendor", r.PCI[0].Vendor, "Intel Corporation"},
		{"pci class", r.PCI[0].Class, "VGA compatible controller"},
		{"intel gpu model", r.GPUs[0].Model, r.GPUs[0].Chip},
		{"intel gpu vendor", r.GPUs[0].Vendor, "Intel Corporation"},
		{"amd gpu retail name", r.GPUs[1].Model, "AMD Radeon 860M Graphics"},
		{"amd gpu ids backfilled", r.GPUs[1].DeviceID, "1114"},
		{"usb vendor", r.USB[0].Vendor, "Logitech, Inc."},
		{"usb product kept", r.USB[0].Product, "Device's own name"},
		{"usb class", r.USB[0].Class, "Human Interface Device"},
		{"nic from pci", r.Network[0].Vendor, "Intel Corporation"},
		{"nic mac vendor", r.Network[0].MACVendor, "HP Inc."},
		{"nic from usb", r.Network[1].Model, "Device's own name"},
		{"local mac", r.Network[1].MACVendor, ""},
		{"display", r.Displays[0].Manufacturer, "Dell Inc."},
		{"memory decoded", r.Memory.Modules[0].Manufacturer, "Avant Technology"},
		{"memory raw kept", r.Memory.Modules[0].ManufacturerRaw, "Unknown - [0xF785]"},
		{"memory plain name", r.Memory.Modules[1].Manufacturer, "Samsung"},
		{"memory plain raw", r.Memory.Modules[1].ManufacturerRaw, ""},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.name, c.got, c.want)
		}
	}
	if r.GPUs[0].Chip == "" {
		t.Error("intel gpu chip name empty")
	}
	if r.Tool.IDDatabases["pci"] == "" || r.Tool.IDDatabases["jedec"] == "" {
		t.Errorf("id_databases not recorded: %v", r.Tool.IDDatabases)
	}

	// Running again (e.g. `hwspec show` on the saved file) is stable.
	before := r.Memory.Modules[0]
	Names(r)
	if r.Memory.Modules[0] != before {
		t.Errorf("second pass changed memory module: %+v", r.Memory.Modules[0])
	}
}
