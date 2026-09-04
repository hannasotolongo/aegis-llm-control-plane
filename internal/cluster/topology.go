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

func MatchesTopologyRequirement(
	workload Workload,
	worker Worker,
) bool {
	requirement := workload.TopologyRequirement

	if requirement.RequiredNodeID != "" &&
		worker.Topology.NodeID != requirement.RequiredNodeID {
		return false
	}

	if requirement.RequiredRackID != "" &&
		worker.Topology.RackID != requirement.RequiredRackID {
		return false
	}

	if requirement.RequiredNVLinkDomain != "" &&
		worker.Topology.NVLinkDomain != requirement.RequiredNVLinkDomain {
		return false
	}

	if requirement.RequiredInterconnect != "" &&
		worker.Topology.Interconnect != requirement.RequiredInterconnect {
		return false
	}

	if workload.RequiredTopologyDomain != "" {
		if worker.Topology.RackID != "" {
			return worker.Topology.RackID ==
				workload.RequiredTopologyDomain
		}

		return worker.TopologyDomain ==
			workload.RequiredTopologyDomain
	}

	return true
}
