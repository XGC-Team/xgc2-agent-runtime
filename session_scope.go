package nativeagent

import "context"

// SessionScopedDriver is an optional trusted-host composition hook. Broker
// invokes it after Prepare validates the retained scope/workspace and before
// Open. The values come from the durable native session, not model messages.
// On any binding/open failure Broker closes the driver. A fresh driver is bound
// again during explicit resume; a host can therefore rotate delegated access.
// Implementations must reject rebinding and release all grants in Close.
type SessionScopedDriver interface {
	BindSession(context.Context, Create, string) error
}
