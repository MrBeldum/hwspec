package collect

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// root is prefixed to every path, so tests can point it at a fake tree.
var root = "/"

func p(path string) string { return filepath.Join(root, path) }

// readStr returns the trimmed file contents, or "" if it can't be read.
func readStr(path string) string {
	b, err := os.ReadFile(p(path))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// readStrErr is readStr but keeps the error, for fields where a failure is
// worth reporting (usually permission denied on root-only files).
func readStrErr(path string) (string, error) {
	b, err := os.ReadFile(p(path))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func readInt(path string) (int64, bool) {
	s := readStr(path)
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseInt(s, 0, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func readUint(path string) uint64 {
	v, err := strconv.ParseUint(readStr(path), 10, 64)
	if err != nil {
		return 0
	}
	return v
}

// linkBase returns the last element of a symlink target, e.g. the driver
// name from .../device/driver.
func linkBase(path string) string {
	t, err := os.Readlink(p(path))
	if err != nil {
		return ""
	}
	return filepath.Base(t)
}

// list returns the sorted entry names of a directory (nil if missing).
func list(dir string) []string {
	entries, err := os.ReadDir(p(dir))
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names
}

// realPath resolves all symlinks, returning "" on error.
func realPath(path string) string {
	r, err := filepath.EvalSymlinks(p(path))
	if err != nil {
		return ""
	}
	return r
}

func exists(path string) bool {
	_, err := os.Stat(p(path))
	return err == nil
}

// hex4 normalises "0x8086" to "8086".
func hex4(s string) string {
	return strings.ToLower(strings.TrimPrefix(s, "0x"))
}

// busOf resolves a sysfs device symlink (e.g. /sys/class/net/eth0/device)
// to its bus ("pci", "usb", ...) and address.
func busOf(deviceLink string) (bus, addr string) {
	target, err := filepath.EvalSymlinks(p(deviceLink))
	if err != nil {
		return "", ""
	}
	sub, err := filepath.EvalSymlinks(filepath.Join(target, "subsystem"))
	if err != nil {
		return "", filepath.Base(target)
	}
	bus = filepath.Base(sub)
	addr = filepath.Base(target)
	// A USB interface (1-2:1.0) belongs to the USB device one level up (1-2).
	if bus == "usb" && strings.Contains(addr, ":") {
		addr = filepath.Base(filepath.Dir(target))
	}
	return bus, addr
}
