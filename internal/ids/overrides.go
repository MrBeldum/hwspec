package ids

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
)

// OverridesHelp documents the overrides file format.
const OverridesHelp = `# hwspec name overrides: one "KIND KEY = Name" per line; later lines win.
#
#   pci 8086 = Intel                       PCI vendor
#   pci 8086:3e92 = UHD Graphics 630       PCI device
#   pci 8086:3e92:103c:8595 = HP iGPU      PCI subsystem (vendor:device:subvendor:subdevice)
#   pci class 0300 = VGA controller        PCI class or subclass
#   usb 046d = Logitech                    USB vendor
#   usb 046d:c52b = Unifying Receiver      USB product
#   usb class 03 = HID                     USB class
#   pnp DEL = Dell                         monitor manufacturer (EDID)
#   oui 04:0E:3C = HP Inc.                 MAC address prefix
#   jedec F785 = Avant Technology          memory maker, as the firmware reports it
#   jedec 6:77 = Avant Technology          memory maker, as JEP106 bank:ID (hex)
#   amdgpu 1114:c2 = Radeon 860M           AMD GPU device:revision
#   bluetooth 0002 = Intel                 Bluetooth SIG company ID (hex)
#   cpu intel:6:9e:10 = Coffee Lake | Skylake   CPU vendor:family:model[:stepping]
#                                          (hex, stepping decimal) = codename | core
`

var (
	ovMu    sync.Mutex
	ovCache = map[string]ovResult{}
)

type ovResult struct {
	m   map[Kind]map[string]string
	err error
}

// loadOverrides parses the overrides file. A missing file is not an error.
// Malformed lines are skipped and reported together in the error, while the
// valid lines still apply.
func loadOverrides(path string) (map[Kind]map[string]string, error) {
	if path == "" {
		return nil, nil
	}
	ovMu.Lock()
	defer ovMu.Unlock()
	if r, ok := ovCache[path]; ok {
		return r.m, r.err
	}
	m, err := parseOverrides(path)
	ovCache[path] = ovResult{m, err}
	return m, err
}

func parseOverrides(path string) (map[Kind]map[string]string, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	out := map[Kind]map[string]string{}
	var bad []string
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		kind, key, name, err := parseOverrideLine(line)
		if err != nil {
			bad = append(bad, fmt.Sprintf("line %d: %v", n, err))
			continue
		}
		if out[kind] == nil {
			out[kind] = map[string]string{}
		}
		out[kind][key] = name
	}
	if err := sc.Err(); err != nil {
		bad = append(bad, err.Error())
	}
	if len(bad) > 0 {
		return out, errors.New(strings.Join(bad, "; "))
	}
	return out, nil
}

// parseOverrideLine turns "pci 8086:3e92 = Name" into the key the
// matching database uses.
func parseOverrideLine(line string) (Kind, string, string, error) {
	lhs, name, ok := strings.Cut(line, "=")
	name = cleanName(name)
	if !ok || name == "" {
		return "", "", "", errors.New(`expected "KIND KEY = Name"`)
	}
	kind, id, ok := strings.Cut(strings.TrimSpace(lhs), " ")
	if !ok || strings.TrimSpace(id) == "" {
		return "", "", "", errors.New(`expected "KIND KEY = Name"`)
	}
	k := Kind(strings.ToLower(kind))
	key, err := NormalizeKey(k, id)
	if err != nil {
		return "", "", "", err
	}
	if k == CPU {
		codename, uarch, _ := strings.Cut(name, "|")
		name = strings.TrimSpace(codename) + "\t" + strings.TrimSpace(uarch)
	}
	return k, key, name, nil
}
