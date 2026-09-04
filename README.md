# Aegis: Risk-Aware GPU Workload Control Plane for LLM Inference

## Overview

Aegis is an experimental GPU workload control plane for LLM inference that uses short-horizon resource forecasts to make more proactive workload placement decisions.

Rather than scheduling from current GPU conditions alone, Aegis combines memory and compute telemetry with utilization forecasts and forecast uncertainty to identify workers that may be approaching contention. Prediction remains advisory: worker health, memory capacity, topology constraints, and commit-time resource availability remain authoritative.

Aegis is evaluated in a simulated multi-GPU environment and includes a telemetry boundary modeled around NVIDIA DCGM metrics. The project explores a central systems question: can prediction improve GPU scheduling while preserving correctness when telemetry, forecasts, and cluster state are imperfect? 

## The Problem
GPU capacity is expensive and continuously changing. A placement that appears efficient from current utilization can become inefficient shortly afterward as memory pressure, compute demand, and competing workloads evolve. Scheduling only from the latest resource snapshot can therefore place work on a GPU that is already trending toward contention.

Concurrency creates a separate correctness problem. Multiple scheduling decisions may observe the same available capacity before either placement is committed, allowing each workload to independently select resources that cannot support them together.

Aegis addresses both problems by using recent resource behavior to make placement more proactive while coordinating final resource reservation against authoritative cluster state.

## Architecture

Aegis separates cluster state, telemetry, forecasting, risk evaluation, scheduling, and workload lifecycle management into distinct control-plane components.

When a workload enters the scheduling path, Aegis first identifies workers that satisfy hard constraints such as health, available GPU memory, and topology requirements. Eligible workers are then evaluated using current resource conditions and, when available, short-horizon utilization forecasts and forecast uncertainty.

Worker selection is intentionally separated from placement commit. After a worker is selected, Aegis revalidates the workload and worker against current cluster state before atomically reserving GPU memory and recording the assignment. This prevents a decision made from an earlier snapshot from being committed after resource availability or worker state has changed.

The resulting scheduling path is:

`telemetry → forecast → uncertainty → risk-aware selection → commit-time revalidation → atomic reservation`

## Evaluation & Results

Aegis includes a synthetic benchmark comparing baseline, predictive, and risk-aware scheduling across the same stream of 48 simulated workloads.

| Scenario | Baseline Pressure Steps | Predictive Pressure Steps | Risk-Aware Pressure Steps |
|---|---:|---:|---:|
| Steady Forecast | 34 | 33 | 27 |
| Rising Forecast Pressure | 34 | 27 | 22 |
| Forecast Uncertainty | 34 | 37 | 22 |
| Stale Forecast | 34 | 34 | 34 |

All three policies placed all 48 workloads with zero rejections in these runs.

Under rising forecast pressure, risk-aware scheduling reduced time spent under simulated resource pressure from 34 to 22 steps, approximately a 35% reduction relative to baseline. Under forecast uncertainty, predictive scheduling alone increased pressure exposure to 37 steps, while the risk-aware policy recorded 22, demonstrating why forecast uncertainty can matter when using predictions for placement.

The stale-forecast scenario tests degraded behavior. When predictions exceed the configured freshness window, both predictive policies fall back to baseline scheduling and reproduce its placement and pressure behavior rather than acting on outdated forecasts.

These results measure control-plane behavior in Aegis's simulator. They are not measurements of physical GPU throughput, inference latency, or production hardware performance.

## Scheduling Policies

Aegis compares three placement strategies:

- **Baseline:** selects eligible workers using current resource state, capacity, topology, and model locality.
- **Predictive:** adds short-horizon memory and compute forecasts to avoid workers trending toward contention.
- **Risk-Aware:** also incorporates recent forecast error and prospective workload pressure so uncertain predictions are treated more conservatively.

Health, memory, and topology remain hard constraints under every policy. If forecasts are missing or stale, predictive policies fall back to baseline scheduling.

## Telemetry & Forecasting

Aegis maintains timestamped GPU memory and compute telemetry and uses recent observations to forecast short-horizon utilization. Forecast uncertainty is estimated from recent prediction error so less reliable forecasts can be treated more conservatively.

The telemetry layer includes a boundary modeled around NVIDIA DCGM GPU utilization and framebuffer-memory metrics. Incoming samples are validated for freshness and consistency before affecting cluster state, and observed free memory cannot override capacity already reserved by Aegis.

Forecasts are also freshness-limited. Missing or stale predictions are ignored, causing predictive policies to fall back to current-state scheduling.

## Reliability & Concurrency

Worker selection and placement commit are intentionally separated. Aegis may select a worker from a snapshot, but cluster conditions can change before that decision is committed.

At commit time, Aegis revalidates worker health, available memory, topology, and workload state while holding exclusive access to cluster state. Resource reservation and workload assignment are then applied atomically, preventing concurrent placements from claiming the same capacity.

Worker health is also monitored through heartbeats. Unhealthy workers are excluded from new placements, and affected workloads can be released for recovery or marked failed depending on their lifecycle configuration.

## Workload Lifecycle

Aegis tracks workloads beyond initial placement so reserved resources can be returned as execution completes or infrastructure fails. Completed workloads release GPU memory and active workload capacity, while workloads affected by worker failure can enter recovery or transition to a failed state. This allows scheduling decisions to operate against changing capacity rather than a cluster where resource pressure only accumulates.

## API & Running Aegis

Aegis exposes an HTTP API for worker registration, workload submission, cluster inspection, and scheduling. The control plane can be run locally without GPU hardware.

```bash
git clone https://github.com/hannasotolongo/aegis-llm-control-plane.git
cd aegis-llm-control-plane

go test ./...
go run ./cmd/aegis

## Limitations

Aegis currently evaluates scheduling behavior in a simulated multi-GPU environment rather than on physical GPU infrastructure. The benchmark measures control-plane behavior under controlled workload and forecast conditions and should not be interpreted as production GPU throughput or inference performance.

The DCGM integration currently provides a validated telemetry boundary modeled around GPU utilization and framebuffer-memory metrics rather than a live DCGM Exporter deployment. Production use would require integration with real accelerator telemetry, persistent cluster state, distributed coordination, and validation under GPU-backed LLM inference workloads.

## Repository Structure

```text
aegis-llm-control-plane/
├── cmd/
│   ├── aegis/          # Control-plane runtime
│   └── benchmark/      # Scheduling benchmark
├── internal/
│   ├── api/            # HTTP API
│   ├── cluster/        # Cluster state, workers, workloads, reservations
│   ├── predictor/      # Short-horizon utilization forecasting
│   ├── risk/           # Forecast uncertainty and placement risk
│   ├── scheduler/      # Baseline, predictive, and risk-aware scheduling
│   ├── simulator/      # Simulated workload and GPU environment
│   └── telemetry/      # Telemetry collection and DCGM-compatible ingestion
└── README.md

