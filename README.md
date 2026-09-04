# Aegis: Risk Aware GPU Workload Control Plane for LLM Inference

## Overview

Aegis is an experimental GPU workload control plane designed to improve how limited GPU capacity is allocated for LLM inference as cluster conditions change.

A placement that looks efficient at one moment can become inefficient shortly afterward as GPU memory pressure, compute utilization, and competing workload demand evolve. Concurrent scheduling creates an additional challenge because multiple workloads can observe the same available capacity and attempt to claim resources that cannot support them together.

Aegis addresses these problems across the scheduling lifecycle. It maintains concurrency safe cluster state, models LLM workload requirements, accounts for GPU topology and model locality, coordinates placement with resource reservation, monitors worker health, and collects resource telemetry over time.

Recent telemetry is used to forecast short horizon GPU memory and compute utilization. Rather than treating those forecasts as certain, Aegis also considers observed forecast error when evaluating workers. This allows the scheduler to account for current resource state, predicted pressure, and uncertainty before committing a workload to a GPU.

The goal is to make placement decisions that remain efficient as cluster conditions evolve, reducing avoidable resource contention while making better use of available GPU capacity.

## The Problem

GPU inference capacity is continuously changing. Every placement consumes resources that are no longer available to the next workload, while workloads already running on the cluster continue to change memory and compute pressure over time. A scheduler making decisions from current utilization alone can therefore select a worker that is valid at placement time but becomes heavily constrained shortly afterward.

Concurrency creates a separate correctness problem. Multiple scheduling operations can observe the same available capacity before either placement has been committed. Without coordinated validation and reservation, both workloads can select the same worker and collectively request more capacity than is actually available.

LLM inference makes placement more complex because available GPU memory alone does not determine whether a worker is a good target. Workload demand, existing utilization, model locality, GPU topology, latency requirements, predicted resource pressure, and the reliability of those predictions can all affect placement quality. Aegis treats these factors as part of a coordinated scheduling decision rather than relying on a single snapshot of available capacity.

## System Design

Aegis separates cluster state, scheduling, telemetry, forecasting, risk evaluation, and worker health into distinct control plane components. This allows placement decisions to combine information about what a workload requires, what resources are currently available, and how those resources are expected to change.

When a workload enters the scheduling path, Aegis first determines which workers are eligible based on hard constraints such as worker health, available GPU memory, and topology requirements. Eligible workers can then be evaluated using current utilization, workload activity, model locality, predicted resource pressure, and forecast uncertainty.

Worker selection is kept separate from final resource reservation. Before a placement is committed, Aegis revalidates the selected worker against current cluster state and reserves the required resources as part of the placement operation. This prevents a decision made from an earlier snapshot from being committed after the worker has become unhealthy, lost capacity, or no longer satisfies the workload's topology requirements.

This design gives Aegis a scheduling path that combines current resource awareness with forward looking placement decisions while preserving correctness when cluster conditions change concurrently. 

## Benchmark Results

Aegis includes a synthetic benchmark that compares baseline, predictive, and risk-aware scheduling against the same stream of 48 simulated workloads. The benchmark evaluates how each policy responds as forecast conditions change while workload demand and available resources remain consistent.

| Forecast Regime          | Policy     | Placements | Rejections | Pressure Steps | Contention Transitions |
| ------------------------ | ---------- | ---------: | ---------: | -------------: | ---------------------: |
| Steady Forecast          | Baseline   |         48 |          0 |             34 |                      8 |
| Steady Forecast          | Predictive |         48 |          0 |             33 |                     10 |
| Steady Forecast          | Risk-Aware |         48 |          0 |             27 |                     27 |
| Rising Forecast Pressure | Baseline   |         48 |          0 |             34 |                      8 |
| Rising Forecast Pressure | Predictive |         48 |          0 |             27 |                      9 |
| Rising Forecast Pressure | Risk-Aware |         48 |          0 |             22 |                     16 |
| Forecast Uncertainty     | Baseline   |         48 |          0 |             34 |                      8 |
| Forecast Uncertainty     | Predictive |         48 |          0 |             37 |                      3 |
| Forecast Uncertainty     | Risk-Aware |         48 |          0 |             22 |                     16 |

All three policies placed all 48 workloads without rejection in these runs, but their behavior under resource pressure differed. Under rising forecast pressure, the risk-aware policy reduced recorded pressure steps from 34 for the baseline to 22, while the predictive policy recorded 27. Under forecast uncertainty, risk-aware scheduling again recorded 22 pressure steps compared with 34 for baseline and 37 for predictive scheduling.

