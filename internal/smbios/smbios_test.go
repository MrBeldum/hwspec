package smbios

import (
	"encoding/binary"
	"testing"
)

func structure(typ byte, length int, set func(f []byte), strs ...string) []byte {
	f := make([]byte, length)
	f[0], f[1] = typ, byte(length)
	set(f)
	out := f
	for _, s := range strs {
		out = append(out, s...)
		out = append(out, 0)
	}
	if len(strs) == 0 {
		out = append(out, 0)
	}
	return append(out, 0)
}

func testTable() []byte {
	var t []byte
	// Type 16: 2 slots, 64 GiB max, no ECC.
	t = append(t, structure(16, 0x17, func(f []byte) {
		f[0x05] = 0x03
		f[0x06] = 0x03
		binary.LittleEndian.PutUint32(f[0x07:], 64<<20) // KiB
		binary.LittleEndian.PutUint16(f[0x0D:], 2)
	})...)
	// Type 17: 16 GiB DDR4 SODIMM.
	t = append(t, structure(17, 0x28, func(f []byte) {
		binary.LittleEndian.PutUint16(f[0x08:], 64)
		binary.LittleEndian.PutUint16(f[0x0A:], 64)
		binary.LittleEndian.PutUint16(f[0x0C:], 16384) // MiB
		f[0x0E] = 0x0D
		f[0x10], f[0x11] = 1, 2
		f[0x12] = 0x1A
		binary.LittleEndian.PutUint16(f[0x15:], 2667)
		f[0x17], f[0x18], f[0x1A] = 3, 4, 5
		f[0x1B] = 2
		binary.LittleEndian.PutUint16(f[0x20:], 2400)
		binary.LittleEndian.PutUint16(f[0x26:], 1200)
	}, "DIMM 1", "BANK 0", "Samsung", "12345678", "M471A2K43DB1-CTD  ")...)
	// Type 17: empty slot with placeholder strings.
	t = append(t, structure(17, 0x28, func(f []byte) {
		f[0x10], f[0x17] = 1, 2
	}, "DIMM 2", "Not Specified")...)
	t = append(t, structure(127, 4, func([]byte) {})...)
	return t
}

func TestMemory(t *testing.T) {
	structs := Parse(testTable())
	if len(structs) != 4 {
		t.Fatalf("parsed %d structures, want 4", len(structs))
	}

	arrays := MemoryArrays(structs)
	if len(arrays) != 1 || arrays[0].Slots != 2 || arrays[0].MaxCapacityBytes != 64<<30 || arrays[0].ErrorCorrection != "None" {
		t.Errorf("arrays = %+v", arrays)
	}

	devs := MemoryDevices(structs)
	if len(devs) != 2 {
		t.Fatalf("got %d devices, want 2", len(devs))
	}
	want := MemoryDevice{
		Locator: "DIMM 1", BankLocator: "BANK 0", SizeBytes: 16 << 30,
		FormFactor: "SODIMM", Type: "DDR4", SpeedMTs: 2667, ConfiguredMTs: 2400,
		Manufacturer: "Samsung", Serial: "12345678", PartNumber: "M471A2K43DB1-CTD",
		TotalWidth: 64, DataWidth: 64, Rank: 2, ConfiguredMV: 1200,
	}
	if devs[0] != want {
		t.Errorf("device 0:\n got %+v\nwant %+v", devs[0], want)
	}
	if devs[1].SizeBytes != 0 || devs[1].Manufacturer != "" || devs[1].Locator != "DIMM 2" {
		t.Errorf("empty slot = %+v", devs[1])
	}
}

func TestParseTruncated(t *testing.T) {
	table := testTable()
	for n := 0; n < len(table); n++ {
		Parse(table[:n]) // must not panic
		MemoryDevices(Parse(table[:n]))
	}
}

func TestChassisType(t *testing.T) {
	for n, want := range map[int]string{3: "Desktop", 10: "Notebook", 0x23: "Mini PC", 0x89: "Laptop", 0: "", 99: ""} {
		if got := ChassisType(n); got != want {
			t.Errorf("ChassisType(%d) = %q, want %q", n, got, want)
		}
	}
}
