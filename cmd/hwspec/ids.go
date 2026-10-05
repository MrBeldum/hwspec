package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/jiegui2025/hwspec/internal/ids"
)

func idsCmd(args []string) error {
	if len(args) == 0 {
		return idsStatus()
	}
	switch args[0] {
	case "lookup":
		if len(args) < 3 {
			return errors.New("usage: hwspec ids lookup KIND ID   (e.g. pci 8086:3e92, usb class 03, jedec F785)")
		}
		kind := ids.Kind(strings.ToLower(args[1]))
		id := strings.Join(args[2:], " ")
		name, key, err := ids.Lookup(kind, id)
		if err != nil {
			return err
		}
		if name == "" {
			fmt.Printf("%s %s: not found (key %s)\n", kind, id, key)
			os.Exit(1)
		}
		fmt.Printf("%s\t(%s %s)\n", name, kind, key)
		return nil
	case "template":
		fmt.Print(ids.OverridesHelp)
		return nil
	case "update":
		return idsUpdate(args[1:])
	}
	return fmt.Errorf("unknown ids subcommand %q (want update, lookup or template)", args[0])
}

func idsUpdate(args []string) error {
	fs := newFlags("ids update")
	var check, allowOlder bool
	url := os.Getenv("HWSPEC_IDS_URL")
	fs.BoolVar(&check, "check", false, "")
	fs.BoolVar(&allowOlder, "allow-older", false, "")
	fs.StringVar(&url, "url", url, "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if url == "" {
		url = ids.DefaultSyncURL
	}
	fmt.Fprintf(os.Stderr, "Checking %s\n", url)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	results, err := ids.Update(ctx, ids.UpdateOptions{
		BaseURL:    url,
		UserAgent:  "hwspec/" + fullVersion(),
		DryRun:     check,
		AllowOlder: allowOlder,
	})
	if err != nil {
		return fmt.Errorf("update failed, nothing changed: %w", err)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	changed := 0
	for _, r := range results {
		status := r.Status
		if check && status != "unchanged" {
			status = "would update"
		}
		if r.Status != "unchanged" {
			changed++
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d names\n", r.File, status, r.Date, r.Entries)
	}
	w.Flush()
	switch {
	case check:
		fmt.Printf("\n%d of %d databases have updates. Run `hwspec ids update` to install them.\n", changed, len(results))
	case changed == 0:
		fmt.Println("\nAlready up to date.")
	default:
		fmt.Printf("\nInstalled %d databases into %s.\n", changed, ids.SyncedDir())
	}
	return nil
}

func idsStatus() error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "DATABASE\tNAMES\tSOURCE (newest of embedded, distro, synced; then overrides)")
	for _, k := range ids.Kinds {
		var parts []string
		for _, l := range ids.Layers(k) {
			s := l.Source
			if l.Date != "" {
				s += " " + l.Date
			}
			if l.Source == ids.OverridesPath() {
				s = fmt.Sprintf("overrides (%d)", l.Entries)
			}
			if l.Err != "" {
				s += " (unusable: " + l.Err + ")"
			}
			parts = append(parts, s)
		}
		fmt.Fprintf(w, "%s\t%d\t%s\n", k, ids.Entries(k), strings.Join(parts, "  →  "))
	}
	w.Flush()

	if t := ids.SyncedAt(); t.IsZero() {
		fmt.Println("\nNot synced yet. `hwspec ids update` downloads the latest databases (signed, ~1 MB).")
	} else {
		fmt.Printf("\nSynced bundle: built %s, in %s\n", t.Format("2006-01-02"), ids.SyncedDir())
	}

	path := ids.OverridesPath()
	if _, err := os.Stat(path); err == nil {
		fmt.Printf("Overrides: %s\n", path)
	} else {
		fmt.Printf("No overrides file. To correct or add names, create %s\n(start from `hwspec ids template`).\n", path)
	}
	return nil
}
