package collect

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/jiegui2025/hwspec/internal/edid"
	"github.com/jiegui2025/hwspec/internal/report"
)

const pciDir = "/sys/bus/pci/devices/"

func (c *collector) pci() {
	c.r.PCI = []report.PCIDevice{}
	for _, addr := range list(pciDir) {
		d := pciDir + addr + "/"
		vid, did := hex4(readStr(d+"vendor")), hex4(readStr(d+"device"))
		svid, sdid := hex4(readStr(d+"subsystem_vendor")), hex4(readStr(d+"subsystem_device"))
		class := hex4(readStr(d + "class"))
		dev := report.PCIDevice{
			Address:     addr,
			VendorID:    vid,
			DeviceID:    did,
			SubVendorID: svid,
			SubDeviceID: sdid,
			ClassCode:   class,
			Revision:    hex4(readStr(d + "revision")),
			Driver:      linkBase(d + "driver"),
			IOMMUGroup:  linkBase(d + "iommu_group"),
			Link:        pcieLink(d),
		}
		c.r.PCI = append(c.r.PCI, dev)
	}
	if len(c.r.PCI) == 0 {
		c.warn("pci: no devices under %s", pciDir)
	}
}

func pcieLink(dir string) *report.PCIeLink {
	speed := readStr(dir + "current_link_speed")
	if speed == "" || strings.HasPrefix(speed, "Unknown") {
		return nil
	}
	w, _ := readInt(dir + "current_link_width")
	mw, _ := readInt(dir + "max_link_width")
	return &report.PCIeLink{
		Speed:    speed,
		Width:    int(w),
		MaxSpeed: readStr(dir + "max_link_speed"),
		MaxWidth: int(mw),
	}
}

// gpus picks display controllers (PCI class 03) from the PCI list and adds
// DRM details: card name, VRAM (amdgpu) and output connectors. Names are
// filled in later by the resolve package.
func (c *collector) gpus() {
	c.r.GPUs = []report.GPU{}
	cards := drmCards()
	for _, dev := range c.r.PCI {
		if !strings.HasPrefix(dev.ClassCode, "03") {
			continue
		}
		g := report.GPU{
			PCIAddress: dev.Address,
			VendorID:   dev.VendorID,
			DeviceID:   dev.DeviceID,
			Revision:   dev.Revision,
			Driver:     dev.Driver,
			BootVGA:    readStr(pciDir+dev.Address+"/boot_vga") == "1",
			Link:       dev.Link,
			Outputs:    []string{},
		}
		g.VRAMBytes = readUint(pciDir + dev.Address + "/mem_info_vram_total") // amdgpu
		for card, addr := range cards {
			if addr != dev.Address {
				continue
			}
			g.DRMCard = card
			for _, conn := range list("/sys/class/drm") {
				if strings.HasPrefix(conn, card+"-") {
					g.Outputs = append(g.Outputs, strings.TrimPrefix(conn, card+"-"))
				}
			}
		}
		c.r.GPUs = append(c.r.GPUs, g)
	}
}

// drmCards maps DRM card names (card0, card1) to their PCI addresses.
func drmCards() map[string]string {
	out := map[string]string{}
	for _, n := range list("/sys/class/drm") {
		if !strings.HasPrefix(n, "card") || strings.Contains(n, "-") {
			continue
		}
		if bus, addr := busOf("/sys/class/drm/" + n + "/device"); bus == "pci" {
			out[n] = addr
		}
	}
	return out
}

// displays decodes the EDID of every connected monitor.
func (c *collector) displays() {
	c.r.Displays = []report.Display{}
	for _, conn := range list("/sys/class/drm") {
		if !strings.Contains(conn, "-") || readStr("/sys/class/drm/"+conn+"/status") != "connected" {
			continue
		}
		disp := report.Display{Connector: conn}
		raw, err := os.ReadFile(p("/sys/class/drm/" + conn + "/edid"))
		if err == nil && len(raw) > 0 {
			if e, err := edid.Parse(raw); err == nil {
				disp.ManufacturerID = e.ManufacturerID
				disp.Model = e.Name
				disp.ProductCode = fmt.Sprintf("%04X", e.ProductCode)
				disp.Serial = e.SerialText
				if disp.Serial == "" && e.SerialNumber != 0 && e.SerialNumber != 0x01010101 {
					disp.Serial = strconv.FormatUint(uint64(e.SerialNumber), 10)
				}
				disp.Year = e.Year
				disp.WidthMM, disp.HeightMM = e.WidthMM, e.HeightMM
				disp.DiagonalIn = e.DiagonalInches()
				disp.NativeWidth, disp.NativeHeight = e.NativeWidth, e.NativeHeight
				disp.NativeRefreshHz = e.NativeRefreshHz
				disp.EDIDVersion = e.Version
			} else {
				c.warn("display %s: %v", conn, err)
			}
		}
		c.r.Displays = append(c.r.Displays, disp)
	}
}
