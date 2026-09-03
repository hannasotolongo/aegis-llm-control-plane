# Aegis: Predictive GPU Workload Control Plane for LLM Inference

## Overview
Aegis is a GPU workload control plane designed to make LLM inference infrastructure more efficient by improving how limited GPU capacity is allocated as cluster conditions change.

GPU scheduling is difficult because placement decisions are made against a constantly changing system. Available capacity can disappear, resource pressure can develop after placement, and concurrent workloads can compete for the same resources. Aegis addresses these problems by coordinating scheduling and resource allocation around a consistent view of the GPU cluster. It protects capacity from conflicting claims, evaluates whether a worker is appropriate for a specific workload, and uses resource telemetry to anticipate developing memory and compute pressure before placement.

Rather than asking only where a workload can run now, Aegis considers where it should run as resource conditions evolve. The goal is to prevent conflicting allocations, reduce avoidable contention, improve utilization of existing GPU capacity, and support growing inference demand without treating additional hardware as the first solution.

## The Problem

GPU inference capacity is not static. Every placement changes the resources available to the next workload, while workloads already running on the cluster continue to change memory and compute pressure over time. A scheduler making decisions from current capacity alone can therefore choose a worker that is technically valid at placement time but becomes a poor choice shortly afterward.

Concurrency makes this more difficult. Multiple scheduling operations can observe the same available capacity before either has completed its placement. If resource validation and reservation are not coordinated, both can select the same worker and collectively request more capacity than actually exists. The individual scheduling decisions may each appear valid while the resulting cluster state is not.

LLM inference adds another layer to the placement problem because available memory alone does not determine whether two workers are equally suitable. The workload being served, existing utilization, model locality, GPU topology, latency requirements, and expected resource demand can all affect the quality of a placement.

Aegis treats scheduling as a resource coordination problem rather than a simple capacity lookup. Placement decisions must remain correct under concurrency, satisfy workload and infrastructure constraints, and account for developing resource pressure before additional work is assigned.

## System Design

Aegis separates scheduling into a set of control plane components responsible for cluster state, workload placement, telemetry, forecasting, and worker health. This separation allows each scheduling decision to combine information about what a workload requires, what resources are currently available, and how those resources are changing over time.

When a workload enters the system, Aegis first determines which workers are eligible to run it. Eligibility is based on hard constraints such as worker health, available GPU memory, and topology requirements. Eligible workers are then scored using current resource pressure, active workloads, available memory, and model locality.

Placement is coordinated with resource reservation rather than treated as a separate action. Before a workload is committed to a worker, Aegis verifies that the worker is still eligible and that the required capacity remains available. The workload assignment and resource state are then updated together, preventing concurrent scheduling decisions from independently claiming the same capacity.

Alongside this scheduling path, Aegis maintains recent worker telemetry and produces short horizon forecasts of memory and compute utilization. These forecasts provide the scheduler with information about developing resource pressure, allowing worker selection to consider both current conditions and where those conditions may be heading.

Hard constraints determine where a workload can safely run, while resource scoring and predictive signals help Aegis choose the most efficient placement among eligible workers.

## LLM Workload Model

LLM inference workloads can place very different demands on the same GPU. Aegis represents those differences directly so placement decisions are based on the workload being scheduled rather than GPU availability alone.

Each workload captures its model, prompt size, generation limit, batch size, GPU memory requirement, KV-cache demand, expected compute intensity, latency target, priority, and expected duration. Together, these characteristics provide the control plane with a more useful representation of the resources and service requirements associated with the workload.

Memory is modeled carefully to avoid double counting. `RequiredMemoryMB` represents the workload's total GPU memory reservation, while `KVCacheMemoryMB` identifies the portion associated with KV-cache demand. The scheduler therefore reserves the total requirement once rather than adding KV-cache memory on top of an amount that already includes it.

Workloads also carry lifecycle state and placement information, allowing Aegis to track them from arrival through scheduling, placement, execution, recovery, completion, or failure. This gives the control plane a consistent representation of both what a workload requires and where it is in the scheduling lifecycle.

By modeling workload demand explicitly, Aegis creates the foundation for placement decisions that account for the cost of running a particular inference workload instead of treating all requests as interchangeable.

## GPU Topology

Available GPU capacity does not tell the scheduler how that GPU is connected to the rest of the infrastructure. For inference workloads, placement can also depend on where a GPU is located and the communication path available to it.

Aegis represents GPU topology explicitly. Workers can be identified by node, rack, GPU index, NVLink domain, and interconnect type, while workloads can specify corresponding placement requirements. This allows topology to become an eligibility constraint rather than an assumption hidden inside the scheduler.

During worker selection, Aegis removes workers that do not satisfy the workload's topology requirements before scoring begins. A worker with excellent memory and compute availability is therefore not considered a valid placement if it does not meet the required infrastructure constraints.

Keeping topology eligibility separate from resource scoring prevents an attractive utilization score from overriding a physical placement requirement. It also provides a foundation for more advanced scheduling decisions where communication cost and data movement become increasingly important as workloads span larger GPU environments.

