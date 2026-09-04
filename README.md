# Aegis: Risk-Aware GPU Workload Control Plane for LLM Inference

## Overview

Aegis is an experimental GPU workload control plane for LLM inference that uses short-horizon resource forecasts to make more proactive workload placement decisions.

Rather than scheduling from current GPU conditions alone, Aegis combines memory and compute telemetry with utilization forecasts and forecast uncertainty to identify workers that may be approaching contention. Prediction remains advisory: worker health, memory capacity, topology constraints, and commit-time resource availability remain authoritative.

Aegis is evaluated in a simulated multi-GPU environment and includes a telemetry boundary modeled around NVIDIA DCGM metrics. The project explores a central systems question: can prediction improve GPU scheduling while preserving correctness when telemetry, forecasts, and cluster state are imperfect? 

## The Problem

GPU inference capacity changes continuously as workloads consume memory and compute resources over time. A worker that appears suitable from current utilization alone may already be trending toward contention, making purely reactive placement inefficient.

Concurrency creates a separate correctness problem. Multiple scheduling decisions can observe the same available GPU capacity before either placement is committed, allowing workloads to independently select resources that cannot support them together.

LLM inference adds further constraints including memory demand, KV-cache requirements, model locality, GPU topology, latency targets, and workload priority. Aegis combines these constraints with current resource state, predicted pressure, and forecast uncertainty while preserving authoritative resource checks at placement time.

## Architecture

Aegis separates cluster state, telemetry, forecasting, risk evaluation, scheduling, and workload lifecycle management into distinct control-plane components.

When a workload enters the scheduling path, Aegis first filters workers using hard constraints including health, available GPU memory, and topology requirements. Eligible workers are then evaluated using current resource conditions, model locality, predicted memory and compute pressure, and forecast uncertainty.

Selection is intentionally separated from placement commit. After a worker is selected, Aegis revalidates the workload and worker against current cluster state before atomically reserving GPU memory and recording the assignment. This protects placement correctness when resource availability or worker state changes concurrently.

The resulting control path is:

`telemetry → forecast → uncertainty → risk-aware selection → commit-time revalidation → atomic reservation`

## Evaluation

### Benchmark Results

Aegis includes a synthetic benchmark comparing baseline, predictive, and risk-aware scheduling across the same stream of 48 simulated workloads.

| Forecast Regime | Policy | Placements | Rejections | Pressure Steps | Contention Transitions |
|---|---|---:|---:|---:|---:|
| Steady Forecast | Baseline | 48 | 0 | 34 | 8 |
| Steady Forecast | Predictive | 48 | 0 | 33 | 10 |
| Steady Forecast | Risk-Aware | 48 | 0 | 27 | 27 |
| Rising Forecast Pressure | Baseline | 48 | 0 | 34 | 8 |
| Rising Forecast Pressure | Predictive | 48 | 0 | 27 | 9 |
| Rising Forecast Pressure | Risk-Aware | 48 | 0 | 22 | 16 |
| Forecast Uncertainty | Baseline | 48 | 0 | 34 | 8 |
| Forecast Uncertainty | Predictive | 48 | 0 | 37 | 3 |
| Forecast Uncertainty | Risk-Aware | 48 | 0 | 22 | 16 |
| Stale Forecast | Baseline | 48 | 0 | 34 | 8 |
| Stale Forecast | Predictive | 48 | 0 | 34 | 8 |
| Stale Forecast | Risk-Aware | 48 | 0 | 34 | 8 |

All policies placed all 48 workloads without rejection. Under rising forecast pressure, risk-aware scheduling reduced pressure exposure from 34 baseline steps to 22, approximately a 35% reduction. Under forecast uncertainty, predictive scheduling alone increased pressure exposure to 37 steps, while risk-aware scheduling recorded 22.

The results also expose tradeoffs: risk-aware scheduling reduced sustained pressure but produced more contention transitions in several scenarios. Aegis therefore does not assume that prediction universally improves scheduling; forecast quality and uncertainty materially affect placement behavior.

The stale-forecast scenario validates degraded behavior. Once predictions exceed the freshness window, predictive and risk-aware scheduling reproduce baseline behavior rather than acting on outdated forecasts.

These results measure control-plane behavior in Aegis's simulator, not physical GPU throughput, inference latency, or production hardware performance.

### Failure-Path Validation

Aegis also tests correctness under failure and concurrency conditions that are not captured by the benchmark:

