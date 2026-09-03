package risk

type Level string

const (
	LevelLow      Level = "LOW"
	LevelModerate Level = "MODERATE"
	LevelHigh     Level = "HIGH"
	LevelCritical Level = "CRITICAL"
)

type Breakdown struct {
	MemoryPressure      float64
	ComputePressure     float64
	ForecastUncertainty float64
	WorkloadDemand      float64
}

type PlacementRisk struct {
	WorkerID   string
	WorkloadID string

	Score     float64
	Level     Level
	Breakdown Breakdown
}
