package nativeagent

import "context"

// Session binding may outlive the create/resume HTTP response, but never the
// broker. Copy only context values, not the request lifetime; private scoped
// enrollment can reach prepare/BindSession without entering Scope or journals.
// No binding values are retained by the broker after this connection attempt.
type sessionBindingContext struct {
	context.Context
	values context.Context
}

func (c sessionBindingContext) Value(key any) any {
	if c.values != nil {
		if value := c.values.Value(key); value != nil {
			return value
		}
	}
	return c.Context.Value(key)
}
