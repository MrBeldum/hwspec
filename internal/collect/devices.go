package collect

import (
	"bufio"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/jiegui2025/hwspec/internal/report"
)

// network lists physical interfaces. Virtual ones (bridges, veth, VPN
// tunnels, containers) have no backing device and are skipped.
func (c *collector) network() {
	c.r.Network = []report.NIC{}
	for _, name := range list("/sys/class/net") {
		d := "/sys/class/net/" + name + "/"
		if !exists(d + "device") {
			continue
		}
		nic := report.NIC{
			Name:   name,
			Type:   "ethernet",
			MAC:    readStr(d + "address"),
			Driver: linkBase(d + "device/driver"),
			State:  readStr(d + "operstate"),
		}
		if exists(d+"wireless") || exists(d+"phy80211") {
			nic.Type = "wireless"
		} else if t := readStr(d + "type"); t != "1" {
			nic.Type = "other (" + t + ")"
		}
		nic.Bus, nic.BusAddress = busOf(d + "device")
		// speed and duplex are only meaningful while the link is up;
		// reading them on a down link fails or returns -1.
		if v, ok := readInt(d + "speed"); ok && v > 0 {
			nic.SpeedMbps = int(v)
			nic.Duplex = readStr(d + "duplex")
		}
		if v, ok := readInt(d + "mtu"); ok {
			nic.MTU = int(v)
		}
		c.r.Network = append(c.r.Network, nic)
	}
}

var asoundCard = regexp.MustCompile(`^\s*(\d+)\s+\[(.*?)\s*\]:\s*(.*?)\s+-\s+(.*)$`)

func (c *collector) audio() {
	c.r.Audio = []report.SoundCard{}
	f, err := os.Open(p("/proc/asound/cards"))
	if err != nil {
		return // no ALSA (headless server, container)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		m := asoundCard.FindStringSubmatch(sc.Text())
		if m == nil {
			continue
		}
		idx, _ := strconv.Atoi(m[1])
		card := report.SoundCard{Index: idx, ID: m[2], Driver: m[3], Name: m[4]}
		card.Bus, card.BusAddress = busOf("/sys/class/sound/card" + m[1] + "/device")
		c.r.Audio = append(c.r.Audio, card)
	}
}

func (c *collector) batteries() {
	c.r.Batteries = []report.Battery{}
	for _, n := range list("/sys/class/power_supply") {
		d := "/sys/class/power_supply/" + n + "/"
		// scope=Device marks peripheral batteries (mice, headsets).
		if readStr(d+"type") != "Battery" || readStr(d+"scope") == "Device" {
			continue
		}
		b := report.Battery{
			Name:         n,
			Manufacturer: readStr(d + "manufacturer"),
			Model:        readStr(d + "model_name"),
			Serial:       strings.TrimSpace(readStr(d + "serial_number")),
			Technology:   readStr(d + "technology"),
			Status:       readStr(d + "status"),
		}
		if v, ok := readInt(d + "capacity"); ok {
			b.CapacityPercent = int(v)
		}
		if v, ok := readInt(d + "cycle_count"); ok {
			b.CycleCount = int(v)
		}
		// Energy in µWh, or charge in µAh × design voltage in µV.
		design, full := float64(readUint(d+"energy_full_design")), float64(readUint(d+"energy_full"))
		if design == 0 {
			volts := float64(readUint(d+"voltage_min_design")) / 1e6
			design = float64(readUint(d+"charge_full_design")) * volts
			full = float64(readUint(d+"charge_full")) * volts
		}
		b.DesignWh = round(design/1e6, 2)
		b.FullWh = round(full/1e6, 2)
		if design > 0 {
			b.HealthPercent = round(full/design*100, 1)
		}
		c.r.Batteries = append(c.r.Batteries, b)
	}
}

func round(v float64, places int) float64 {
	m := math.Pow(10, float64(places))
	return math.Round(v*m) / m
}

var hwmonInput = regexp.MustCompile(`^(temp|fan|in|power|curr)(\d+)_(input|average)$`)

// sensors takes one reading of every hwmon sensor.
func (c *collector) sensors() {
	c.r.Sensors = []report.Sensor{}
	for _, h := range list("/sys/class/hwmon") {
		d := "/sys/class/hwmon/" + h + "/"
		s := report.Sensor{Chip: readStr(d + "name"), Readings: []report.SensorReading{}}
		_, s.Device = busOf(d + "device")
		seen := map[string]bool{}
		for _, f := range list(d) {
			m := hwmonInput.FindStringSubmatch(f)
			if m == nil || seen[m[1]+m[2]] {
				continue
			}
			seen[m[1]+m[2]] = true
			raw, ok := readInt(d + f)
			if !ok {
				continue
			}
			prefix := d + m[1] + m[2] + "_"
			label := readStr(prefix + "label")
			if label == "" {
				label = m[1] + m[2]
			}
			r := report.SensorReading{Label: label}
			scale := 1.0
			switch m[1] {
			case "temp":
				r.Kind, r.Unit, scale = "temperature", "C", 1000
			case "fan":
				r.Kind, r.Unit = "fan", "RPM"
			case "in":
				r.Kind, r.Unit, scale = "voltage", "V", 1000
			case "power":
				r.Kind, r.Unit, scale = "power", "W", 1e6
			case "curr":
				r.Kind, r.Unit, scale = "current", "A", 1000
			}
			r.Value = round(float64(raw)/scale, 3)
			if v, ok := readInt(prefix + "max"); ok {
				r.Max = round(float64(v)/scale, 3)
			}
			if v, ok := readInt(prefix + "crit"); ok {
				r.Crit = round(float64(v)/scale, 3)
			}
			s.Readings = append(s.Readings, r)
		}
		if len(s.Readings) > 0 {
			c.r.Sensors = append(c.r.Sensors, s)
		}
	}
}

// usb lists USB devices (not interfaces, not root hubs).
func (c *collector) usb() {
	c.r.USB = []report.USBDevice{}
	const dir = "/sys/bus/usb/devices/"
	for _, n := range list(dir) {
		if strings.Contains(n, ":") || strings.HasPrefix(n, "usb") {
			continue
		}
		d := dir + n + "/"
		vid, pid := readStr(d+"idVendor"), readStr(d+"idProduct")
		// The device's own strings; resolve replaces them with database
		// names where it has them, which are usually more consistent.
		dev := report.USBDevice{
			Path:       n,
			VendorID:   vid,
			ProductID:  pid,
			Vendor:     readStr(d + "manufacturer"),
			Product:    readStr(d + "product"),
			Serial:     readStr(d + "serial"),
			USBVersion: strings.TrimSpace(readStr(d + "version")),
		}
		if v, ok := readInt(d + "busnum"); ok {
			dev.Bus = int(v)
		}
		if v, ok := readInt(d + "devnum"); ok {
			dev.Device = int(v)
		}
		if v, err := strconv.ParseFloat(readStr(d+"speed"), 64); err == nil {
			dev.SpeedMbps = v
		}
		class := readStr(d + "bDeviceClass")
		seenDrv := map[string]bool{}
		for _, iface := range list(d) {
			if !strings.HasPrefix(iface, n+":") {
				continue
			}
			if class == "00" || class == "" {
				class = readStr(d + iface + "/bInterfaceClass")
			}
			if drv := linkBase(d + iface + "/driver"); drv != "" && !seenDrv[drv] {
				seenDrv[drv] = true
				dev.Drivers = append(dev.Drivers, drv)
			}
		}
		dev.ClassCode = class
		c.r.USB = append(c.r.USB, dev)
	}
}
