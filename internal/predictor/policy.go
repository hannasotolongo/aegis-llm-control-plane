package predictor

const DefaultConfidenceThreshold = 0.70

type DecisionMode string

const (
	DecisionPredictive    DecisionMode = "PREDICTIVE"
	DecisionDeterministic DecisionMode = "DETERMINISTIC"
)

type Decision struct {
	Mode                DecisionMode
	TrustPrediction     bool
	PredictedContention bool
	Confidence          float64
}

func Decide(prediction Prediction, threshold float64) Decision {
	trust := prediction.Confidence >= threshold

	mode := DecisionDeterministic
	if trust {
		mode = DecisionPredictive
	}

	return Decision{
		Mode:                mode,
		TrustPrediction:     trust,
		PredictedContention: prediction.PredictedContention,
		Confidence:          prediction.Confidence,
	}
}
