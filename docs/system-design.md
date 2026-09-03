# Aegis System Design

## 1. Problem Statement

Large scale LLM inference systems must distribute requests across GPU resources while maintaining low latency, high resource utilization, and service availability.

Simple scheduling strategies such as round-robin or least-loaded placement react primarily to the current state of a cluster. They may not account for upcoming demand, GPU memory pressure, KV-cache locality, workload priority, topology, or worker failures.

Aegis is a distributed control plane for LLM inference infrastructure. It maintains cluster state, monitors GPU workers, makes workload placement and routing decisions, detects failures, and reconciles workloads when observed cluster state diverges from desired state.

The system will be evaluated against simpler scheduling policies using reproducible simulated GPU workloads and failure scenarios.

## 2. Goals

Aegis will be designed to:

- maintain a current view of GPU workers, resources, and active workloads
- schedule inference workloads according to available resources and cluster state
- incorporate GPU memory pressure, utilization, topology, and KV-cache locality into scheduling decisions
- support workload priorities and backpressure during cluster saturation
- detect unhealthy workers using heartbeat-based failure detection
- recover workloads following simulated worker or GPU failures
- reconcile desired workload state with observed cluster state
- expose scheduling, resource, latency, and recovery telemetry
- support reproducible workload and failure simulation
- benchmark advanced scheduling policies against baseline algorithms

## 3. Non-Goals

The initial implementation will not:

- claim to operate a production GPU data center
- require access to a large physical GPU cluster
- implement a complete LLM serving runtime
- replace Kubernetes
- claim production-scale fault tolerance or performance

GPU cluster behavior will initially be simulated so scheduling and distributed-systems behavior can be tested reproducibly.
## 4. Functional Requirements

Aegis must provide the following core capabilities.

### Worker Management

- register and deregister GPU workers
- track worker identity, health, capacity, and current utilization
- receive periodic heartbeats from workers
- mark workers as suspected or unhealthy when heartbeats are missed
- prevent new workloads from being placed on unhealthy workers

### Workload Management

- accept new inference workloads
- track workload priority and resource requirements
- maintain desired and observed workload state
- queue workloads when sufficient GPU capacity is unavailable
- support workload reassignment following worker failure

### Scheduling

- provide a baseline round-robin scheduler
- provide a least-loaded scheduler
- support an advanced Aegis scheduling policy
- score candidate workers using resource pressure and workload characteristics
- reject or queue workloads when placement constraints cannot be satisfied

### Recovery

- detect worker failure
- identify workloads affected by the failure
- select replacement workers when capacity exists
- recreate workload placement state safely
- avoid duplicate recovery operations

### Observability

- expose scheduler decision latency
- expose worker health and utilization
- expose workload queue depth
- expose placement and recovery events
- expose workload latency and SLO violations
- record benchmark results for scheduler comparison
## 5. Initial Service-Level Objectives

The simulator will use explicit service-level objectives so scheduling policies can be evaluated quantitatively.

Initial targets are design goals rather than claims of production performance.

### Scheduling Decision Latency

The control plane should make a placement decision within:

- P50: less than 10 ms
- P95: less than 50 ms
- P99: less than 100 ms

### Queue Latency

Latency between workload admission and successful placement will be measured at:

- P50
- P95
- P99

The primary objective is for the Aegis scheduler to reduce tail queue latency relative to baseline scheduling policies under bursty load.

### Time to First Token

For simulated LLM inference requests, Time to First Token (TTFT) represents the delay between request arrival and generation of the first output token.

Aegis will measure TTFT and evaluate whether scheduling decisions reduce P95 and P99 TTFT under resource contention.

### Throughput

Throughput will be measured as successfully completed inference requests per second.

Scheduling policies will be compared under identical workload traces.

### GPU Utilization

The simulator will measure GPU compute and memory utilization over time.

The scheduler should improve aggregate utilization without causing excessive queue latency or memory pressure.

### Recovery Time

Recovery time is the interval between a worker being declared unhealthy and its affected workloads becoming successfully placed on healthy workers.

The recovery engine will be evaluated using P50, P95, and P99 recovery latency.

### SLO Violations

A workload produces an SLO violation when its simulated latency exceeds the threshold associated with its priority class.

The benchmark suite will compare total SLO violations across scheduling policies.
## 6. Cluster Model

Aegis models a cluster as a collection of GPU workers managed by the control plane.

Each worker represents a schedulable execution target and maintains the following state:

- worker ID
- node ID
- GPU type
- total GPU memory
- available GPU memory
- current compute utilization
- current memory utilization
- active workload count
- health state
- last heartbeat timestamp
- topology metadata
- cached model or KV-cache metadata

Workers may transition between:

- `HEALTHY`
- `SUSPECTED`
- `UNHEALTHY`
- `DRAINING`

A worker marked `UNHEALTHY` must not receive new workloads.

A worker marked `DRAINING` remains active long enough for existing workloads to complete or migrate, but does not receive new placements.

### GPU Topology

The simulator will model communication relationships between workers.

Topology metadata may include:

- same GPU
- same node
- same high-bandwidth interconnect domain
- remote node

Scheduling policies may assign different placement costs depending on communication distance.

