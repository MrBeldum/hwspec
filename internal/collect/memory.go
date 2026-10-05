package collect

import (
	"os"
	"strconv"
	"strings"

	"github.com/jiegui2025/hwspec/internal/report"
	"github.com/jiegui2025/hwspec/internal/smbios"
)

func (c *collector) memory() {
	m := &c.r.Memory
	m.Modules = []report.MemoryModule{}
	for _, line := range strings.Split(readStr("/proc/meminfo"), "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		kb, err := strconv.ParseUint(strings.TrimSuffix(strings.TrimSpace(v), " kB"), 10, 64)
		if err != nil {
			continue
		}
		switch k {
		case "MemTotal":
			m.TotalBytes = kb << 10
		case "SwapTotal":
			m.SwapBytes = kb << 10
		}
	}

	table, err := os.ReadFile(p("/sys/firmware/dmi/tables/DMI"))
	if err != nil {
		if !os.IsNotExist(err) {
			c.warnRead("memory modules (SMBIOS)", err)
		}
		return
	}
	structs := smbios.Parse(table)
	for _, a := range smbios.MemoryArrays(structs) {
		m.MaxCapacityBytes += a.MaxCapacityBytes
		m.Slots += a.Slots
		if m.ECC == "" {
			m.ECC = a.ErrorCorrection
		}
	}
	for _, d := range smbios.MemoryDevices(structs) {
		if d.SizeBytes == 0 {
			continue // empty slot
		}
		m.InstalledBytes += d.SizeBytes
		m.Modules = append(m.Modules, report.MemoryModule{
			Locator:       d.Locator,
			BankLocator:   d.BankLocator,
			SizeBytes:     d.SizeBytes,
			Type:          d.Type,
			FormFactor:    d.FormFactor,
			SpeedMTs:      d.SpeedMTs,
			ConfiguredMTs: d.ConfiguredMTs,
			Manufacturer:  d.Manufacturer,
			PartNumber:    d.PartNumber,
			Serial:        d.Serial,
			DataWidth:     d.DataWidth,
			TotalWidth:    d.TotalWidth,
			Rank:          d.Rank,
			VoltageMV:     d.ConfiguredMV,
		})
	}
}
