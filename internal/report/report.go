// Package report defines the hwspec file format.
//
// Bump SchemaVersion when a field is renamed, removed or changes meaning.
// Adding fields is backwards compatible and doesn't need a bump.
package report

import "time"

const SchemaVersion = 1

type Report struct {
	SchemaVersion int       `json:"schema_version"`
	Tool          Tool      `json:"tool"`
	CapturedAt    time.Time `json:"captured_at"`
	Hostname      string    `json:"hostname"`
	// Privileged is true when the capture ran as root, so serial numbers,
	// memory modules (SMBIOS) and drive health could be read.
	Privileged bool `json:"privileged"`
	Redacted   bool `json:"redacted"`

	OS        OS                    `json:"os"`
	System    System                `json:"system"`
	Board     Board                 `json:"board"`
	BIOS      BIOS                  `json:"bios"`
	CPU       CPU                   `json:"cpu"`
	Memory    Memory                `json:"memory"`
	Storage   []Disk                `json:"storage"`
	GPUs      []GPU                 `json:"gpus"`
	Displays  []Display             `json:"displays"`
	Network   []NIC                 `json:"network"`
	Bluetooth []BluetoothController `json:"bluetooth"`
	Audio     []SoundCard           `json:"audio"`
	Batteries []Battery             `json:"batteries"`
	Sensors   []Sensor              `json:"sensors"`
	PCI       []PCIDevice           `json:"pci"`
	USB       []USBDevice           `json:"usb"`

	// Warnings lists what couldn't be read and why (e.g. permission denied).
	Warnings []string `json:"warnings"`
}

type Tool struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	// IDDatabases records, per ID database, the layers names came from
	// (e.g. "pci": "embedded (2026-06-26) + /usr/share/hwdata/pci.ids (2026-09-03)").
	IDDatabases map[string]string `json:"id_databases"`
}

type OS struct {
	Name           string `json:"name"`
	ID             string `json:"id"`
	IDLike         string `json:"id_like,omitempty"`
	Version        string `json:"version,omitempty"`
	PrettyName     string `json:"pretty_name"`
	Kernel         string `json:"kernel"`
	Arch           string `json:"arch"`
	Init           string `json:"init,omitempty"`
	Virtualization string `json:"virtualization"` // none, vm, container
	BootMode       string `json:"boot_mode"`      // uefi, bios
	SecureBoot     *bool  `json:"secure_boot,omitempty"`
}

type System struct {
	Vendor        string `json:"vendor"`
	Product       string `json:"product"`
	Version       string `json:"version,omitempty"`
	Family        string `json:"family,omitempty"`
	SKU           string `json:"sku,omitempty"`
	Serial        string `json:"serial,omitempty"`
	UUID          string `json:"uuid,omitempty"`
	ChassisType   string `json:"chassis_type"`
	ChassisVendor string `json:"chassis_vendor,omitempty"`
	ChassisSerial string `json:"chassis_serial,omitempty"`
}

type Board struct {
	Vendor   string `json:"vendor"`
	Product  string `json:"product"`
	Version  string `json:"version,omitempty"`
	Serial   string `json:"serial,omitempty"`
	AssetTag string `json:"asset_tag,omitempty"`
}

type BIOS struct {
	Vendor  string `json:"vendor"`
	Version string `json:"version"`
	Date    string `json:"date"`
	Release string `json:"release,omitempty"`
}

type CPU struct {
	Model  string `json:"model"`
	Vendor string `json:"vendor"`
	// x86 signature (family and model as the kernel reports them, decimal).
	Family   int `json:"family,omitempty"`
	ModelID  int `json:"model_id,omitempty"`
	Stepping int `json:"stepping,omitempty"`
	// Codename and Microarchitecture come from the cpu ID database.
	Codename          string `json:"codename,omitempty"`
	Microarchitecture string `json:"microarchitecture,omitempty"`
	Sockets           int    `json:"sockets"`
	Cores             int    `json:"cores"`
	Threads           int    `json:"threads"`
	// CoreTypes is set on hybrid CPUs (Intel P/E cores).
	CoreTypes      []CoreType `json:"core_types,omitempty"`
	MinFreqMHz     int        `json:"min_freq_mhz,omitempty"`
	MaxFreqMHz     int        `json:"max_freq_mhz,omitempty"`
	ScalingDriver  string     `json:"scaling_driver,omitempty"`
	Governor       string     `json:"governor,omitempty"`
	Microcode      string     `json:"microcode,omitempty"`
	Caches         []Cache    `json:"caches"`
	Virtualization string     `json:"virtualization,omitempty"` // vmx, svm
	Flags          []string   `json:"flags"`
}