The results also show that lower resource pressure does not automatically mean better performance on every scheduling metric. Risk aware scheduling produced more contention transitions in these experiments, while predictive scheduling produced fewer transitions in the uncertainty regime. The benchmark is therefore intended to expose scheduling tradeoffs rather than demonstrate that one policy dominates every metric.

These results come from Aegis's controlled simulation environment rather than physical GPU hardware. They demonstrate differences in control-plane behavior under identical simulated workload demand and should not be interpreted as production GPU performance measurements.


## LLM Workload Model

LLM inference workloads can place very different demands on the same GPU, so Aegis represents workload requirements explicitly rather than treating every request as interchangeable. The workload model captures characteristics such as model identity, token volume, batch size, GPU memory requirements, KV-cache demand, expected compute intensity, latency targets, priority, and expected duration.

Memory requirements are modeled to avoid double counting. `RequiredMemoryMB` represents the workload's total GPU memory reservation, while `KVCacheMemoryMB` identifies the portion associated with KV-cache demand. The scheduler therefore reserves the total memory requirement once while retaining information about how that demand is composed.

The model also tracks placement and lifecycle state, allowing workloads to move from admission through scheduling, execution, recovery, completion, or failure. These characteristics give the scheduler information about both the resources a workload requires and how it should be handled throughout its lifecycle.

## GPU Topology

Available GPU capacity alone does not determine whether a worker is an appropriate placement target. Where a GPU is located and how it is connected to the surrounding infrastructure can also affect whether it satisfies a workload's requirements.

Aegis represents topology using node, rack, GPU index, NVLink domain, and interconnect information. Workloads can specify corresponding topology requirements, allowing the scheduler to exclude workers that do not satisfy the required infrastructure constraints before comparing their resource characteristics.

Topology remains a hard constraint throughout the placement process. Aegis checks these requirements during worker selection and revalidates them when the placement is committed, preventing an otherwise attractive utilization or risk score from overriding an infrastructure requirement.

## Cluster State & Concurrency

Scheduling decisions depend on a consistent view of workers, workloads, resource availability, and placement state. Aegis maintains this information in a concurrency-safe in-memory state store so multiple control-plane operations can interact with shared cluster state without unsafe access.

Worker selection and resource reservation are intentionally separated. A scheduler can identify a suitable worker from a snapshot, but cluster conditions may change before that placement is committed. Aegis therefore revalidates the workload and selected worker while holding exclusive access to cluster state before modifying resources.

During the commit, Aegis verifies that the workload is still schedulable, the worker remains healthy, topology requirements are still satisfied, and sufficient GPU memory remains available. Only after those checks succeed are the worker's resources and workload assignment updated.

This prevents concurrent scheduling decisions from independently claiming the same capacity and keeps resource accounting consistent as workloads enter and leave the system.

## Workload Scheduling & Worker Scoring

Once Aegis determines which workers are eligible for a workload, it compares those candidates to determine which represents the best placement under current cluster conditions. The baseline scheduler considers available memory, compute and memory utilization, active workload count, and model locality when evaluating each worker.

Workers with greater memory headroom receive a higher score, while existing resource pressure and active workloads reduce it. Model locality provides an additional advantage when the requested model is already cached on a worker, allowing the scheduler to favor an otherwise suitable GPU that can serve the workload without unnecessary model movement.

Eligibility remains separate from scoring. A favorable score cannot override worker health, memory capacity, or topology requirements. When workers receive the same score, Aegis uses a deterministic tie-break so identical cluster state produces reproducible placement decisions.

This baseline policy provides the foundation for the predictive and risk-aware scheduling layers, which introduce information about expected future resource conditions without removing the scheduler's existing safety constraints.

## Telemetry, Forecasting & Uncertainty

Aegis collects timestamped memory and compute utilization from each worker so scheduling decisions can consider how resource conditions are changing rather than relying only on the latest measurement. Recent observations are maintained as a bounded telemetry history and used to estimate short-horizon utilization trends.

The forecasting layer measures how memory and compute utilization are changing over time and projects those trends across a defined horizon. This gives the scheduler an indication of developing resource pressure that may not yet be apparent from current utilization alone.

Because forecasts are imperfect, Aegis also tracks historical prediction error using mean absolute error for memory and compute independently. This provides an explicit measure of recent forecast uncertainty rather than assuming every prediction is equally reliable.

