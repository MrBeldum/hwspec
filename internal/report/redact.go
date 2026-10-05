package report

// Redact clears identifiers that tie a report to one physical machine or
// person (serial numbers, UUIDs, MAC addresses, hostname), so the file can be
// shared publicly. Models, versions and sizes are kept.
func (r *Report) Redact() {
	r.Redacted = true
	r.Hostname = ""
	r.System.Serial, r.System.UUID, r.System.ChassisSerial = "", "", ""
	r.Board.Serial, r.Board.AssetTag = "", ""
	for i := range r.Memory.Modules {
		r.Memory.Modules[i].Serial = ""
	}
	for i := range r.Storage {
		d := &r.Storage[i]
		d.Serial, d.WWN = "", ""
		for j := range d.Partitions {
			d.Partitions[j].UUID = ""
			d.Partitions[j].Label = ""
		}
	}
	for i := range r.Displays {
		r.Displays[i].Serial = ""
	}
	for i := range r.Network {
		r.Network[i].MAC = ""
	}
	for i := range r.Batteries {
		r.Batteries[i].Serial = ""
	}
	for i := range r.USB {
		r.USB[i].Serial = ""
	}
}
