package main

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestUnknownOutputExtensionIsAnError(t *testing.T) {
	cases := []struct{ format, path, want string }{
		{"", "spec.json", "json"}, {"", "spec.yml", "yaml"}, {"", "spec.txt", "text"},
		{"", "", "json"}, {"yaml", "spec.xml", "yaml"}, {"", "-", "json"},
	}
	for _, c := range cases {
		if got, err := pickFormat(c.format, c.path, "json"); err != nil || got != c.want {
			t.Errorf("pickFormat(%q, %q) = %q, %v; want %q", c.format, c.path, got, err, c.want)
		}
	}
	if _, err := pickFormat("", "spec.xml", "json"); err == nil || !strings.Contains(err.Error(), ".xml") {
		t.Errorf("spec.xml: err = %v, want an error naming .xml", err)
	}
	if _, err := pickFormat("csv", "", "json"); err == nil {
		t.Error("unknown -f accepted")
	}
}

// --full must not elevate a binary that a non-root user could replace.
func TestOnlyRootOwnedBinariesAreElevated(t *testing.T) {
	userOwned := filepath.Join(t.TempDir(), "hwspec")
	if err := os.WriteFile(userOwned, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if os.Geteuid() != 0 {
		if err := checkRootOwned(userOwned); err == nil {
			t.Error("a user-owned binary was accepted for elevation")
		}
	}
	// The positive case needs a real root-owned system file, which build
	// sandboxes (Nix) don't have: there /bin/sh belongs to the build user.
	for _, sys := range []string{"/usr/bin/env", "/bin/sh"} {
		st, err := os.Stat(sys)
		if err != nil {
			continue
		}
		if owner, ok := st.Sys().(*syscall.Stat_t); !ok || owner.Uid != 0 {
			t.Skipf("%s isn't root-owned here (build sandbox); positive case not testable", sys)
		}
		if err := checkRootOwned(sys); err != nil {
			t.Errorf("%s: %v", sys, err)
		}
		return
	}
}

func TestWritesAreAtomicAndPrivateWhenTheyHoldSerials(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "elsewhere")
	if err := os.WriteFile(target, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "spec.json")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := writeFileAtomic(link, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(target); string(b) != "keep" {
		t.Errorf("write followed the symlink and clobbered %s", target)
	}
	st, err := os.Lstat(link)
	if err != nil || st.Mode()&os.ModeSymlink != 0 || st.Mode().Perm() != 0o600 {
		t.Errorf("spec.json: mode %v, err %v; want a regular 0600 file", st.Mode(), err)
	}
}
