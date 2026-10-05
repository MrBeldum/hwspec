package ids

import (
	"strconv"
	"strings"
)

// maxJEDECBank bounds plausible bank numbers. JEP106 has about 16 banks
// today; the margin leaves room for growth while still rejecting the wrong
// byte order (which usually yields a bank number in the 40s or above).
const maxJEDECBank = 32

// jedecCandidates interprets a memory manufacturer string from SMBIOS as a
// JEDEC JEP106 code and returns the possible "BANK:ID" keys, most likely
// first. Firmware writes these codes in several ways:
//
//	"80CE"                 continuation-count byte, then ID byte (Samsung)
//	"CE00", "2C00"         ID byte first, count without parity
//	"Unknown - [0xF785]"   HP: the two bytes swapped (Avant Technology)
//	"7F7F7F7F7F9B"         older SPD style: one 0x7F per bank skipped
//	"00CE000000000000"     padded with zeros
//
// A plain name like "Samsung" returns nil.
func jedecCandidates(raw string) []string {
	s := strings.ToUpper(raw)
	for _, junk := range []string{"UNKNOWN", "0X", "-", "[", "]", "(", ")", " ", "\t"} {
		s = strings.ReplaceAll(s, junk, "")
	}
	if len(s) < 2 || len(s)%2 != 0 {
		return nil
	}
	b := make([]int, len(s)/2)
	for i := range b {
		v, err := strconv.ParseUint(s[2*i:2*i+2], 16, 8)
		if err != nil {
			return nil
		}
		b[i] = int(v)
	}

	// Continuation codes: 0x7F per bank, then the ID.
	if b[0] == 0x7F {
		n := 0
		for n < len(b) && b[n] == 0x7F {
			n++
		}
		if n < len(b) && validID(b[n]) && n+1 <= maxJEDECBank {
			return []string{jedecKey(n+1, b[n]&0x7F)}
		}
		return nil
	}
	if len(b) == 1 {
		if validID(b[0]) {
			return []string{jedecKey(1, b[0]&0x7F)}
		}
		return nil
	}
	for _, v := range b[2:] {
		if v != 0 {
			return nil // not a zero-padded two-byte code
		}
	}
	var out []string
	for _, pair := range [][2]int{{b[0], b[1]}, {b[1], b[0]}} {
		count, id := pair[0]&0x7F, pair[1]
		if count+1 <= maxJEDECBank && validID(id) {
			out = append(out, jedecKey(count+1, id&0x7F))
		}
	}
	return out
}

func validID(b int) bool {
	id := b & 0x7F
	return id >= 1 && id <= 126
}

// MemoryManufacturer decodes a JEDEC code from an SMBIOS memory device's
// manufacturer field. It returns the name and the "BANK:ID" key used, or
// ok=false when the string isn't a known code.
func MemoryManufacturer(raw string) (name, key string, ok bool) {
	names := get(JEDEC).names
	for _, k := range jedecCandidates(raw) {
		if n, found := names[k]; found {
			return n, k, true
		}
	}
	return "", "", false
}

// ouiPrefix extracts the first three octets of a MAC address as "040E3C".
func ouiPrefix(mac string) (string, bool) {
	s := strings.ToUpper(strings.NewReplacer(":", "", "-", "", ".", "").Replace(strings.TrimSpace(mac)))
	if len(s) < 6 {
		return "", false
	}
	if _, err := strconv.ParseUint(s[:6], 16, 32); err != nil {
		return "", false
	}
	return s[:6], true
}

// MACVendor returns the registered owner of a MAC address's prefix. It
// returns "" for locally administered addresses (randomised Wi-Fi MACs,
// virtual adapters), which have no registered owner.
func MACVendor(mac string) string {
	prefix, ok := ouiPrefix(mac)
	if !ok {
		return ""
	}
	first, _ := strconv.ParseUint(prefix[:2], 16, 8)
	if first&0x02 != 0 {
		return ""
	}
	return get(OUI).names[prefix]
}
