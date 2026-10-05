package collect

import (
	"sort"
	"strconv"
	"strings"

	"github.com/jaypipes/ghw"

	"github.com/jiegui2025/hwspec/internal/report"
)

const cpuDir = "/sys/devices/system/cpu/"

func (c *collector) cpu() {
	out := &c.r.CPU
	info, err := ghw.CPU(ghw.WithDisableWarnings(), ghw.WithDisableTools())
	if err != nil {
		c.warn("cpu: %v", err)
	} else {
		out.Sockets = len(info.Processors)
		out.Cores = int(info.TotalCores)
		out.Threads = int(info.TotalHardwareThreads)
		if len(info.Processors) > 0 {
			p0 := info.Processors[0]
			out.Model = p0.Model
			out.Vendor = p0.Vendor
			out.Flags = p0.Capabilities
		}
	}
	if out.Model == "" {
		// ARM kernels often have no "model name". "Processor" (older arm32)
		// and "Hardware" (the SoC) describe the CPU; "Model" is the board,
		// which belongs to System, so it isn't used here.
		for _, k := range []string{"model name", "cpu model", "Processor", "Hardware"} {
			if v := c.cpuinfoField(k); v != "" {
				out.Model = v
				break
			}
		}
	}
	if out.Vendor == "" {
		out.Vendor = c.cpuinfoField("vendor_id")
	}
	out.Microcode = c.cpuinfoField("microcode")
	out.Family, _ = strconv.Atoi(c.cpuinfoField("cpu family"))
	out.ModelID, _ = strconv.Atoi(c.cpuinfoField("model"))
	out.Stepping, _ = strconv.Atoi(c.cpuinfoField("stepping"))
	if out.Flags == nil {
		out.Flags = strings.Fields(c.cpuinfoField("flags"))
	}
	sort.Strings(out.Flags)
	for _, f := range out.Flags {
		if f == "vmx" || f == "svm" {
			out.Virtualization = f
		}
	}

	if khz, ok := readInt32(cpuDir + "cpu0/cpufreq/cpuinfo_min_freq"); ok {
		out.MinFreqMHz = khz / 1000
	}
	// On hybrid CPUs cpu0 may be an E-core, so take the highest max.
	for _, cpu := range cpuDirs() {
		if khz, ok := readInt32(cpuDir + cpu + "/cpufreq/cpuinfo_max_freq"); ok && khz/1000 > out.MaxFreqMHz {
			out.MaxFreqMHz = khz / 1000
		}
	}
	out.ScalingDriver = readStr(cpuDir + "cpu0/cpufreq/scaling_driver")
	out.Governor = readStr(cpuDir + "cpu0/cpufreq/scaling_governor")

	// Intel hybrid: the kernel registers separate PMUs for each core type.
	for _, t := range []struct{ pmu, name string }{
		{"cpu_core", "performance"},
		{"cpu_atom", "efficiency"},
	} {
		if list := readStr("/sys/devices/" + t.pmu + "/cpus"); list != "" {
			out.CoreTypes = append(out.CoreTypes, report.CoreType{Name: t.name, Threads: countCPUList(list)})
		}
	}

	out.Caches = c.caches()
}

func cpuDirs() []string {
	var out []string
	for _, n := range list(cpuDir) {
		if strings.HasPrefix(n, "cpu") {
			if _, err := strconv.Atoi(n[3:]); err == nil {
				out = append(out, n)
			}
		}
	}
	return out
}

// caches walks every CPU's cache entries and counts each distinct cache
// once (caches are shared, e.g. L3 by all cores, so they appear under
// several CPUs with the same shared_cpu_list).
func (c *collector) caches() []report.Cache {
	type key struct {
		level int
		typ   string
		size  uint64
	}
	seen := map[string]bool{}
	counts := map[key]int{}
	for _, cpu := range cpuDirs() {
		base := cpuDir + cpu + "/cache/"
		for _, idx := range list(base) {
			if !strings.HasPrefix(idx, "index") {
				continue
			}
			d := base + idx + "/"
			level, _ := readInt32(d + "level")
			typ := readStr(d + "type")
			size := parseSize(readStr(d + "size"))
			id := strings.Join([]string{strconv.Itoa(level), typ, readStr(d + "shared_cpu_list")}, "|")
			if seen[id] || size == 0 {
				continue
			}
			seen[id] = true
			counts[key{level, typ, size}]++
		}
	}
	out := []report.Cache{}
	for k, n := range counts {
		out = append(out, report.Cache{Level: k.level, Type: k.typ, SizeBytes: k.size, Instances: n})
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Level != b.Level {
			return a.Level < b.Level
		}
		if a.Type != b.Type {
			return a.Type < b.Type
		}
		return a.SizeBytes > b.SizeBytes
	})
	return out
}

// parseSize handles sysfs cache sizes like "32K", "1024K", "16M".
func parseSize(s string) uint64 {
	if s == "" {
		return 0
	}
	mult := uint64(1)
	switch s[len(s)-1] {
	case 'K':
		mult, s = 1<<10, s[:len(s)-1]
	case 'M':
		mult, s = 1<<20, s[:len(s)-1]
	case 'G':
		mult, s = 1<<30, s[:len(s)-1]
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return v * mult
}

// countCPUList counts the CPUs in a kernel cpu list like "0-11,16,18-19".
func countCPUList(s string) int {
	n := 0
	for _, part := range strings.Split(strings.TrimSpace(s), ",") {
		if part == "" {
			continue
		}
		lo, hi, isRange := strings.Cut(part, "-")
		if !isRange {
			n++
			continue
		}
		a, err1 := strconv.Atoi(lo)
		b, err2 := strconv.Atoi(hi)
		if err1 == nil && err2 == nil && b >= a {
			n += b - a + 1
		}
	}
	return n
}
