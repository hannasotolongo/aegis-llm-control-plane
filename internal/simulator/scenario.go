package simulator

import "fmt"

type Scenario string

const (
	ScenarioSteady            Scenario = "STEADY"
	ScenarioRisingPressure    Scenario = "RISING_PRESSURE"
	ScenarioBurst             Scenario = "BURST"
	ScenarioDistributionShift Scenario = "DISTRIBUTION_SHIFT"
)

type Utilization struct {
	Compute float64
	Memory  float64
}

func (s Scenario) Utilization(step, workerIndex int) (Utilization, error) {
	switch s {
	case ScenarioSteady:
		return steadyUtilization(workerIndex), nil
	case ScenarioRisingPressure:
		return risingPressureUtilization(step, workerIndex), nil
	case ScenarioBurst:
		return burstUtilization(step, workerIndex), nil
	case ScenarioDistributionShift:
		return distributionShiftUtilization(step, workerIndex), nil
	default:
		return Utilization{}, fmt.Errorf("unknown simulator scenario %q", s)
	}
}

func steadyUtilization(workerIndex int) Utilization {
	offset := float64(workerIndex * 2)

	return Utilization{
		Compute: clampUtilization(42 + offset),
		Memory:  clampUtilization(48 + offset),
	}
}

func risingPressureUtilization(step, workerIndex int) Utilization {
	offset := float64(workerIndex * 2)

	return Utilization{
		Compute: clampUtilization(30 + float64(step)*5 + offset),
		Memory:  clampUtilization(35 + float64(step)*5 + offset),
	}
}

func burstUtilization(step, workerIndex int) Utilization {
	offset := float64(workerIndex * 2)

	if step >= 6 && step <= 8 {
		return Utilization{
			Compute: clampUtilization(96 + offset),
			Memory:  clampUtilization(92 + offset),
		}
	}

	return Utilization{
		Compute: clampUtilization(38 + offset),
		Memory:  clampUtilization(44 + offset),
	}
}

func distributionShiftUtilization(step, workerIndex int) Utilization {
	offset := float64(workerIndex * 2)

	if step < 8 {
		return Utilization{
			Compute: clampUtilization(30 + float64(step)*5 + offset),
			Memory:  clampUtilization(35 + float64(step)*5 + offset),
		}
	}

	if step < 11 {
		return Utilization{
			Compute: clampUtilization(95 - float64(workerIndex)*3),
			Memory:  clampUtilization(96 - float64(workerIndex)*2),
		}
	}

	return Utilization{
		Compute: clampUtilization(35 + offset),
		Memory:  clampUtilization(40 + offset),
	}
}

func clampUtilization(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}
