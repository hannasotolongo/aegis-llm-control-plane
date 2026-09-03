# Aegis: Predictive GPU Workload Control Plane

GPU infrastructure is expensive and finite, yet multiple AI workloads often compete for the same resources while demand and utilization are constantly changing. Reactive scheduling can place workloads based on capacity that appears available now, creating contention, conflicting resource claims, and inefficient GPU utilization as conditions change.

Aegis addresses this problem with a distributed control plane that coordinates workload placement and GPU resource allocation around a consistent view of cluster state. Rather than relying only on current capacity. Aegis tracks recent resource behavior to identify developing pressure and make placement decisions before contention becomes the problem.

### Objective

Aegis is designed to make limited GPU infrastructure more efficient by improving how workloads are placed and resources are allocated across a changing cluster. The system coordinates competing scheduling decisions to prevent conflicting resource claims and uses recent resource behavior to recognize when a worker may be moving toward contention. The broader objective is to shift GPU workload management from reacting to resource pressure after it occurs to making better placement decisions before performance is affected.

## The Problem

GPU scheduling decisions are made against a cluster that is constantly changing. Available memory, utilization, worker health, and workload demand can change between the moment a worker is evaluated and the moment a workload is actually placed.

This becomes especially problematic when workloads are scheduled concurrently. Two workloads can evaluate the same GPU at nearly the same time, both see sufficient capacity, and independently attempt to claim the same resources. Without coordinated placement and resource accounting, the result can be over-allocation, inconsistent cluster state, and unnecessary contention.

Even when a placement is valid at the moment it is made, current capacity alone does not indicate whether that worker is moving toward resource pressure. A scheduler that only reacts to the present state may recognize a poor placement only after contention has already developed.

Aegis was built around these two problems. Making concurrent resource allocation safe and making placement decisions more aware of where resource conditions are heading, not just where they are now.

## System Architecture

Aegis is structured as a control plane that maintains a shared view of GPU workers, workloads, and available resources. Scheduling decisions are made against this state so workload placement and resource allocation remain coordinated as cluster conditions change.

When a workload enters the system, Aegis determines its scheduling priority, identifies workers capable of running it, evaluates the eligible workers, and selects a placement. The placement is then coordinated with the corresponding resource reservation so the cluster state reflects the resources that have actually been claimed.

Alongside this scheduling path, Aegis collects worker telemetry and maintains recent resource history. This information is used to estimate short-horizon resource pressure and can influence placement decisions when sufficient prediction information is available.

The system separates cluster state, scheduling, telemetry, prediction, simulation, and API handling into independent components so each part of the control plane can be developed and tested independently.

## Cluster State & Concurrency

Aegis maintains a shared view of the workers and workloads known to the control plane, including resource availability, workload assignments, and worker state. Scheduling decisions depend on this information remaining consistent as multiple operations occur at the same time.

To protect shared state, Aegis uses concurrency-safe state management so reads and updates cannot interfere with one another. Before a placement is committed, the system verifies that the selected worker is still eligible and that the required capacity remains available.

This closes the gap between checking capacity and actually reserving it. A workload is not treated as successfully placed until its assignment and the corresponding resource allocation are reflected in cluster state, preventing competing scheduling operations from independently claiming the same available capacity.

## Workload Scheduling & Worker Selection

Once a workload is ready to be scheduled, Aegis first determines which workers can safely run it. Workers that are unhealthy, lack sufficient GPU capacity, or fail the workload's placement requirements are removed from consideration.

The remaining workers are evaluated to determine which represents the best placement under current cluster conditions. Rather than assigning a workload to the first GPU with enough capacity, Aegis considers how effectively the available resources are already being used and whether the placement is appropriate for that workload.

Scheduling is deterministic, meaning the same workload evaluated against the same cluster state produces the same placement decision. This makes scheduling behavior predictable, reproducible, and easier to validate while providing a stable foundation for incorporating predictive resource signals.

## Telemetry & Predictive Scheduling

