package scheduler

import "github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"

func matchesTopology(
	workload cluster.Workload,
	worker cluster.Worker,
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
