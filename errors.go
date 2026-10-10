package agentruntime

import (
	"context"
	"errors"
	"fmt"

	"github.com/XGC-Team/xgc2-storage/api"
	xrpc "github.com/XGC-Team/xgc2-xrpc/go"
)

// The errors a Service returns. Callers classify them with errors.Is; the XRPC
// service maps each class onto the XRPC error vocabulary (see errorCode).
var (
	// ErrInvalid wraps a rejected argument: the request was malformed or
	// violated a documented bound. Nothing changed.
	ErrInvalid = errors.New("invalid request")
	// ErrNotFound names a missing session, queued message or request owner.
	ErrNotFound = errors.New("session not found")
	// ErrConflict is a lost compare-and-swap or a reused identity with different content.
	ErrConflict = errors.New("request identity conflict")
	// ErrStale means the request, turn or runtime the caller addressed is no
	// longer current, for example an answer to an approval that already resolved.
	ErrStale = errors.New("request is no longer pending")
	// ErrCursor means an event cursor is ahead of the session's journal. Event
	// streams answer it with a reset instead of an error.
	ErrCursor = errors.New("event cursor is ahead of this session")
	// ErrCapacity wraps a reached quota: workers, retained conversations, queue
	// length, event storage.
	ErrCapacity = errors.New("capacity reached")
	// ErrUnavailable means the runtime cannot take the operation now: it is
	// closed, the session is not ready, or the provider is not configured.
	ErrUnavailable = errors.New("agent runtime unavailable")
)

type classified struct {
	class error
	msg   string
}

func (e classified) Error() string        { return e.msg }
func (e classified) Is(target error) bool { return target == e.class }

func invalid(format string, args ...any) error {
	return classified{ErrInvalid, fmt.Sprintf(format, args...)}
}
func exhausted(format string, args ...any) error {
	return classified{ErrCapacity, fmt.Sprintf(format, args...)}
}

// errorCode classifies err as one name of the XRPC error vocabulary. Errors of
// the storage client keep the code the storage service gave them.
func errorCode(err error) string {
	var call *xrpc.CallError
	var domain *api.Error
	switch {
	case errors.Is(err, ErrInvalid):
		return "invalid_argument"
	case errors.Is(err, ErrNotFound):
		return "not_found"
	case errors.Is(err, ErrConflict), errors.Is(err, ErrStale), errors.Is(err, ErrCursor):
		return "conflict"
	case errors.Is(err, ErrCapacity):
		return "resource_exhausted"
	case errors.Is(err, ErrUnavailable):
		return "unavailable"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline_exceeded"
	case errors.Is(err, context.Canceled):
		return "cancelled"
	case errors.As(err, &call) && vocabulary[call.Code]:
		return call.Code
	case errors.As(err, &domain) && vocabulary[domain.Code]:
		return domain.Code
	}
	return "internal"
}

var vocabulary = map[string]bool{
	"invalid_argument": true, "not_found": true, "conflict": true, "resource_exhausted": true, "deadline_exceeded": true,
	"cancelled": true, "unavailable": true, "internal": true, "unauthenticated": true, "permission_denied": true,
}