This allows Aegis to model cases where placing related workloads on physically or logically closer GPUs reduces communication overhead.
## 7. Workload Model

A workload represents an LLM inference execution request or persistent inference worker that requires GPU capacity.

Each workload contains:

- workload ID
- model ID
- arrival timestamp
- priority class
- GPU memory requirement
- estimated compute requirement
- expected execution duration
- latency SLO
- checkpoint capability
- model locality requirements
- KV-cache locality information
- current state
- assigned worker

Workloads may transition between:

- `PENDING`
- `QUEUED`
- `PLACED`
- `RUNNING`
- `PREEMPTED`
- `RECOVERING`
- `COMPLETED`
- `FAILED`

### Priority Classes

The initial simulator will support three workload priorities:

- `CRITICAL`
- `STANDARD`
- `BATCH`

Critical workloads receive the strongest latency guarantees.

Standard workloads represent normal interactive inference traffic.

Batch workloads prioritize throughput and cluster utilization over latency.

The scheduler may delay or preempt lower-priority workloads when necessary to protect higher-priority latency objectives.
## 8. KV-Cache Locality Model

LLM inference systems may retain intermediate attention state in a KV cache.

Aegis will model whether useful cached state already exists on a worker for a given model or request prefix.

A scheduler may prefer a worker with reusable cache state even when another worker has slightly lower utilization.

The simulator will assign a cache-locality score representing the estimated benefit of routing a workload to a particular worker.

This allows scheduling policies to evaluate the tradeoff between:

- current GPU load
- available memory
- queue depth
- cache reuse
- expected latency

KV-cache behavior will initially be simulated rather than implemented through a complete LLM serving runtime.
## 9. Control Plane Architecture

Aegis separates cluster coordination from workload execution.

The control plane maintains cluster state, makes scheduling decisions, detects failures, and reconciles workloads. GPU workers execute assigned workloads and report their current state.

The initial control plane contains five primary components.

### API / Admission Controller

The admission controller receives incoming workload requests and validates:

- workload identity
- priority
- resource requirements
- model requirements
- latency objectives

Valid workloads are recorded as desired state and submitted for scheduling.

When the cluster cannot safely accept additional work, the admission controller may queue or reject requests rather than allowing uncontrolled resource saturation.

### Cluster State Manager

The cluster state manager maintains Aegis's current view of:

- registered workers
- worker health
- GPU capacity
- GPU memory availability
- active workloads
- workload assignments
- queue state
- topology information
- cache-locality metadata

Scheduling and recovery components use this state when making decisions.

### Scheduler

The scheduler selects a worker for each schedulable workload.

Candidate workers are first filtered using hard constraints such as:

- worker health
- available GPU memory
- workload requirements
- placement restrictions

Remaining workers are scored according to the active scheduling policy.

Aegis will support multiple interchangeable scheduling policies so their behavior can be benchmarked under identical workload traces.

### Failure Detector

Workers periodically send heartbeats to the control plane.

The failure detector tracks heartbeat timing and transitions workers through health states when expected heartbeats are missed.

A missed heartbeat does not immediately imply permanent failure. A worker may first enter a suspected state before being declared unhealthy.

This reduces unnecessary recovery caused by temporary network delay or transient worker stalls.

### Reconciliation and Recovery Engine

The reconciler continuously compares desired workload state with observed cluster state.

For example:

Desired state:

    Workload A should be running.

Observed state:

    Worker 7 failed.
    Workload A is no longer running.

The reconciler identifies this divergence and initiates recovery.

Recovery may return the workload to the scheduling queue, select another healthy worker, and update cluster state after successful placement.
## 10. Worker Architecture

Each GPU worker runs an Aegis worker agent.

The worker agent communicates with the control plane and is responsible for:

- worker registration
- heartbeat transmission
- resource telemetry
- workload lifecycle reporting
- placement acknowledgement
- failure reporting

The initial implementation will use simulated GPU resources.

Later implementations may connect the worker agent to real GPU telemetry and inference runtimes without requiring the scheduling architecture to be redesigned.
## 11. Cluster State Strategy

The initial Aegis control plane will maintain cluster state in memory behind a storage abstraction.

Scheduling components will interact with a state-store interface rather than directly accessing a concrete storage implementation.

The initial implementation will provide an in-memory state store containing:

- worker registrations
- worker health
- worker resource state
- workload state
- workload assignments
- queue state
- topology metadata
- cache-locality metadata

This approach keeps the first implementation deterministic and allows scheduling behavior to be developed and tested without introducing an external distributed database.

The storage interface will be designed so a durable implementation can be introduced later without requiring the scheduler or recovery engine to be rewritten.

### Failure Semantics

The initial in-memory implementation does not survive complete control-plane process failure.

Worker failure recovery will be implemented before control-plane state recovery.

A later phase will introduce durable state and controller recovery semantics. At that point, Aegis must define how stale state, duplicate operations, and conflicting observations are reconciled.

### Design Principle

The scheduler must not depend directly on the storage implementation.

Instead:

    Scheduler
        |
        v
    StateStore Interface
        |
        +-- InMemoryStateStore
        |
        +-- Future DurableStateStore

This separation allows storage consistency and durability mechanisms to evolve independently from scheduling policy.