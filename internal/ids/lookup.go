package ids

import "strings"

// Lookup resolves one ID given the way a person would type it, for the
// `hwspec ids lookup` command. It accepts the same spellings as the
// overrides file (see NormalizeKey). It returns the name and the database
// key it looked up; a JEDEC code is tried in every plausible byte order.
func Lookup(kind Kind, id string) (name, key string, err error) {
	if kind == JEDEC && !strings.Contains(id, ":") {
		if n, k, ok := MemoryManufacturer(id); ok {
			return n, k, nil
		}
	}
	key, err = NormalizeKey(kind, id)
	if err != nil {
		return "", "", err
	}
	if kind == PCI && strings.HasPrefix(key, "class:") {
		return PCIClass(strings.TrimPrefix(key, "class:")), key, nil
	}
	name = get(kind).names[key]
	if kind == CPU {
		if name == "" && strings.Count(key, ":") == 3 {
			name = get(kind).names[key[:strings.LastIndex(key, ":")]] // no stepping-specific entry
		}
		name = strings.TrimSuffix(strings.ReplaceAll(name, "\t", " | "), " | ")
	}
	return name, key, nil
}

func kindList() string {
	var s []string
	for _, k := range Kinds {
		s = append(s, string(k))
	}
	return strings.Join(s, ", ")
}