A placement can be valid based on current resource availability and still become a poor decision shortly afterward. To account for this, Aegis collects worker telemetry over time rather than relying only on a single snapshot of cluster conditions.

Recent resource history is used to estimate short-horizon GPU pressure and determine whether a worker appears to be moving toward contention. When sufficient information is available, these estimates can influence scheduling so workloads are less likely to be placed on workers showing increasing resource pressure.

When prediction data is unavailable or confidence is insufficient, Aegis falls back to its deterministic scheduling logic. The current predictor uses a lightweight trend-based approach, providing a transparent baseline for evaluating whether predictive resource signals can improve placement decisions without making scheduling dependent on unreliable forecasts.

## Worker Health & Failure Recovery

A scheduling decision is only useful while the selected worker remains available. Aegis tracks worker health so unhealthy or unavailable workers are excluded from new placement decisions.

When worker state changes, Aegis can identify workloads affected by the failure and move them through recovery oriented workload states rather than continuing to treat their original placement as valid. Resource state can then be reconciled so the control plane maintains an accurate view of the cluster.

This allows scheduling decisions to account not only for where a workload should initially run, but also for what happens when the infrastructure supporting that placement changes.

## Simulation & Testing

Aegis includes controlled simulation scenarios that reproduce changing GPU utilization and resource pressure, allowing scheduling and predictive behavior to be evaluated under repeatable conditions.

Automated tests validate cluster-state operations, workload scheduling, placement, prediction, and failure handling. Because concurrency is central to the system, Aegis is also tested with Go's race detector to identify unsafe access to shared state and verify that concurrent operations preserve consistent resource accounting.

Together, simulation and testing provide a repeatable way to verify that Aegis continues making valid placement decisions as workloads compete for resources and cluster conditions change.

## HTTP API

Aegis exposes an HTTP API that provides an external interface to the control plane. Through the API, workers can be registered and inspected, workloads can be submitted and tracked, and the health of the service can be checked.

The API is intentionally separated from the underlying scheduling and state-management logic. Requests enter through the API layer, are validated, and then passed to the appropriate control-plane components rather than embedding scheduling decisions directly inside request handlers.

This separation allows the scheduling system to evolve independently from the interface used to interact with it.

## Running Aegis

From the repository root, run the full test suite:

```bash
go test ./...
```

Run the tests with Go's race detector to validate concurrent state operations:

```bash
go test -race ./...
```

Start the Aegis control plane:

```bash
go run ./cmd/aegis
```

The service runs locally on port `8080`. Verify that the control plane is running:

```bash
curl http://localhost:8080/healthz
```

A healthy instance returns:

```json
{
  "status": "ok"
}
```

## Repository Structure

```text
aegis-llm-control-plane/
├── cmd/
│   └── aegis/              # Application entry point
├── internal/
│   ├── api/                # HTTP interface and request handling
│   ├── cluster/            # Worker, workload, and shared cluster state
│   ├── scheduler/          # Scheduling, worker selection, and placement
│   ├── telemetry/          # Resource telemetry and history
│   ├── predictor/          # Resource-pressure prediction
│   └── simulator/          # Controlled GPU utilization simulation
├── go.mod
└── README.md
```

The project is organized so that cluster state, scheduling policy, telemetry, prediction, simulation, and external API handling remain separate concerns within the control plane.

## Current Status & Limitations

Aegis is currently an experimental control plane prototype designed to evaluate resource aware and predictive GPU workload scheduling. The core scheduling, concurrent state management, resource reservation, worker health, telemetry, prediction, simulation, and API components are implemented and tested.

The current system operates against simulated GPU workers rather than physical GPU infrastructure, and cluster state is maintained in memory rather than persisted to an external database. Resource pressure prediction currently uses a lightweight trend based model rather than a trained forecasting system.

Aegis is not intended to represent a production ready GPU orchestrator in its current form. Persistent state, live GPU integration, production observability, distributed coordination, and additional infrastructure hardening remain areas for future development.
