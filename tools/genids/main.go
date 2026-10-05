// Command genids converts upstream ID sources into the compact, gzipped
// files embedded in internal/ids/data. Run through `make update-ids`.
//
//	genids jedec <decode-dimms> <out.gz>   JEDEC JEP106 table from i2c-tools
//	genids oui   <oui.txt>      <out.gz>   IEEE MA-L registry
//	genids gzip  <file>         <out.gz>   any file as-is (pci/usb/pnp/amdgpu)
package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "usage: genids jedec|oui|gzip <source> <out.gz>")
		os.Exit(2)
	}
	src, err := os.ReadFile(os.Args[2])
	if err != nil {
		fail(err)
	}
	var out []byte
	switch os.Args[1] {
	case "jedec":
		out, err = jedec(src)
	case "oui":
		out, err = oui(src)
	case "gzip":
		out = src
	default:
		err = fmt.Errorf("unknown kind %q", os.Args[1])
	}
	if err != nil {
		fail(err)
	}
	var buf bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	zw.Write(out)
	zw.Close()
	if err := os.WriteFile(os.Args[3], buf.Bytes(), 0o644); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "genids:", err)
	os.Exit(1)
}

var (
	perlPage   = regexp.MustCompile(`(?s)\[(.*?)\]`)
	perlString = regexp.MustCompile(`"((?:[^"\\]|\\.)*)"`)
)

// jedec extracts the @vendors array of arrays from decode-dimms (one array
// per JEP106 bank, 126 names each) into "BANK ID<TAB>Name" lines, with the
// bank 1-based in decimal and the ID in hex without the parity bit.
func jedec(src []byte) ([]byte, error) {
	s := string(src)
	start := strings.Index(s, "@vendors = (")
	if start < 0 {
		return nil, fmt.Errorf("no @vendors table found")
	}
	s = s[start:]
	s = s[:strings.Index(s, ");")]
	var b strings.Builder
	fmt.Fprintf(&b, "# JEDEC JEP106 manufacturer IDs, from i2c-tools decode-dimms\n# Date: %s\n", time.Now().UTC().Format("2006-01-02"))
	n := 0
	for bank, page := range perlPage.FindAllStringSubmatch(s, -1) {
		for i, m := range perlString.FindAllStringSubmatch(page[1], -1) {
			name := strings.ReplaceAll(m[1], `\"`, `"`)
			fmt.Fprintf(&b, "%d %02X\t%s\n", bank+1, i+1, name)
			n++
		}
	}
	if n < 1000 {
		return nil, fmt.Errorf("only %d JEDEC entries parsed; source format changed?", n)
	}
	return []byte(b.String()), nil
}

// oui keeps the "(base 16)" lines of the IEEE registry as "XXXXXX<TAB>Name".
func oui(src []byte) ([]byte, error) {
	entries := map[string]string{}
	sc := bufio.NewScanner(bytes.NewReader(src))
	for sc.Scan() {
		prefix, name, ok := strings.Cut(sc.Text(), "(base 16)")
		if !ok {
			continue
		}
		if key := strings.TrimSpace(prefix); len(key) == 6 {
			entries[strings.ToUpper(key)] = strings.TrimSpace(name)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(entries) < 10000 {
		return nil, fmt.Errorf("only %d OUI entries parsed; source format changed?", len(entries))
	}
	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	fmt.Fprintf(&b, "# IEEE MA-L (OUI) assignments\n# Date: %s\n", time.Now().UTC().Format("2006-01-02"))
	for _, k := range keys {
		fmt.Fprintf(&b, "%s\t%s\n", k, entries[k])
	}
	return []byte(b.String()), nil
}
