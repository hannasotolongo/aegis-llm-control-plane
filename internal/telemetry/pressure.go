package telemetry

const (
	DefaultMemoryPressureThreshold  = 85.0
	DefaultComputePressureThreshold = 90.0
)

type PressureLabel struct {
	MemoryPressure  bool
	ComputePressure bool
	Contention      bool
}

func LabelPressure(record Record) PressureLabel {
	memoryPressure := record.MemoryUtilization >= DefaultMemoryPressureThreshold
	computePressure := record.ComputeUtilization >= DefaultComputePressureThreshold

	return PressureLabel{
		MemoryPressure:  memoryPressure,
		ComputePressure: computePressure,
		Contention:      memoryPressure || computePressure,
	}
}
