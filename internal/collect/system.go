package collect

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/jiegui2025/hwspec/internal/smbios"
)

func (c *collector) osInfo() {
	o := &c.r.OS
	rel := parseOSRelease(readStr("/etc/os-release"))
	if len(rel) == 0 {
		rel = parseOSRelease(readStr("/usr/lib/os-release"))
	}
	if len(rel) == 0 {
		c.warn("os-release: not found")
	}
	o.Name = rel["NAME"]
	o.ID = rel["ID"]
	o.IDLike = rel["ID_LIKE"]
	o.Version = rel["VERSION_ID"]
	o.PrettyName = rel["PRETTY_NAME"]

	var u syscall.Utsname
	if err := syscall.Uname(&u); err == nil {
		o.Kernel = utsString(u.Release[:])
		o.Arch = utsString(u.Machine[:])
	}
	o.Init = readStr("/proc/1/comm")

	o.Virtualization = "none"
	switch {
	case exists("/run/.containerenv"), exists("/.dockerenv"), readStr("/run/systemd/container") != "":
		o.Virtualization = "container"
	case strings.Contains(" "+cpuinfoField("flags")+" ", " hypervisor "):
		o.Virtualization = "vm"
	}

	o.BootMode = "bios"
	if exists("/sys/firmware/efi") {
		o.BootMode = "uefi"
		// efivars data = 4 attribute bytes + 1 value byte.
		b, err := os.ReadFile(p("/sys/firmware/efi/efivars/SecureBoot-8be4df61-93ca-11d2-aa0d-00e098032b8c"))
		if err == nil && len(b) >= 5 {
			on := b[4] == 1
			o.SecureBoot = &on
		}
	}
}

func parseOSRelease(s string) map[string]string {
	m := map[string]string{}
	for _, line := range strings.Split(s, "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok || strings.HasPrefix(k, "#") {
			continue
		}
		if uq, err := strconv.Unquote(v); err == nil {
			v = uq
		} else {
			v = strings.Trim(v, `"'`)
		}
		m[k] = v
	}
	return m
}

// utsString converts a NUL-terminated Utsname field (int8 or uint8
// depending on the architecture) to a string.
func utsString[T int8 | uint8](f []T) string {
	b := make([]byte, 0, len(f))
	for _, ch := range f {
		if ch == 0 {
			break
		}
		b = append(b, byte(ch))
	}
	return string(b)
}

const dmiDir = "/sys/class/dmi/id/"

func (c *collector) dmi() {
	if !exists(dmiDir) {
		c.warn("dmi: /sys/class/dmi/id not present (no SMBIOS firmware tables, common on ARM boards)")
		return
	}
	var denied []string
	get := func(name string) string {
		v, err := readStrErr(dmiDir + name)
		if err != nil {
			if os.IsPermission(err) {
				denied = append(denied, name)
			}
			return ""
		}
		return smbios.Clean(v)
	}

	s := &c.r.System
	s.Vendor = get("sys_vendor")
	s.Product = get("product_name")
	s.Version = get("product_version")
	s.Family = get("product_family")
	s.SKU = get("product_sku")
	s.Serial = get("product_serial")
	s.UUID = get("product_uuid")
	s.ChassisVendor = get("chassis_vendor")
	s.ChassisSerial = get("chassis_serial")
	if n, err := strconv.Atoi(get("chassis_type")); err == nil {
		s.ChassisType = smbios.ChassisType(n)
	}

	b := &c.r.Board
	b.Vendor = get("board_vendor")
	b.Product = get("board_name")
	b.Version = get("board_version")
	b.Serial = get("board_serial")
	b.AssetTag = get("board_asset_tag")

	bi := &c.r.BIOS
	bi.Vendor = get("bios_vendor")
	bi.Version = get("bios_version")
	bi.Date = get("bios_date")
	bi.Release = get("bios_release")

	if len(denied) > 0 {
		c.warn("dmi %s: needs root (run with --full)", strings.Join(denied, ", "))
	}
}

var cpuinfoCache map[string]string

// cpuinfoField returns a field from the first processor block of /proc/cpuinfo.
func cpuinfoField(key string) string {
	if cpuinfoCache == nil {
		cpuinfoCache = map[string]string{}
		f, err := os.Open(p("/proc/cpuinfo"))
		if err != nil {
			return ""
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 64*1024), 1024*1024)
		for sc.Scan() {
			line := sc.Text()
			if line == "" {
				break // end of the first processor block
			}
			k, v, ok := strings.Cut(line, ":")
			if ok {
				cpuinfoCache[strings.TrimSpace(k)] = strings.TrimSpace(v)
			}
		}
	}
	return cpuinfoCache[key]
}