type CoreType struct {
	Name    string `json:"name"` // performance, efficiency
	Threads int    `json:"threads"`
}

type Cache struct {
	Level     int    `json:"level"`
	Type      string `json:"type"` // Data, Instruction, Unified
	SizeBytes uint64 `json:"size_bytes"`
	// Instances is how many copies exist (e.g. one L2 per core).
	Instances int `json:"instances"`
}

type Memory struct {
	// TotalBytes is what the kernel can use (MemTotal), a little less than
	// InstalledBytes because firmware and the kernel reserve some.
	TotalBytes uint64 `json:"total_bytes"`
	SwapBytes  uint64 `json:"swap_bytes"`
	// From SMBIOS (root only).
	InstalledBytes   uint64         `json:"installed_bytes,omitempty"`
	MaxCapacityBytes uint64         `json:"max_capacity_bytes,omitempty"`
	Slots            int            `json:"slots,omitempty"`
	ECC              string         `json:"ecc,omitempty"`
	Modules          []MemoryModule `json:"modules"`
}

type MemoryModule struct {
	Locator       string `json:"locator"`
	BankLocator   string `json:"bank_locator,omitempty"`
	SizeBytes     uint64 `json:"size_bytes"`
	Type          string `json:"type,omitempty"`        // DDR4, DDR5, LPDDR5...
	FormFactor    string `json:"form_factor,omitempty"` // DIMM, SODIMM...
	SpeedMTs      int    `json:"speed_mts,omitempty"`
	ConfiguredMTs int    `json:"configured_speed_mts,omitempty"`
	Manufacturer  string `json:"manufacturer,omitempty"`
	// ManufacturerRaw keeps the firmware's string when Manufacturer was
	// decoded from a JEDEC code, e.g. "Unknown - [0xF785]".
	ManufacturerRaw string `json:"manufacturer_raw,omitempty"`
	PartNumber      string `json:"part_number,omitempty"`
	Serial          string `json:"serial,omitempty"`
	DataWidth       int    `json:"data_width_bits,omitempty"`
	TotalWidth      int    `json:"total_width_bits,omitempty"`
	Rank            int    `json:"rank,omitempty"`
	VoltageMV       int    `json:"configured_voltage_mv,omitempty"`
}

type Disk struct {
	Name               string      `json:"name"` // nvme0n1, sda
	Model              string      `json:"model"`
	Vendor             string      `json:"vendor,omitempty"`
	Serial             string      `json:"serial,omitempty"`
	WWN                string      `json:"wwn,omitempty"`
	Firmware           string      `json:"firmware,omitempty"`
	SizeBytes          uint64      `json:"size_bytes"`
	Type               string      `json:"type"`      // ssd, hdd, nvme, removable...
	Transport          string      `json:"transport"` // nvme, sata, usb, virtio, mmc...
	Rotational         bool        `json:"rotational"`
	Removable          bool        `json:"removable"`
	LogicalBlockBytes  uint64      `json:"logical_block_bytes,omitempty"`
	PhysicalBlockBytes uint64      `json:"physical_block_bytes,omitempty"`
	Partitions         []Partition `json:"partitions"`
	Health             *DiskHealth `json:"health,omitempty"`
}

type Partition struct {
	Name       string `json:"name"`
	SizeBytes  uint64 `json:"size_bytes"`
	Filesystem string `json:"filesystem,omitempty"`
	Label      string `json:"label,omitempty"`
	UUID       string `json:"uuid,omitempty"`
	MountPoint string `json:"mount_point,omitempty"`
}

