package trace

import "errors"

// Trace-related errors.
var (
	// ErrInvalidMaxHops indicates max hops is out of valid range (1-255)
	ErrInvalidMaxHops = errors.New("max hops must be between 1 and 255")

	// ErrInvalidProbeCount indicates probe count is out of valid range
	ErrInvalidProbeCount = errors.New("probe count must be between 1 and 10")

	// ErrInvalidTimeout indicates timeout is too short
	ErrInvalidTimeout = errors.New("timeout must be at least 100ms")

	// ErrInvalidFirstHop indicates first hop is invalid
	ErrInvalidFirstHop = errors.New("first hop must be between 1 and max hops")

	// ErrTargetResolution indicates the target could not be resolved
	ErrTargetResolution = errors.New("could not resolve target hostname")

	// ErrNoRoute indicates no route to the destination
	ErrNoRoute = errors.New("no route to destination")

	// ErrTraceIncomplete indicates the trace did not reach the destination
	ErrTraceIncomplete = errors.New("trace did not reach destination")
)