- **Concurrent overcommit:** two workloads race for capacity that can satisfy only one; exactly one placement commits.
- **Worker state change:** a worker becomes unhealthy between selection and commit; placement is rejected without consuming resources or modifying the workload assignment.
- **Stale forecasts:** predictive policies discard outdated predictions and reproduce baseline scheduling behavior.
- **Invalid telemetry:** stale, malformed, or worker-mismatched DCGM samples are rejected before cluster state is mutated.
- **Reservation safety:** external telemetry cannot increase allocatable GPU memory above capacity already available to the control plane.

These tests validate the central safety property of Aegis: prediction and telemetry may influence worker selection, but authoritative cluster state determines whether a placement can actually commit.

## Scheduling Model

### LLM Workloads

Aegis models LLM inference workloads explicitly because placement depends on more than available GPU memory. Each workload can include model identity, token volume, batch size, total GPU memory, KV-cache demand, estimated compute intensity, latency SLO, priority, expected duration, and topology requirements.

`RequiredMemoryMB` represents the total GPU memory reservation, while `KVCacheMemoryMB` identifies the portion associated with KV-cache demand, preventing memory from being double counted.

Workloads also carry lifecycle state and placement metadata, allowing Aegis to coordinate scheduling, execution, recovery, completion, and resource release. 

### GPU Topology

Aegis treats topology as a hard placement constraint rather than a scoring preference. Workers are modeled with node, rack, GPU index, NVLink domain, and interconnect information, while workloads can specify corresponding topology requirements.

Workers that do not satisfy those requirements are excluded before scoring, and topology is revalidated again at placement commit so a favorable utilization or risk score can never override an infrastructure constraint.

### Baseline Scheduling

Aegis first filters workers using hard constraints: worker health, sufficient GPU memory, and topology compatibility. Eligible workers are then scored using available memory, compute and memory utilization, active workload count, and model locality.

Greater resource headroom improves a worker's score, while existing utilization and workload pressure reduce it. Model locality provides an advantage when the requested model is already cached on a worker.

Hard constraints remain separate from scoring, so a favorable score cannot override health, capacity, or topology requirements. Deterministic tie-breaking keeps placement reproducible when candidates receive equivalent scores.

## Predictive Scheduling

### Telemetry & Forecasting

Aegis maintains timestamped GPU memory and compute-utilization telemetry for each worker. Recent observations are used to estimate short-horizon utilization trends, allowing the scheduler to identify developing resource pressure that may not yet be visible from the current cluster snapshot.

The forecasting layer projects memory and compute utilization across a defined horizon. Forecasts remain advisory: they influence selection among otherwise eligible workers but never replace health, memory, topology, or commit-time resource checks.

Forecasts are freshness-limited. If predictive information is missing or older than the configured freshness window, Aegis ignores it and falls back to current-state baseline scheduling.

### Forecast Uncertainty

Aegis tracks recent prediction error using mean absolute error (MAE) for memory and compute utilization independently. This gives the scheduler an explicit measure of how reliable recent forecasts have been rather than treating every prediction as equally trustworthy.

For observed utilization \(y_i\) and predicted utilization \(\hat{y}_i\), forecast error is measured as:

$$
\mathrm{MAE}=\frac{1}{n}\sum_{i=1}^{n}\left|y_i-\hat{y}_i\right|
$$

Higher recent error increases the risk associated with a forecast, making workers with uncertain predictions less attractive even when their predicted utilization appears favorable. This uncertainty is incorporated directly into Aegis's placement-risk model.

### Placement Risk Model

Aegis defines placement risk as the maximum projected pressure across GPU memory and compute after accounting for forecast uncertainty and the additional demand introduced by the workload.

For workload \(w\), worker \(g\), and forecast horizon \(h\):

$$
R(w,g,t)=
\max
\left(
\hat{M}_{g,t+h}+E^M_g+D^M_w,\;
\hat{C}_{g,t+h}+E^C_g
\right)
$$

where:

- \(\hat{M}_{g,t+h}\) and \(\hat{C}_{g,t+h}\) are predicted memory and compute utilization.
- \(E^M_g\) and \(E^C_g\) are recent memory and compute forecast error.
- \(D^M_w\) is the additional memory pressure introduced by the workload.

Adding forecast error makes the model intentionally conservative: two workers with similar predicted utilization can receive different risk scores when one has less reliable recent forecasts. Workload demand is also included before placement so Aegis evaluates expected post-placement pressure rather than the worker in isolation.

