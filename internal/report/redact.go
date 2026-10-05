package report

import (
	"regexp"
	"strings"
)

// systemMounts are mount points that say nothing about the user.
var systemMounts = map[string]bool{
	"/": true, "/boot": true, "/boot/efi": true, "/efi": true, "/home": true, "/var": true,
	"/tmp": true, "/usr": true, "/opt": true, "/srv": true, "/nix": true, "/nix/store": true, "[SWAP]": true,
}

// macInterfaceName matches names udev derives from the MAC address
// (enx001122334455, wlx…), which would leak the redacted MAC.
var macInterfaceName = regexp.MustCompile(`^(enx|wlx)[0-9a-f]{12}$`)

// Redact clears identifiers that tie a report to one physical machine or
// person (serial numbers, UUIDs, MAC addresses, hostname), so the file can be
// shared publicly. Models, versions and sizes are kept.
func (r *Report) Redact() {
	r.Redacted = true
	r.Hostname = ""
	r.System.Serial, r.System.UUID, r.System.ChassisSerial = "", "", ""
	r.Board.Serial, r.Board.AssetTag = "", ""
	for i := range r.Memory.Modules {
		r.Memory.Modules[i].Serial = ""
	}
	for i := range r.Storage {
		d := &r.Storage[i]
		d.Serial, d.WWN = "", ""
		for j := range d.Partitions {
			p := &d.Partitions[j]
			p.UUID, p.Label = "", ""
			// /home/alice, /run/media/alice/Backup: user names and labels.
			if !systemMounts[p.MountPoint] {
				p.MountPoint = ""
			}
		}
	}
	for i := range r.Displays {
		r.Displays[i].Serial = ""
	}
	for i := range r.Network {
		n := &r.Network[i]
		n.MAC = ""
		if m := macInterfaceName.FindStringSubmatch(n.Name); m != nil {
			n.Name = m[1] + strings.Repeat("x", 12)
		}
	}
	for i := range r.Bluetooth {
		// The local name usually defaults to the hostname.
		r.Bluetooth[i].Address, r.Bluetooth[i].LocalName = "", ""
	}
	for i := range r.Batteries {
		r.Batteries[i].Serial = ""
	}
	for i := range r.USB {
		r.USB[i].Serial = ""
	}
}
