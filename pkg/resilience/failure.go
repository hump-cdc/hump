package resilience

// FailureMode selects what happens once every retry attempt is exhausted.
type FailureMode string

const (
	FailureModeCrash FailureMode = "crash"
	FailureModeSkip  FailureMode = "skip"
)