type DiskHealth struct {
	Source             string   `json:"source"` // nvme, smartctl
	Passed             *bool    `json:"passed,omitempty"`
	TemperatureC       *float64 `json:"temperature_c,omitempty"`
	PowerOnHours       *uint64  `json:"power_on_hours,omitempty"`
	PowerCycles        *uint64  `json:"power_cycles,omitempty"`
	PercentageUsed     *int     `json:"percentage_used,omitempty"`
	AvailableSpare     *int     `json:"available_spare_percent,omitempty"`
	DataReadBytes      *uint64  `json:"data_read_bytes,omitempty"`
	DataWrittenBytes   *uint64  `json:"data_written_bytes,omitempty"`
	UnsafeShutdowns    *uint64  `json:"unsafe_shutdowns,omitempty"`
	MediaErrors        *uint64  `json:"media_errors,omitempty"`
	ReallocatedSectors *uint64  `json:"reallocated_sectors,omitempty"`
}

type GPU struct {
	PCIAddress string `json:"pci_address"`
	VendorID   string `json:"vendor_id"`
	DeviceID   string `json:"device_id"`
	Revision   string `json:"revision,omitempty"`
	Vendor     string `json:"vendor"`
	// Model is the retail name where known (AMD: libdrm's amdgpu.ids),
	// otherwise the chip name.
	Model     string    `json:"model"`
	Chip      string    `json:"chip,omitempty"`
	Subsystem string    `json:"subsystem,omitempty"`
	Driver    string    `json:"driver,omitempty"`
	VRAMBytes uint64    `json:"vram_bytes,omitempty"`
	BootVGA   bool      `json:"boot_vga"`
	DRMCard   string    `json:"drm_card,omitempty"`
	Link      *PCIeLink `json:"pcie_link,omitempty"`
	Outputs   []string  `json:"outputs"` // connectors, e.g. DP-1, HDMI-A-1
}

type Display struct {
	Connector       string  `json:"connector"` // card1-DP-1
	Manufacturer    string  `json:"manufacturer,omitempty"`
	ManufacturerID  string  `json:"manufacturer_id,omitempty"` // PNP ID, e.g. DEL
	Model           string  `json:"model,omitempty"`
	ProductCode     string  `json:"product_code,omitempty"`
	Serial          string  `json:"serial,omitempty"`
	Year            int     `json:"year,omitempty"`
	WidthMM         int     `json:"width_mm,omitempty"`
	HeightMM        int     `json:"height_mm,omitempty"`
	DiagonalIn      float64 `json:"diagonal_in,omitempty"`
	NativeWidth     int     `json:"native_width,omitempty"`
	NativeHeight    int     `json:"native_height,omitempty"`
	NativeRefreshHz float64 `json:"native_refresh_hz,omitempty"`
	EDIDVersion     string  `json:"edid_version,omitempty"`
}

type NIC struct {
	Name       string `json:"name"`
	Type       string `json:"type"` // ethernet, wireless, virtual...
	MAC        string `json:"mac,omitempty"`
	MACVendor  string `json:"mac_vendor,omitempty"` // registered owner of the MAC prefix
	Driver     string `json:"driver,omitempty"`
	Bus        string `json:"bus,omitempty"` // pci, usb
	BusAddress string `json:"bus_address,omitempty"`
	Vendor     string `json:"vendor,omitempty"`
	Model      string `json:"model,omitempty"`
	State      string `json:"state"`
	SpeedMbps  int    `json:"speed_mbps,omitempty"`
	Duplex     string `json:"duplex,omitempty"`
	MTU        int    `json:"mtu,omitempty"`
}

type BluetoothController struct {
	Name    string `json:"name"` // hci0
	Address string `json:"address,omitempty"`
	// AddressVendor is the registered owner of the address prefix.
	AddressVendor  string `json:"address_vendor,omitempty"`
	ManufacturerID int    `json:"manufacturer_id,omitempty"` // Bluetooth SIG company ID
	Manufacturer   string `json:"manufacturer,omitempty"`
	Version        string `json:"version,omitempty"` // core spec version, e.g. "5.2"
	LocalName      string `json:"local_name,omitempty"`
	Powered        *bool  `json:"powered,omitempty"`
	Bus            string `json:"bus,omitempty"`
	BusAddress     string `json:"bus_address,omitempty"`
	Vendor         string `json:"vendor,omitempty"` // adapter's USB/PCI vendor
	Model          string `json:"model,omitempty"`
}

