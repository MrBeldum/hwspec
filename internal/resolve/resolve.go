// Package resolve fills in human-readable names from the raw hardware IDs
// stored in a report. It runs after every capture and again whenever a
// saved capture is shown, so older files pick up newer ID databases and the
// user's overrides.
//
// A name is only replaced when a database has one; otherwise the existing
// value (e.g. a USB device's own product string) is kept.
package resolve

import (
	"github.com/jiegui2025/hwspec/internal/ids"
	"github.com/jiegui2025/hwspec/internal/report"
)

func Names(r *report.Report) {
	for i := range r.PCI {
		d := &r.PCI[i]
		set(&d.Vendor, ids.PCIVendor(d.VendorID))
		set(&d.Device, ids.PCIDevice(d.VendorID, d.DeviceID))
		set(&d.Subsystem, ids.PCISubsystem(d.VendorID, d.DeviceID, d.SubVendorID, d.SubDeviceID))
		set(&d.Class, ids.PCIClass(d.ClassCode))
	}

	for i := range r.GPUs {
		g := &r.GPUs[i]
		if dev := pciByAddress(r, g.PCIAddress); dev != nil {
			// Captures from before the GPU ID fields existed.
			if g.VendorID == "" {
				g.VendorID, g.DeviceID, g.Revision = dev.VendorID, dev.DeviceID, dev.Revision
			}
			set(&g.Subsystem, dev.Subsystem)
		}
		set(&g.Vendor, ids.PCIVendor(g.VendorID))
		set(&g.Chip, ids.PCIDevice(g.VendorID, g.DeviceID))
		model := g.Chip
		if g.VendorID == "1002" {
			if retail := ids.AMDGPUName(g.DeviceID, g.Revision); retail != "" {
				model = retail
			}
		}
		set(&g.Model, model)
		if g.Model == "" && g.VendorID != "" {
			g.Model = g.VendorID + ":" + g.DeviceID
		}
	}

	for i := range r.USB {
		u := &r.USB[i]
		set(&u.Vendor, ids.USBVendor(u.VendorID))
		set(&u.Product, ids.USBProduct(u.VendorID, u.ProductID))
		set(&u.Class, ids.USBClass(u.ClassCode))
	}

	for i := range r.Network {
		n := &r.Network[i]
		n.Vendor, n.Model = adapterName(r, n.Bus, n.BusAddress, n.Vendor, n.Model)
		// A redacted file has no MAC; keep the vendor recorded at capture.
		if n.MAC != "" {
			n.MACVendor = ids.MACVendor(n.MAC)
		}
	}

	if r.CPU.Family > 0 {
		codename, uarch := ids.CPUCodename(r.CPU.Vendor, r.CPU.Family, r.CPU.ModelID, r.CPU.Stepping)
		set(&r.CPU.Codename, codename)
		set(&r.CPU.Microarchitecture, uarch)
	}

	for i := range r.Bluetooth {
		b := &r.Bluetooth[i]
		if b.ManufacturerID > 0 || b.Version != "" {
			set(&b.Manufacturer, ids.BluetoothCompany(uint16(b.ManufacturerID)))
		}
		if b.Address != "" {
			b.AddressVendor = ids.MACVendor(b.Address)
		}
		b.Vendor, b.Model = adapterName(r, b.Bus, b.BusAddress, b.Vendor, b.Model)
	}

	for i := range r.Audio {
		for j := range r.Audio[i].Codecs {
			c := &r.Audio[i].Codecs[j]
			// HDA vendor IDs are the PCI vendor in the top 16 bits.
			if len(c.VendorID) == 8 {
				set(&c.Vendor, ids.PCIVendor(c.VendorID[:4]))
			}
		}
	}

	for i := range r.Displays {
		d := &r.Displays[i]
		set(&d.Manufacturer, ids.PNPVendor(d.ManufacturerID))
	}

	for i := range r.Memory.Modules {
		m := &r.Memory.Modules[i]
		raw := m.ManufacturerRaw
		if raw == "" {
			raw = m.Manufacturer
		}
		if name, _, ok := ids.MemoryManufacturer(raw); ok {
			m.Manufacturer, m.ManufacturerRaw = name, raw
		}
	}

	if r.Tool.IDDatabases == nil {
		r.Tool.IDDatabases = map[string]string{}
	}
	for k, v := range ids.Loaded() {
		r.Tool.IDDatabases[string(k)] = v
	}
}

// adapterName looks up the PCI or USB device behind an interface.
func adapterName(r *report.Report, bus, addr, vendor, model string) (string, string) {
	switch bus {
	case "pci":
		if dev := pciByAddress(r, addr); dev != nil {
			set(&vendor, dev.Vendor)
			set(&model, dev.Device)
		}
	case "usb":
		for _, u := range r.USB {
			if u.Path == addr {
				set(&vendor, u.Vendor)
				set(&model, u.Product)
			}
		}
	}
	return vendor, model
}

func set(field *string, name string) {
	if name != "" {
		*field = name
	}
}

func pciByAddress(r *report.Report, addr string) *report.PCIDevice {
	for i := range r.PCI {
		if r.PCI[i].Address == addr {
			return &r.PCI[i]
		}
	}
	return nil
}
