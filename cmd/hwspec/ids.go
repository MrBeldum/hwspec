package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

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
	}
	return fmt.Errorf("unknown ids subcommand %q (want lookup or template)", args[0])
}

func idsStatus() error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "DATABASE\tNAMES\tLAYERS (applied in order, later wins)")
	for _, k := range ids.Kinds {
		var parts []string
		for _, l := range ids.Layers(k) {
			s := l.Source
			if l.Date != "" {
				s += " " + l.Date
			}
			s += fmt.Sprintf(" [+%d]", l.Entries)
			if l.Err != "" {
				s += " ERROR: " + l.Err
			}
			parts = append(parts, s)
		}
		fmt.Fprintf(w, "%s\t%d\t%s\n", k, ids.Entries(k), strings.Join(parts, "  →  "))
	}
	w.Flush()

	path := ids.OverridesPath()
	if _, err := os.Stat(path); err == nil {
		fmt.Printf("\nOverrides: %s\n", path)
	} else {
		fmt.Printf("\nNo overrides file. To correct or add names, create %s\n(start from `hwspec ids template`).\n", path)
	}
	return nil
}