## Cluster State & Concurrency

Scheduling depends on an accurate view of the cluster. Aegis maintains shared state for workers, workloads, resource availability, assignments, and worker health so scheduling decisions operate against a consistent representation of the system.

The challenge is that scheduling operations can happen concurrently. Two workloads may inspect the same worker before either placement is complete, both observe sufficient capacity, and both attempt to reserve resources that cannot support them together. Checking capacity before placement is therefore not enough; the resource state must still be valid when the placement is committed.

Aegis protects shared cluster state with concurrency-safe access and coordinates workload placement with resource reservation. Before committing a placement, the system verifies that the selected worker remains eligible and still has sufficient capacity. The worker's available memory and workload count are updated alongside the workload assignment so the resources consumed by the placement are immediately reflected in cluster state.

This prevents independently valid scheduling decisions from creating an invalid combined allocation. It also gives subsequent scheduling operations an updated view of available capacity, keeping resource accounting consistent as multiple workloads compete for the same GPU infrastructure.

## Workload Scheduling & Worker Scoring

Once Aegis identifies the workers capable of running a workload, it evaluates the eligible candidates to determine which worker represents the better use of available GPU capacity.

Worker scoring considers available memory, current compute and memory utilization, the number of active workloads, and whether the requested model is already available on the worker. Memory headroom increases a worker's score, while existing resource pressure and workload activity reduce it. Model locality provides an additional advantage because a worker that already has the required model is generally a more attractive placement than an otherwise similar worker that does not.

Eligibility and scoring remain deliberately separate. Worker health, available capacity, and topology requirements determine whether a placement is allowed. Scoring only compares workers that have already passed those checks, preventing a favorable score from overriding a resource or infrastructure constraint.

Scheduling is deterministic, so the same workload evaluated against the same cluster state produces the same selection. This makes placement behavior reproducible and provides a stable baseline against which predictive scheduling decisions can be evaluated.

## Telemetry, Forecasting & Uncertainty

Aegis uses worker telemetry to track how GPU resource conditions change over time rather than relying only on the latest utilization reading. Each worker maintains a bounded history of memory and compute utilization, giving the control plane recent resource behavior without allowing telemetry data to grow indefinitely.

The forecasting layer uses timestamped telemetry to measure how quickly memory and compute utilization are changing and projects those trends across a defined forecast horizon. This allows Aegis to identify developing resource pressure that may not yet be visible from current utilization alone.

Forecasts contain predicted memory and compute utilization along with an indication of expected contention. Aegis also tracks prediction uncertainty by measuring historical forecast error separately for memory and compute using mean absolute error. This gives the system an explicit measure of how much recent predictions have differed from observed resource behavior.

Forecasts are treated as additional scheduling information rather than guaranteed future state. Predictions are timestamped and must remain fresh to influence placement. If predictive information is missing or stale, Aegis falls back to current-state worker scoring so scheduling can continue without depending on the forecasting layer.

## Predictive Scheduling

Aegis extends its baseline worker scoring with short-horizon resource forecasts. Instead of ranking eligible workers only by their current state, the predictive scheduler also considers the memory and compute pressure expected to develop after placement.

Fresh forecasts influence worker scoring by penalizing GPUs that are expected to become more heavily utilized or enter contention. This allows Aegis to avoid a worker that appears attractive based on current utilization but is trending toward higher resource pressure.

Prediction does not replace the scheduler's safety checks. Worker health, available memory, and topology requirements must still be satisfied before predictive scoring is considered. Forecasts help choose between valid workers; they cannot make an invalid placement acceptable.

Aegis also avoids making scheduling dependent on the forecasting system. If a prediction is missing or stale, the scheduler falls back to its baseline scoring behavior using current cluster state. This keeps placement deterministic and operational even when predictive information is unavailable.

## Worker Health & Failure Recovery

A workload control plane must account for failures that occur after placement. Aegis tracks worker health through heartbeats and uses worker state to prevent new workloads from being assigned to infrastructure that is degraded, unhealthy, or being drained.

When a worker stops reporting within the expected heartbeat window, Aegis can transition it to an unhealthy state and remove it from normal scheduling eligibility. This prevents the scheduler from continuing to place workloads on a worker whose availability can no longer be trusted.

Aegis tracks workload state so failures can be handled beyond simply marking the worker unavailable. Workloads affected by worker failure can move through recovery states, and checkpointable workloads can be returned to the scheduling path for placement on another eligible worker.

Separating failure detection from recovery keeps the control plane responsible for both sides of the problem: identifying when infrastructure should no longer receive work and determining what should happen to workloads that were already assigned to it.

## Simulation & Testing

Aegis currently operates against simulated GPU workers, allowing scheduling, resource allocation, forecasting, failure recovery, and concurrency behavior to be exercised without requiring a physical multi-GPU cluster.

