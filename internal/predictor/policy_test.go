package predictor

import (
	"testing"
)

func TestDecideUsesPredictionWhenConfidenceIsHigh(t *testing.T) {
	prediction := Prediction{
		PredictedContention: true,
		Confidence:          0.81,
	}

	decision := Decide(prediction, DefaultConfidenceThreshold)

	if decision.Mode != DecisionPredictive {
		t.Fatalf("expected predictive mode, got %s", decision.Mode)
	}

	if !decision.TrustPrediction {
		t.Fatal("expected prediction to be trusted")
	}
}

func TestDecideFallsBackWhenConfidenceIsLow(t *testing.T) {
	prediction := Prediction{
		PredictedContention: true,
		Confidence:          0.09,
	}

	decision := Decide(prediction, DefaultConfidenceThreshold)

	if decision.Mode != DecisionDeterministic {
		t.Fatalf("expected deterministic mode, got %s", decision.Mode)
	}

	if decision.TrustPrediction {
		t.Fatal("expected prediction not to be trusted")
	}
}

func TestDecideBoundaryUsesPrediction(t *testing.T) {
	prediction := Prediction{
		Confidence: 0.70,
	}

	decision := Decide(prediction, DefaultConfidenceThreshold)

	if decision.Mode != DecisionPredictive {
		t.Fatalf("expected predictive mode at threshold, got %s", decision.Mode)
	}
}
