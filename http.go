package agentruntime

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// These handlers are mounted under the existing Go authority and BrowserSecurity
// middleware. A product may preserve Host/Origin through a same-machine proxy
// over real loopback after checking its resource authority. Forwarded headers
// establish no trust; remote ingress requires a separate authenticated design.
var routeBase = regexp.MustCompile(`^/(?:[A-Za-z0-9_-]+/)*[A-Za-z0-9_-]+$`)

const ClientHeader = "X-XGC-Agent-Client"

// RegisterRoutes mounts one broker at a product-selected same-origin API root.
func RegisterRoutes(mux *http.ServeMux, b *Broker, basePath string) error {
	basePath = strings.TrimSuffix(basePath, "/")
	if !routeBase.MatchString(basePath) {
		return errors.New("invalid agent runtime route base")
	}
	register := func(pattern string, h http.HandlerFunc) {
		mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			if !localRequest(r) {
				writeFailure(w, http.StatusForbidden, "local_only", "agent runtime requires same-origin loopback access")
				return
			}
			h(w, r)
		})
	}
	register("GET "+basePath+"/settings", func(w http.ResponseWriter, r *http.Request) {
		v, err := b.Settings()
		if err != nil {
			domainError(w, err)
			return
		}
		writeData(w, 200, v)
	})
	register("POST "+basePath+"/settings", func(w http.ResponseWriter, r *http.Request) {
		var c SettingsUpdate
		if !decodeBody(w, r, &c) {
			return
		}
		v, err := b.UpdateSettings(r.Context(), c)
		if err != nil {
			domainError(w, err)
			return
		}
		writeData(w, 200, v)
	})
	register("POST "+basePath+"/settings/refresh", func(w http.ResponseWriter, r *http.Request) {
		var c struct {
			ID string `json:"id"`
		}
		if !decodeBody(w, r, &c) {
			return
		}
		v, err := b.RefreshSettings(r.Context(), c.ID)
		if err != nil {
			domainError(w, err)
			return
		}
		writeData(w, 200, v)
	})
	register("GET "+basePath+"/providers", func(w http.ResponseWriter, r *http.Request) { writeData(w, 200, b.Providers()) })
	register("GET "+basePath+"/sessions", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		limit := 50
		var err error
		if query.Get("limit") != "" {
			limit, err = strconv.Atoi(query.Get("limit"))
		}
		if err != nil {
			domainError(w, errors.New("invalid conversation page limit"))
			return
		}
		page, err := b.ListPage(SessionListOptions{Limit: limit, After: query.Get("after"), Context: ContextRef{Kind: query.Get("contextKind"), ID: query.Get("contextId")}})
		if err != nil {
			domainError(w, err)
			return
		}
		writeData(w, 200, page)
	})
	register("GET "+basePath+"/attention", func(w http.ResponseWriter, r *http.Request) {
		snapshot := b.Attention()
		etag := `"` + snapshot.Revision + `"`
		w.Header().Set("ETag", etag)
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		writeData(w, 200, snapshot)
	})
	register("POST "+basePath+"/sessions", func(w http.ResponseWriter, r *http.Request) {
		var c Create
		if !decodeBody(w, r, &c) {
			return
		}
		s, err := b.Create(r.Context(), r.Header.Get("Idempotency-Key"), c)
		if err != nil {
			domainError(w, err)
			return
		}
		writeData(w, 202, s)
	})
	register("GET "+basePath+"/sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
		s, err := b.Get(r.PathValue("id"))
		if err != nil {
			domainError(w, err)
			return
		}
		writeData(w, 200, s)
	})
	register("POST "+basePath+"/sessions/{id}/metadata", func(w http.ResponseWriter, r *http.Request) {
		var update MetadataUpdate
		if !decodeBody(w, r, &update) {
			return
		}
		session, err := b.UpdateMetadata(r.PathValue("id"), update)
		if err != nil {
			domainError(w, err)
			return
		}
		writeData(w, 200, session)
	})
	register("GET "+basePath+"/sessions/{id}/inputs", func(w http.ResponseWriter, r *http.Request) {
		inputs, err := b.Inputs(r.PathValue("id"))
		if err != nil {
			domainError(w, err)
			return
		}
		writeData(w, 200, inputs)
	})
	register("POST "+basePath+"/sessions/{id}/evaluate-inputs", func(w http.ResponseWriter, r *http.Request) {
		if err := b.EvaluateInputs(r.PathValue("id")); err != nil {
			domainError(w, err)
			return
		}
		writeData(w, 202, map[string]bool{"requested": true})
	})
	register("POST "+basePath+"/sessions/{id}/queue", func(w http.ResponseWriter, r *http.Request) {
		var command QueueCommand
		if !decodeBody(w, r, &command) {
			return
		}
		queue, err := b.Queue(r.PathValue("id"), r.Header.Get("Idempotency-Key"), command)
		if err != nil {
			domainError(w, err)
			return
		}
		writeData(w, 202, queue)
	})
	register("POST "+basePath+"/sessions/{id}/prompts", func(w http.ResponseWriter, r *http.Request) {
		var c struct {
			Text    string       `json:"text"`
			Options AgentOptions `json:"options,omitempty"`
		}
		if !decodeBody(w, r, &c) {
			return
		}
		turn, err := b.PromptWithOptions(r.PathValue("id"), r.Header.Get("Idempotency-Key"), c.Text, c.Options)
		if err != nil {
			domainError(w, err)
			return
		}
		writeData(w, 202, map[string]string{"turnId": turn})
	})
	register("POST "+basePath+"/sessions/{id}/inputs/{requestId}", func(w http.ResponseWriter, r *http.Request) {
		var a Answer
		if !decodeBody(w, r, &a) {
			return
		}
		if err := b.AnswerContext(r.Context(), r.PathValue("id"), r.PathValue("requestId"), a); err != nil {
			domainError(w, err)
			return
		}
		writeData(w, 202, map[string]bool{"submitted": true})
	})
	register("POST "+basePath+"/sessions/{id}/cancel", func(w http.ResponseWriter, r *http.Request) {
		if err := b.Cancel(r.Context(), r.PathValue("id")); err != nil {
			domainError(w, err)
			return
		}
		writeData(w, 202, map[string]bool{"requested": true})
	})
	register("POST "+basePath+"/sessions/{id}/reconnect", func(w http.ResponseWriter, r *http.Request) {
		if err := b.ReconnectContext(r.Context(), r.PathValue("id")); err != nil {
			domainError(w, err)
			return
		}
		writeData(w, 202, map[string]bool{"requested": true})
	})
	register("POST "+basePath+"/sessions/{id}/close", func(w http.ResponseWriter, r *http.Request) {
		if err := b.CloseSession(r.PathValue("id")); err != nil {
			domainError(w, err)
			return
		}
		writeData(w, 200, map[string]bool{"closed": true})
	})
	register("GET "+basePath+"/sessions/{id}/events", func(w http.ResponseWriter, r *http.Request) {
		after, err := eventCursor(r)
		if err != nil {
			domainError(w, err)
			return
		}
		events, changed, err := b.Replay(r.PathValue("id"), after)
		if err != nil {
			domainError(w, err)
			return
		}
		f, ok := w.(http.Flusher)
		if !ok {
			writeFailure(w, 500, "stream_unavailable", "streaming writer unavailable")
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("X-Accel-Buffering", "no")
		_, _ = io.WriteString(w, "retry: 1000\n\n")
		f.Flush()
		// Short streams avoid the service's global write timeout. EventSource resumes
		// from Last-Event-ID; the original query cursor is only an initial fallback.
		end := time.NewTimer(20 * time.Second)
		defer end.Stop()
		heartbeat := time.NewTicker(5 * time.Second)
		defer heartbeat.Stop()
		for {
			for _, e := range events {
				data, _ := json.Marshal(e)
				if _, err = io.WriteString(w, "id: "+strconv.FormatUint(e.Seq, 10)+"\nevent: native-agent\ndata: "+string(data)+"\n\n"); err != nil {
					return
				}
				after = e.Seq
			}
			f.Flush()
			if len(events) == 256 {
				events, changed, err = b.Replay(r.PathValue("id"), after)
				if err != nil {
					return
				}
				continue
			}
			select {
			case <-r.Context().Done():
				return
			case <-end.C:
				return
			case <-heartbeat.C:
				if _, err = io.WriteString(w, ": heartbeat\n\n"); err != nil {
					return
				}
				f.Flush()
			case <-changed:
			}
			events, changed, err = b.Replay(r.PathValue("id"), after)
			if err != nil {
				return
			}
		}
	})
	return nil
}
func localRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || !net.ParseIP(host).IsLoopback() {
		return false
	}
	h := r.Host
	if parsed, _, e := net.SplitHostPort(h); e == nil {
		h = parsed
	}
	h = strings.Trim(h, "[]")
	if h != "localhost" {
		ip := net.ParseIP(h)
		if ip == nil || !ip.IsLoopback() {
			return false
		}
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		u, e := url.Parse(origin)
		if e != nil || u.Scheme != "http" && u.Scheme != "https" || u.Host != r.Host || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" {
			return false
		}
	}
	if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" && site != "none" {
		return false
	}
	// A custom header makes cross-origin simple POSTs fail before native actions.
	if r.Method == "POST" && r.Header.Get(ClientHeader) != "1" {
		return false
	}
	return true
}
func eventCursor(r *http.Request) (uint64, error) {
	value := r.Header.Get("Last-Event-ID")
	if value == "" {
		value = r.URL.Query().Get("after")
	}
	if value == "" {
		return 0, nil
	}
	n, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, errors.New("invalid event cursor")
	}
	return n, nil
}
func decodeBody(w http.ResponseWriter, r *http.Request, target any) bool {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeFailure(w, 415, "json_required", "JSON is required")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 256<<10)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if d.Decode(target) != nil || d.Decode(new(any)) != io.EOF {
		writeFailure(w, 400, "invalid_json", "invalid or oversized JSON body")
		return false
	}
	return true
}
func writeData(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
}
func writeFailure(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "message": message}})
}
func domainError(w http.ResponseWriter, err error) {
	status, code := 400, "native_request_invalid"
	switch {
	case errors.Is(err, ErrConflict), errors.Is(err, ErrCursor):
		status, code = 409, "native_conflict"
	case errors.Is(err, ErrNotFound):
		status, code = 404, "native_not_found"
	case errors.Is(err, ErrStale):
		status, code = 410, "native_request_expired"
	case errors.Is(err, ErrUnavailable):
		status, code = 503, "native_unavailable"
	}
	writeFailure(w, status, code, err.Error())
}
