package telemetry

import "testing"

func TestLabelPressureNormal(t *testing.T) {
	record := Record{
		MemoryUtilization:  60,
		ComputeUtilization: 70,
	}

	label := LabelPressure(record)

	if label.MemoryPressure {
		t.Fatal("expected no memory pressure")
	}
	if label.ComputePressure {
		t.Fatal("expected no compute pressure")
	}
	if label.Contention {
		t.Fatal("expected no contention")
	}
}

func TestLabelPressureMemory(t *testing.T) {
	record := Record{
		MemoryUtilization:  90,
		ComputeUtilization: 50,
	}

	label := LabelPressure(record)

	if !label.MemoryPressure {
		t.Fatal("expected memory pressure")
	}
	if label.ComputePressure {
		t.Fatal("expected no compute pressure")
	}
	if !label.Contention {
		t.Fatal("expected contention")
	}
}

func TestLabelPressureCompute(t *testing.T) {
	record := Record{
		MemoryUtilization:  50,
		ComputeUtilization: 95,
	}

	label := LabelPressure(record)

	if label.MemoryPressure {
		t.Fatal("expected no memory pressure")
	}
	if !label.ComputePressure {
		t.Fatal("expected compute pressure")
	}
	if !label.Contention {
		t.Fatal("expected contention")
	}
}

func TestLabelPressureBoundary(t *testing.T) {
	record := Record{
		MemoryUtilization:  85,
		ComputeUtilization: 90,
	}

	label := LabelPressure(record)

	if !label.MemoryPressure {
		t.Fatal("expected memory pressure at threshold")
	}
	if !label.ComputePressure {
		t.Fatal("expected compute pressure at threshold")
	}
	if !label.Contention {
		t.Fatal("expected contention at threshold")
	}
}
