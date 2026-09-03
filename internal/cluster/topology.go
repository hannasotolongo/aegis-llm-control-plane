package cluster

type InterconnectType string

const (
	InterconnectPCIe     InterconnectType = "PCIE"
	InterconnectNVLink   InterconnectType = "NVLINK"
	InterconnectNVSwitch InterconnectType = "NVSWITCH"
	InterconnectNetwork  InterconnectType = "NETWORK"
)

type GPUTopology struct {
	NodeID       string
	RackID       string
	GPUIndex     int
	NVLinkDomain string
	Interconnect InterconnectType
}

type TopologyRequirement struct {
	RequiredNodeID       string
	RequiredRackID       string
	RequiredNVLinkDomain string
	RequiredInterconnect InterconnectType
}
