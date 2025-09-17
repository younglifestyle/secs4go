package gem

// DeviceType indicates whether GEM handler behaves as Host or Equipment.
type DeviceType int

const (
	// DeviceHost represents a GEM host.
	DeviceHost DeviceType = iota
	// DeviceEquipment represents a GEM equipment.
	DeviceEquipment
)

func (dt DeviceType) String() string {
	switch dt {
	case DeviceHost:
		return "Host"
	case DeviceEquipment:
		return "Equipment"
	default:
		return "Unknown"
	}
}