The simulator generates changing memory and compute utilization so the control plane can observe resource pressure over time and produce telemetry-driven forecasts. Workloads with different memory, compute, latency, priority, and inference characteristics can then be scheduled against these changing cluster conditions.

Testing focuses on control-plane correctness rather than only successful execution paths. The test suite covers workload and worker validation, topology constraints, scheduling decisions, resource reservation, predictive placement, stale-forecast fallback, failure recovery, and concurrent state access.

Aegis is also tested with Go's race detector to identify unsafe concurrent memory access. This is particularly important for cluster state and placement operations, where multiple scheduling actions may interact with the same workers and workloads.

The simulation does not claim to reproduce the full behavior of production GPU hardware. Its purpose is to provide a controlled environment for validating Aegis's scheduling and control-plane logic before integration with real GPU telemetry and infrastructure.

## HTTP API

Aegis exposes an HTTP API for interacting with control-plane state and submitting cluster resources and workloads. The API provides endpoints for registering and inspecting workers, submitting and inspecting workloads, and checking service health.

Worker and workload requests are validated before they enter cluster state, ensuring malformed resource descriptions or invalid lifecycle states are rejected before they can influence scheduling.

Current endpoints include:

| Method | Endpoint | Purpose |
|---|---|---|
| `GET` | `/healthz` | Check control-plane health |
| `POST` | `/v1/workers` | Register a worker |
| `GET` | `/v1/workers` | List registered workers |
| `POST` | `/v1/workloads` | Submit a workload |
| `GET` | `/v1/workloads` | List workloads |

The API is intentionally focused on exercising the control-plane architecture and is not currently hardened as a production-facing service. Authentication, authorization, rate limiting, and production API lifecycle management remain outside the current implementation.

## Running Aegis

Aegis is written in Go and can be run locally without physical GPU hardware. The simulation environment allows the control-plane logic, scheduling behavior, telemetry, and failure handling to be exercised locally.

### Clone the Repository

```bash
git clone https://github.com/hannasotolongo/aegis-llm-control-plane.git
cd aegis-llm-control-plane
```

### Run the Test Suite

```bash
go test ./...
```

Run the tests with Go's race detector to check for unsafe concurrent memory access:

```bash
go test -race ./...
```

### Start the Control Plane

```bash
go run ./cmd/aegis
```

Aegis starts the HTTP server on port `8080`.

### Verify the Service

```bash
curl http://localhost:8080/healthz
```

Expected response:

```json
{
  "status": "ok"
}
```

## Repository Structure

Aegis is organized around separate control-plane responsibilities so scheduling, state management, prediction, telemetry, and API behavior can evolve independently.

```text
aegis-llm-control-plane/
├── cmd/
│   └── aegis/              # Control-plane entry point and service startup
│
├── internal/
│   ├── api/                # HTTP API and request handling
│   ├── cluster/            # Worker, workload, topology, and cluster state
│   ├── predictor/          # Resource forecasting and uncertainty estimation
│   ├── scheduler/          # Eligibility, worker scoring, and workload placement
│   └── telemetry/          # Worker telemetry collection and resource history
│
├── go.mod                  # Go module definition
└── README.md               # Project architecture and documentation
```

The `cluster` package defines the shared resource model used throughout Aegis, including workers, workloads, lifecycle state, GPU topology, and concurrency-safe cluster state. The `scheduler` package builds on that state to determine worker eligibility, compare valid placement targets, and coordinate workload placement with resource reservation.

The `telemetry` and `predictor` packages provide the historical and forward-looking resource information used by predictive scheduling. Keeping prediction separate from core cluster state allows Aegis to use forecasts when they are available while preserving a deterministic current-state scheduling path when they are not.

## Current Status & Limitations

Aegis is an experimental control-plane implementation designed to validate resource-aware and predictive scheduling behavior for LLM inference workloads. The current system implements the scheduling path from workload admission through worker selection, resource reservation, telemetry-driven prediction, and failure recovery.

The project currently operates with simulated GPU workers rather than physical GPU hardware. Resource utilization, worker behavior, and workload demand are generated within the simulation environment, so the project validates control-plane behavior but does not claim measured performance on a production GPU cluster.

Cluster state is currently maintained in memory. This keeps the scheduling path focused and testable, but state does not survive a control-plane restart and is not replicated across multiple control-plane instances. A production implementation would require durable state, coordination between control-plane replicas, and stronger recovery guarantees.

The forecasting system is intentionally lightweight. It uses recent utilization trends to estimate short-horizon memory and compute pressure and measures historical prediction error using mean absolute error. It is not a machine-learning forecasting model and does not treat its uncertainty estimates as calibrated probabilities.

The HTTP API is also designed for development and simulation rather than exposure as a production service. Authentication, authorization, rate limiting, persistent storage, production observability, and integration with real GPU telemetry remain outside the current implementation.
The `api` package exposes the control plane through HTTP, while `cmd/aegis` assembles the components and starts the running service. This separation keeps the entry point small and places scheduling behavior inside focused packages that can be tested independently.