Forecasts are timestamped and only influence scheduling while they remain fresh. If predictive information is missing or stale, Aegis falls back to current-state worker evaluation so placement can continue without depending on the forecasting layer.

## Predictive Scheduling

Aegis extends its baseline scheduling policy by incorporating expected future GPU conditions into worker selection. Instead of evaluating an eligible worker only by its current utilization, the predictive scheduler also considers forecasted memory and compute pressure.

This allows Aegis to distinguish between a worker that is lightly utilized and expected to remain stable and one that appears equally available now but is trending toward contention. Forecasts influence the choice between eligible workers without replacing the scheduler's health, memory, or topology requirements.

Predictive scheduling remains resilient to missing information. When a usable forecast is unavailable, Aegis returns to baseline worker scoring rather than blocking placement or making assumptions about future resource conditions.

## Risk-Aware Scheduling

Forecasting future utilization is useful, but a prediction should not be treated as certain. Aegis extends predictive scheduling with a risk-aware policy that considers both expected resource pressure and the recent accuracy of those predictions.

For each eligible worker, Aegis evaluates predicted memory and compute utilization together with observed forecast error and the resource demand of the workload being placed. This produces a placement risk that reflects not only how constrained a worker is expected to become, but also how much uncertainty exists around that expectation.

When fresh risk information is available, the scheduler prioritizes workers with lower placement risk before using baseline worker scoring to resolve remaining differences. This allows Aegis to avoid relying too heavily on an attractive forecast when recent prediction error suggests that forecast may be less dependable.

Risk evaluation remains part of worker selection rather than a replacement for placement safety. Worker health, available memory, and topology requirements must still be satisfied before a workload can be committed.

## Placement Risk Model

Aegis defines placement risk as the maximum projected pressure across GPU memory and compute after accounting for both forecast uncertainty and the workload being considered for placement.

For a workload \(w\), worker \(g\), and forecast horizon \(h\), the risk score is represented as:

$$
R(w,g,t)=
\max
\left(
\hat{M}_{g,t+h}+E^M_g+D^M_w,\;
\hat{C}_{g,t+h}+E^C_g
\right)
$$

Here, \(\hat{M}_{g,t+h}\) and \(\hat{C}_{g,t+h}\) represent predicted memory and compute utilization for the worker at the forecast horizon. \(E^M_g\) and \(E^C_g\) represent recent mean absolute forecast error for memory and compute, while \(D^M_w\) represents the additional memory pressure introduced by placing the workload on that worker.

The model is intentionally conservative. Forecast error is added to predicted utilization so workers with less reliable recent predictions are treated as riskier than workers with similar forecasts but lower observed error. Memory demand from the prospective workload is also incorporated before the placement is committed, allowing the scheduler to estimate the resource pressure that would exist after placement rather than evaluating the worker in isolation.

The resulting score is mapped into low, moderate, high, and critical risk levels. These levels are used to compare eligible workers during risk-aware scheduling. The score is a deterministic placement-risk heuristic rather than a calibrated probability of contention, which keeps the model consistent with the uncertainty information currently produced by Aegis.

## Worker Health & Failure Recovery

Aegis monitors worker health so scheduling decisions reflect whether infrastructure remains available after it enters the cluster. Workers that stop reporting within the expected heartbeat window can transition to an unhealthy state and are excluded from new workload placement.

Worker failure also affects workloads that have already been assigned. When Aegis detects an unhealthy worker, the reconciliation path identifies its affected workloads and releases the resources associated with those placements. Checkpointable workloads can transition into recovery and return to the scheduling path, while workloads that cannot be recovered transition to a failed state.

Keeping health detection, resource cleanup, and rescheduling coordinated allows the control plane to respond to infrastructure failure without leaving stale workload assignments or reserved GPU capacity behind.

## Workload Lifecycle & Resource Release

Aegis tracks workloads beyond the initial scheduling decision so reserved GPU capacity can be returned to the cluster as execution progresses. The simulator maintains active workloads and advances them according to their expected duration, allowing completed work to leave the system rather than consuming resources indefinitely.

When a workload completes, Aegis releases its reserved GPU memory, decreases the worker's active workload count, adjusts simulated compute utilization, and transitions the workload to a completed state. The released capacity then becomes available for future scheduling decisions.

Modeling completion and resource release is important for evaluating scheduling behavior over time. Without a workload lifecycle, resource pressure would only accumulate, making it impossible to observe how different placement policies behave as GPU capacity is consumed and later returned to the cluster.