The score is a deterministic placement-risk heuristic, not a calibrated probability of contention.

### Scheduling Objective

Aegis treats risk-aware worker selection as a constrained optimization problem. Workers must first satisfy health, memory-capacity, and topology requirements. Among eligible workers, the scheduler selects the candidate with the lowest placement risk:

$$
g^*=\arg\min_{g\in\mathcal{E}(w,t)} R(w,g,t)
$$

where \(\mathcal{E}(w,t)\) is the set of workers eligible to execute workload \(w\) at time \(t\), and \(R(w,g,t)\) is the placement-risk function defined above.

When risk values do not distinguish otherwise valid candidates, Aegis uses its baseline worker score as a deterministic tie-break. This preserves considerations such as resource headroom, current utilization, active workload count, and model locality while keeping hard infrastructure constraints separate from optimization.

## NVIDIA DCGM-Compatible Telemetry

Aegis includes a telemetry boundary modeled around NVIDIA Data Center GPU Manager (DCGM) metrics, providing a path from simulated scheduling toward integration with real accelerator infrastructure.

The current adapter models GPU utilization and framebuffer-memory signals corresponding to metrics such as `DCGM_FI_DEV_GPU_UTIL`, `DCGM_FI_DEV_FB_TOTAL`, `DCGM_FI_DEV_FB_FREE`, and `DCGM_FI_DEV_FB_USED`. Samples are validated for worker identity, timestamp freshness, utilization range, and memory consistency before they can affect cluster state.

Telemetry is intentionally subordinate to Aegis's reservation state. If DCGM reports more physically free memory than Aegis considers available, the scheduler does not increase allocatable capacity because that memory may already be reserved by workloads. If observed free memory is lower, Aegis can reduce available capacity accordingly.

This preserves a key invariant:

> **External telemetry may make Aegis more conservative, but it cannot manufacture capacity that the control plane has already reserved.**

The current implementation provides a validated DCGM-compatible ingestion boundary rather than a live DCGM Exporter deployment.

## Correctness & Failure Safety

### Atomic Placement

Worker selection and resource reservation are intentionally separate. A scheduler may select a worker from a cluster snapshot, but that decision is not authoritative because cluster conditions can change before placement is committed.

At commit time, Aegis acquires exclusive access to cluster state and revalidates worker health, available GPU memory, topology compatibility, and workload state. Only after those checks succeed are GPU memory, active-workload count, and workload assignment updated atomically.

This protects the control plane from time-of-check/time-of-use (TOCTOU) races between selection and placement.

### Concurrent Overcommit Protection

Aegis prevents concurrent scheduling operations from independently claiming the same GPU capacity. Even if multiple workloads select the same worker from an earlier snapshot, each placement must pass commit-time validation against the worker's current authoritative state.

A concurrency test deliberately races two workloads for capacity that can satisfy only one. Exactly one placement succeeds; the competing placement receives a conflict without partially modifying worker or workload state.

This ensures the core reservation invariant:

> **Committed GPU memory cannot exceed the capacity available at placement time.**


### Stale Forecast & State-Change Safety

Predictions are useful only while they reflect current cluster conditions. Aegis timestamps forecasts and enforces a freshness window; when a forecast becomes stale, predictive and risk-aware scheduling ignore it and fall back to baseline current-state evaluation.

The benchmark explicitly tests this degraded mode. With stale forecasts, predictive and risk-aware policies reproduce baseline placement and pressure behavior rather than making decisions from outdated predictions.

Aegis also handles state changes between selection and commit. If a worker is selected while healthy but becomes unhealthy before placement is committed, commit-time revalidation rejects the placement without consuming GPU memory or modifying the workload assignment.

Together, these mechanisms enforce a broader principle:

> **Prediction may influence placement, but it never overrides authoritative cluster state.**

### Telemetry Safety

External telemetry is validated before it can influence cluster state. Aegis rejects stale samples, invalid utilization values, inconsistent framebuffer-memory measurements, and samples that do not match the target worker.

Invalid samples fail before cluster state is mutated. Valid samples may make scheduling more conservative when observed free memory is lower than Aegis expects, but they cannot increase allocatable memory above the control plane's existing reservation state.

This keeps external observations separate from scheduling authority: telemetry describes physical GPU conditions, while Aegis remains responsible for deciding how much capacity is actually available for new placements.

