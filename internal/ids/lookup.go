package ids

import (
	"fmt"
	"strings"
)

// Lookup resolves one ID given the way a person would type it, for the
// `hwspec ids lookup` command. It uses the same key forms as the overrides
// file (see OverridesHelp). It returns the name and the database key used.
func Lookup(kind Kind, id string) (name, key string, err error) {
	id = strings.TrimSpace(id)
	switch kind {
	case PCI, USB:
		if code, isClass := strings.CutPrefix(strings.ToLower(id), "class "); isClass {
			key = "class:" + norm(code)
			if kind == PCI {
				return PCIClass(code), key, nil
			}
			return USBClass(code), key, nil
		}
		key = norm(id)
		return get(kind).names[key], key, nil
	case PNP:
		key = strings.ToUpper(id)
		return PNPVendor(id), key, nil
	case OUI:
		prefix, ok := ouiPrefix(id)
		if !ok {
			return "", "", fmt.Errorf("%q is not a MAC address or prefix", id)
		}
		return get(OUI).names[prefix], prefix, nil
	case JEDEC:
		if strings.Contains(id, ":") {
			_, k, _, err := parseOverrideLine("jedec " + id + " = x")
			if err != nil {
				return "", "", err
			}
			return get(JEDEC).names[k], k, nil
		}
		if n, k, ok := MemoryManufacturer(id); ok {
			return n, k, nil
		}
		if c := jedecCandidates(id); len(c) > 0 {
			return "", strings.Join(c, " or "), nil
		}
		return "", "", fmt.Errorf("%q is not a JEDEC code", id)
	case AMDGPU:
		key = norm(id)
		return get(AMDGPU).names[key], key, nil
	}
	return "", "", fmt.Errorf("unknown database %q (want one of %s)", kind, kindList())
}

func kindList() string {
	var s []string
	for _, k := range Kinds {
		s = append(s, string(k))
	}
	return strings.Join(s, ", ")
}
