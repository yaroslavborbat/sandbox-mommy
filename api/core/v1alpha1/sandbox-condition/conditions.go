package sandboxcondition

// Type represents the various condition types for the `Sandbox`.
type Type string

func (s Type) String() string {
	return string(s)
}

const (
	TypeReady Type = "Ready"
)

type Reason string

func (s Reason) String() string {
	return string(s)
}

const (
	ReasonReady       Reason = "Ready"
	ReasonPending     Reason = "Pending"
	ReasonFailed      Reason = "Failed"
	ReasonTerminating Reason = "Terminating"
)