### Worker Failure & Recovery

Aegis monitors worker heartbeats and excludes unhealthy workers from new placements. When a worker fails after workloads have already been assigned, the reconciliation path identifies affected workloads and releases their reserved resources.

Checkpointable workloads can transition into recovery and return to the scheduling path, while workloads that cannot be recovered transition to a failed state. Resource cleanup and workload state changes are coordinated so failed workers do not leave stale assignments or reserved GPU capacity behind.

## Workload Lifecycle

Aegis tracks workloads through explicit lifecycle states including `PENDING`, `QUEUED`, `PLACED`, `RUNNING`, `PREEMPTED`, `RECOVERING`, `COMPLETED`, and `FAILED`.

Placement reserves GPU resources and records the assigned worker. As workloads execute, lifecycle transitions keep scheduling state synchronized with resource ownership. Completion and failure paths release reserved capacity so it can safely return to the scheduling pool.

Recovery is integrated into the same lifecycle. Checkpointable workloads affected by worker failure can transition through `RECOVERING` and re-enter scheduling, while workloads that cannot be recovered transition to `FAILED`.

This lifecycle model allows Aegis to reason about both **where a workload should run** and **whether its resource reservation is still valid throughout execution**.

## HTTP API

Aegis exposes a lightweight HTTP API for interacting with control-plane state.

| Endpoint | Method | Purpose |
|---|---|---|
| `/healthz` | `GET` | Check control-plane health |
| `/v1/workers` | `POST` | Register a worker |
| `/v1/workers` | `GET` | List registered workers |
| `/v1/workloads` | `POST` | Submit a workload |
| `/v1/workloads` | `GET` | List workloads and placement state |

The API provides an external boundary for worker registration and workload submission while keeping scheduling, forecasting, telemetry, and resource coordination inside the control plane.

## Running Aegis

### Requirements

- Go 1.22 or later

### Run the Control Plane

From the repository root:

```bash
go run ./cmd/aegis
```

The API starts on port `8080`.

Verify the service:

```bash
curl http://localhost:8080/healthz
```

Expected response:

```json
{"status":"ok"}
```

### Run the Benchmark

Execute the synthetic scheduling benchmark:

```bash
go run ./cmd/benchmark
```

This compares baseline, predictive, and risk-aware scheduling across the benchmark forecast regimes.

### Run Tests

```bash
go test ./...
go test -race ./...
go vet ./...
```

The race-enabled test suite validates concurrency-sensitive paths including placement and shared cluster-state access.

## Repository Structure

```text
aegis-llm-control-plane/
├── cmd/
│   ├── aegis/          # Control-plane entry point
│   └── benchmark/      # Synthetic scheduling benchmark
│
├── internal/
│   ├── api/            # HTTP API
│   ├── cluster/        # Workers, workloads, state, and atomic reservations
│   ├── predictor/      # Short-horizon utilization forecasting
│   ├── risk/           # Forecast uncertainty and placement-risk evaluation
│   ├── scheduler/      # Baseline, predictive, and risk-aware scheduling
│   ├── simulator/      # Workload lifecycle and benchmark simulation
│   └── telemetry/      # Telemetry collection and DCGM-compatible ingestion
│
├── go.mod
├── go.sum
└── README.md
```
The architecture keeps scheduling policy separate from cluster-state authority. Telemetry, forecasting, and risk models inform worker selection, while the cluster layer remains responsible for authoritative resource state and atomic placement commits.

## Limitations

Aegis is an experimental control-plane prototype evaluated with simulated workers, synthetic workloads, and generated telemetry. The benchmark measures scheduling behavior and resource-pressure outcomes inside the simulator; it does not represent physical GPU throughput, model inference latency, or production hardware performance.

The current NVIDIA DCGM integration provides a validated DCGM-compatible telemetry boundary rather than a live DCGM Exporter deployment. Forecasting is intentionally lightweight, and MAE-based uncertainty is used as a deterministic risk signal rather than a calibrated probability of contention.

Cluster state is currently maintained in memory within a single control-plane process. A production deployment would require durable and replicated state, distributed coordination, live accelerator telemetry, authentication and authorization, high availability, and broader observability.

These limitations are intentional: Aegis focuses on the control-plane mechanisms required to evaluate predictive GPU scheduling while preserving correctness under stale forecasts, concurrent placement, worker-state changes, and imperfect telemetry.