type SoundCard struct {
	Index      int          `json:"index"`
	ID         string       `json:"id"`
	Name       string       `json:"name"`
	Driver     string       `json:"driver,omitempty"`
	Bus        string       `json:"bus,omitempty"`
	BusAddress string       `json:"bus_address,omitempty"`
	Codecs     []AudioCodec `json:"codecs,omitempty"`
}

// AudioCodec is an HD Audio codec chip, named by the kernel.
type AudioCodec struct {
	Name        string `json:"name"`
	Vendor      string `json:"vendor,omitempty"`
	VendorID    string `json:"vendor_id"` // e.g. 14f15098: PCI vendor 14f1, device 5098
	SubsystemID string `json:"subsystem_id,omitempty"`
	Revision    string `json:"revision,omitempty"`
}

type Battery struct {
	Name            string  `json:"name"`
	Manufacturer    string  `json:"manufacturer,omitempty"`
	Model           string  `json:"model,omitempty"`
	Serial          string  `json:"serial,omitempty"`
	Technology      string  `json:"technology,omitempty"`
	Status          string  `json:"status,omitempty"`
	CapacityPercent int     `json:"capacity_percent,omitempty"`
	DesignWh        float64 `json:"design_wh,omitempty"`
	FullWh          float64 `json:"full_wh,omitempty"`
	// HealthPercent is full / design capacity.
	HealthPercent float64 `json:"health_percent,omitempty"`
	CycleCount    int     `json:"cycle_count,omitempty"`
}

type Sensor struct {
	Chip     string          `json:"chip"` // coretemp, nvme, amdgpu...
	Device   string          `json:"device,omitempty"`
	Readings []SensorReading `json:"readings"`
}

type SensorReading struct {
	Label string  `json:"label"`
	Kind  string  `json:"kind"` // temperature, fan, voltage, power, current
	Value float64 `json:"value"`
	Unit  string  `json:"unit"` // C, RPM, V, W, A
	Max   float64 `json:"max,omitempty"`
	Crit  float64 `json:"crit,omitempty"`
}

type PCIDevice struct {
	Address     string    `json:"address"`
	VendorID    string    `json:"vendor_id"`
	DeviceID    string    `json:"device_id"`
	SubVendorID string    `json:"subsystem_vendor_id,omitempty"`
	SubDeviceID string    `json:"subsystem_device_id,omitempty"`
	ClassCode   string    `json:"class_code"`
	Class       string    `json:"class"`
	Vendor      string    `json:"vendor"`
	Device      string    `json:"device"`
	Subsystem   string    `json:"subsystem,omitempty"`
	Revision    string    `json:"revision,omitempty"`
	Driver      string    `json:"driver,omitempty"`
	IOMMUGroup  string    `json:"iommu_group,omitempty"`
	Link        *PCIeLink `json:"pcie_link,omitempty"`
}

type PCIeLink struct {
	Speed    string `json:"speed"`
	Width    int    `json:"width"`
	MaxSpeed string `json:"max_speed"`
	MaxWidth int    `json:"max_width"`
}

type USBDevice struct {
	Bus        int      `json:"bus"`
	Device     int      `json:"device"`
	Path       string   `json:"path"` // sysfs name, e.g. 1-2.3
	VendorID   string   `json:"vendor_id"`
	ProductID  string   `json:"product_id"`
	Vendor     string   `json:"vendor"`
	Product    string   `json:"product"`
	Serial     string   `json:"serial,omitempty"`
	ClassCode  string   `json:"class_code,omitempty"`
	Class      string   `json:"class,omitempty"`
	SpeedMbps  float64  `json:"speed_mbps,omitempty"`
	USBVersion string   `json:"usb_version,omitempty"`
	Drivers    []string `json:"drivers,omitempty"`
}
