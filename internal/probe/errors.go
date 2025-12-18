package probe

import "errors"

// Probe-related errors.
var (
	// ErrTimeout indicates the probe timed out waiting for a response
	ErrTimeout = errors.New("probe timeout")

	// ErrPermissionDenied indicates insufficient privileges for raw sockets
	ErrPermissionDenied = errors.New("permission denied: raw socket requires elevated privileges")

	// ErrHostUnreachable indicates the destination host is unreachable
	ErrHostUnreachable = errors.New("destination host unreachable")

	// ErrNetworkUnreachable indicates the network is unreachable
	ErrNetworkUnreachable = errors.New("network unreachable")

	// ErrInvalidPacket indicates a malformed or unexpected packet was received
	ErrInvalidPacket = errors.New("invalid packet received")

	// ErrSocketClosed indicates the socket has been closed
	ErrSocketClosed = errors.New("socket closed")

	// ErrInvalidTTL indicates the TTL value is out of range
	ErrInvalidTTL = errors.New("TTL must be between 1 and 255")

	// ErrNoResponse indicates no response was received (different from timeout)
	ErrNoResponse = errors.New("no response received")
)

// IsTimeout returns true if the error indicates a timeout.
func IsTimeout(err error) bool {
	return errors.Is(err, ErrTimeout)
}

// IsPermissionError returns true if the error is a permission error.
func IsPermissionError(err error) bool {
	return errors.Is(err, ErrPermissionDenied)
}