## Simulation & Testing

Aegis operates against simulated GPU workers so the scheduling and control-plane architecture can be exercised without requiring a physical multi-GPU cluster. The simulation generates changing resource conditions and workload activity, allowing the system to evaluate placement decisions as memory and compute pressure evolve over time.

The test suite focuses on control-plane correctness across scheduling, resource reservation, topology constraints, forecasting, risk-aware placement, workload recovery, and concurrent state access. It also tests cases where cluster conditions change between worker selection and placement commit, ensuring that invalid placements are rejected without partially modifying resource state.

Aegis is tested with Go's race detector in addition to the standard test suite. The simulation is not intended to reproduce the full behavior of production GPU hardware; it provides a controlled environment for validating scheduling behavior and comparing placement policies before integration with real GPU infrastructure.

## HTTP API

Aegis exposes an HTTP API for interacting with the control plane, including registering workers, submitting workloads, inspecting cluster state, and initiating scheduling operations. Requests are validated before entering cluster state so invalid worker or workload definitions do not influence scheduling decisions.

Scheduling initiated through the API uses the same configured scheduling service as the background control-plane loop. This keeps placement behavior consistent regardless of whether scheduling occurs automatically or through an API request.

The API is currently designed for development and simulation rather than production exposure. Authentication, authorization, rate limiting, and other production API controls are outside the current implementation.

## Running Aegis

Aegis can be run locally without physical GPU hardware. After cloning the repository, the full test suite can be run before starting the control plane.

```bash
git clone https://github.com/hannasotolongo/aegis-llm-control-plane.git
cd aegis-llm-control-plane
go test ./...
go test -race ./...
```

Start the control plane with:

```bash
go run ./cmd/aegis
```

The HTTP server runs on port `8080`. Its health endpoint can be verified with:

```bash
curl http://localhost:8080/healthz
```

A healthy instance returns:

```json
{
  "status": "ok"
}
```

The synthetic benchmark can be run separately to compare baseline, predictive, and risk-aware scheduling behavior across the configured forecast conditions.

```bash
go run ./cmd/benchmark
```

## Repository Structure

Aegis is organized into focused packages that separate cluster state, scheduling, prediction, risk evaluation, simulation, telemetry, and API behavior.

```text
aegis-llm-control-plane/
├── cmd/
│   ├── aegis/          # Control-plane runtime
│   └── benchmark/      # Synthetic scheduler benchmark
│
├── internal/
│   ├── api/            # HTTP API
│   ├── cluster/        # Workers, workloads, topology, and cluster state
│   ├── predictor/      # Resource forecasting and forecast error
│   ├── risk/           # Placement-risk evaluation
│   ├── scheduler/      # Worker selection and workload placement
│   ├── simulator/      # Simulated GPU environment and workload lifecycle
│   └── telemetry/      # Resource telemetry collection
│
├── go.mod
└── README.md
```

The separation keeps core scheduling logic independent from the simulator and API. Cluster state provides the shared resource model, while telemetry and forecasting supply the forward-looking information used by predictive and risk-aware scheduling. The runtime in `cmd/aegis` assembles these components into the running control plane.

## Current Status & Limitations

Aegis is an experimental control-plane prototype designed to evaluate resource-aware, predictive, and risk-aware GPU scheduling for LLM inference. The current implementation covers the scheduling lifecycle from workload admission and worker selection through resource reservation, execution, recovery, and resource release.

The system currently operates with simulated GPU workers and synthetic workload demand rather than physical GPU hardware. The benchmark results therefore demonstrate differences in control-plane behavior under controlled conditions and should not be interpreted as production GPU performance measurements.

Cluster state is maintained in memory and is concurrency-safe within a single control-plane process, but it is not durable or replicated across multiple instances. A production implementation would require persistent transactional state or another distributed coordination mechanism capable of preserving placement correctness across control-plane replicas.

The forecasting system is intentionally lightweight and uses recent utilization trends to estimate short-horizon memory and compute pressure. Forecast uncertainty is based on observed mean absolute error and is not intended to represent a calibrated probability or formal confidence bound.

Integration with real GPU telemetry, production inference infrastructure, distributed control-plane coordination, authentication, persistent storage, high availability, and production observability remain outside the current implementation. These limitations keep the project focused on its central systems question: whether GPU placement can be improved by combining workload requirements and current resource state with predicted pressure and the uncertainty associated with those predictions.
