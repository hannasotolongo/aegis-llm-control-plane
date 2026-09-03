# Aegis

Aegis is a GPU workload control plane I'm building in Go to explore a simple question:

**Can a scheduler use short term resource predictions to avoid GPU contention before it happens?**

Most scheduling decisions are based on what resources look like right now. Aegis experiments with adding recent GPU memory and compute behavior to those decisions so the scheduler can account for where resource pressure may be heading.

The system maintains cluster state, schedules workloads across GPU workers, collects telemetry, produces short-horizon resource forecasts, and can incorporate those forecasts into placement decisions.

Because predictions can be wrong, Aegis falls back to deterministic scheduling when a forecast is not considered reliable.

> **Status:** Aegis is under active development. The current predictor is a simple trend-based forecaster, not a machine-learning model. Experimental evaluation is still in progress.

## Why Aegis?

A GPU worker that looks like a good placement target now may become heavily utilized shortly afterward.

I wanted to explore whether recent resource behavior could help a scheduler anticipate that pressure instead of waiting until contention has already occurred.

I started by building deterministic scheduling and cluster-state management before adding prediction as an additional scheduling signal.

The goal is to eventually compare three approaches:

1. Deterministic scheduling using current cluster state
2. Predictive scheduling using short-term forecasts
3. Confidence-aware scheduling that uses predictions only when they are considered reliable

## Architecture

```text
                    HTTP API
                       |
                       v
                 Cluster State
                workers/workloads
                       |
                       v
                    Scheduler
                   /         \
                  /           \
        Current State     Predictions
                              ^
                              |
                           Predictor
                              ^
                              |
                           Telemetry
```

Prediction is kept separate from the core scheduler so Aegis can continue making placement decisions when forecasts are unavailable or unreliable.

## Scheduling

Workers are first filtered using hard constraints such as health, available GPU memory, and topology requirements.

Eligible workers are then scored using signals including:

- memory headroom
- compute and memory utilization
- active workload count
- model locality

Aegis also supports workload priority and aging when determining which pending workload should be scheduled next.

Placement is committed atomically through a concurrency-safe state store. This prevents multiple scheduling operations from independently reserving the same GPU resources.

## Prediction

Aegis collects GPU and workload telemetry over time and uses recent samples to estimate near-term memory and compute utilization.

For example:

```text
Memory utilization:

61% -> 68% -> 77% -> 86%
                     |
                     v
              next utilization
```

The current predictor uses recent utilization trends rather than a trained machine-learning model.

Each prediction includes estimated memory utilization, compute utilization, expected contention, and a confidence value.

When a prediction satisfies the configured confidence policy, predicted resource pressure can influence worker scoring. When it does not, the scheduler falls back to deterministic behavior.

This lets Aegis experiment with prediction without making the scheduler completely dependent on forecasts.

## Failure Handling

Aegis tracks worker heartbeats and health.

If a worker becomes unavailable, the control plane reconciles workloads assigned to it. Checkpointable workloads can enter recovery and become eligible for future scheduling, while non-checkpointable workloads can transition to a failed state.

Resource reservations are also released as workloads leave their placement.

## Current Capabilities

- Concurrent worker and workload state management
- GPU memory-aware workload placement
- Workload priority and aging
- Model-locality-aware scoring
- Atomic resource reservation and release
- Worker heartbeat and failure reconciliation
- Telemetry collection and CSV export
- Simulated resource-pressure scenarios
- Short-horizon resource forecasting
- Confidence-aware predictive scheduling
- Deterministic fallback
- Unit and integration testing
- Go race-condition testing

## Project Structure

```text
aegis-llm-control-plane/
├── cmd/
│   └── aegis/          # application entry point
├── internal/
│   ├── api/            # HTTP API
│   ├── cluster/        # workers, workloads, and state
│   ├── predictor/      # forecasting and prediction policy
│   ├── scheduler/      # workload ordering and placement
│   ├── simulator/      # resource scenarios
│   └── telemetry/      # telemetry collection and export
├── go.mod
└── README.md
```

## Running Aegis

Clone the repository:

```bash
git clone https://github.com/hannasotolongo/aegis-llm-control-plane.git
cd aegis-llm-control-plane
```

Run the tests:

```bash
go test ./...
```

Run the race detector:

```bash
go test -race ./...
```

Start Aegis:

```bash
go run ./cmd/aegis
```

Check the service:

```bash
curl http://localhost:8080/healthz
```

## Next Steps

The next phase is focused on measuring whether prediction actually improves scheduling.

Planned work includes prediction freshness and expiration, better visibility into why placement decisions are made, reproducible scheduler experiments, controlled burst and distribution-shift testing, and evaluation using public GPU workload traces.

The larger goal is to determine **when prediction helps, when it hurts, and whether confidence aware fallback can preserve reliable scheduling when workload behavior changes.**
