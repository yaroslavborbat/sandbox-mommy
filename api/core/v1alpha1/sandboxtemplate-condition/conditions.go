package sandboxtemplatecondition

// Type represents the various condition types for the `SandboxTemplate`.
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
	ReasonTerminating Reason = "Terminating"
)
