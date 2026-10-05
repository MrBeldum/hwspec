package trust

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestUserOwnedFilesAreNotTrusted(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: every file is root-owned")
	}
	f := filepath.Join(t.TempDir(), "hwspec")
	if err := os.WriteFile(f, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := RootOwned(f); err == nil {
		t.Error("a user-owned file was trusted")
	}
	// A symlink to it doesn't launder it.
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(f, link); err != nil {
		t.Fatal(err)
	}
	if err := RootOwned(link); err == nil {
		t.Error("a symlink to a user-owned file was trusted")
	}
}

func TestSystemFilesAreTrusted(t *testing.T) {
	for _, sys := range []string{"/usr/bin/env", "/bin/sh"} {
		st, err := os.Stat(sys)
		if err != nil {
			continue
		}
		// Build sandboxes (Nix) give system files to the build user.
		if owner, ok := st.Sys().(*syscall.Stat_t); !ok || owner.Uid != 0 {
			t.Skipf("%s isn't root-owned here (build sandbox)", sys)
		}
		if err := RootOwned(sys); err != nil {
			t.Errorf("%s: %v", sys, err)
		}
		return
	}
}

// /nix/store is root-owned, group-writable and sticky; files under it
// can't be replaced by the group, so Nix-installed hwspec must be trusted.
func TestStickyWritableDirectoriesAreTrusted(t *testing.T) {
	if _, err := os.Stat("/nix/store"); err != nil {
		t.Skip("no /nix/store on this system")
	}
	entries, _ := filepath.Glob("/nix/store/*-coreutils-*/bin/env")
	if len(entries) == 0 {
		t.Skip("no coreutils in /nix/store")
	}
	if err := RootOwned(entries[0]); err != nil {
		t.Errorf("%s: %v", entries[0], err)
	}
}
