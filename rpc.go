package agentruntime

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

type wire struct {
	JSONRPC string          `json:"jsonrpc,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  map[string]any  `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
type incomingRequest struct {
	cancel   context.CancelFunc
	threadID string
}
type rpcPeer struct {
	writeMu  sync.Mutex
	mu       sync.Mutex
	input    io.WriteCloser
	pending  map[string]chan wire
	incoming map[string]incomingRequest
	count    uint64
	done     chan struct{}
	once     sync.Once
	acp      bool
	notify   func(string, map[string]any)
	request  func(context.Context, string, map[string]any) (any, error)
	ctx      context.Context
	cancel   context.CancelFunc
}

func newPeer(input io.WriteCloser, output io.Reader, acp bool, notify func(string, map[string]any), request func(context.Context, string, map[string]any) (any, error)) *rpcPeer {
	ctx, cancel := context.WithCancel(context.Background())
	p := &rpcPeer{input: input, pending: map[string]chan wire{}, incoming: map[string]incomingRequest{}, done: make(chan struct{}), acp: acp, notify: notify, request: request, ctx: ctx, cancel: cancel}
	go p.read(output)
	return p
}
func (p *rpcPeer) send(w wire) error {
	if p.acp {
		w.JSONRPC = "2.0"
	}
	// Closing a blocked pipe breaks a stalled native stdin write, including queued writers.
	timer := time.AfterFunc(10*time.Second, func() { _ = p.Close() })
	defer timer.Stop()
	p.writeMu.Lock()
	defer p.writeMu.Unlock()
	select {
	case <-p.done:
		return io.EOF
	default:
	}
	if w.Method != "" && w.Params != nil && len(w.Params) == 0 {
		// Preserve an explicitly empty parameter object for native ACP extensions.
		return json.NewEncoder(p.input).Encode(struct {
			wire
			Params map[string]any `json:"params"`
		}{w, w.Params})
	}
	return json.NewEncoder(p.input).Encode(w)
}
func (p *rpcPeer) Call(ctx context.Context, method string, params map[string]any) (map[string]any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p.mu.Lock()
	p.count++
	id := fmt.Sprintf(`"client-%d"`, p.count)
	if p.acp {
		id = fmt.Sprintf("%d", p.count)
	}
	ch := make(chan wire, 1)
	p.pending[id] = ch
	p.mu.Unlock()
	defer func() { p.mu.Lock(); delete(p.pending, id); p.mu.Unlock() }()
	if err := p.send(wire{ID: json.RawMessage(id), Method: method, Params: params}); err != nil {
		return nil, err
	}
	select {
	case w := <-ch:
		if w.Error != nil {
			return nil, protocolError(w.Error.Code)
		}
		var result map[string]any
		if len(w.Result) == 0 || json.Unmarshal(w.Result, &result) != nil || result == nil {
			return nil, errors.New("invalid native RPC response")
		}
		return result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-p.done:
		return nil, io.EOF
	}
}
func (p *rpcPeer) Notify(method string, params map[string]any) error {
	return p.send(wire{Method: method, Params: params})
}
func (p *rpcPeer) read(output io.Reader) {
	defer p.finish()
	scanner := bufio.NewScanner(output)
	scanner.Buffer(make([]byte, 4096), MaxFrame)
	permits := make(chan struct{}, 16)
	for scanner.Scan() {
		var w wire
		if json.Unmarshal(scanner.Bytes(), &w) != nil || (p.acp && w.JSONRPC != "2.0") {
			return
		}
		if len(w.ID) > 256 {
			return
		}
		if w.Method != "" {
			if len(w.Method) > 256 {
				return
			}
			if len(w.ID) > 0 {
				if string(w.ID) == "null" {
					return
				}
				// Requests have independent IDs. Never block the stream reader on user input.
				select {
				case permits <- struct{}{}:
				default:
					_ = p.send(wire{ID: w.ID, Error: &rpcError{-32000, "too many pending native requests"}})
					continue
				}
				requestCtx, requestCancel := context.WithCancel(p.ctx)
				key := string(w.ID)
				p.mu.Lock()
				_, duplicate := p.incoming[key]
				if !duplicate {
					p.incoming[key] = incomingRequest{cancel: requestCancel, threadID: text(w.Params, "threadId")}
				}
				p.mu.Unlock()
				if duplicate {
					requestCancel()
					<-permits
					return
				}
				go func(w wire) {
					defer func() { requestCancel(); p.mu.Lock(); delete(p.incoming, key); p.mu.Unlock(); <-permits }()
					result, err := p.request(requestCtx, w.Method, w.Params)
					// Upstream-resolved/closed requests must not receive a stale answer.
					if requestCtx.Err() != nil {
						return
					}
					response := wire{ID: w.ID}
					if err != nil {
						response.Error = &rpcError{-32601, "unsupported or unavailable client request"}
					} else {
						response.Result, _ = json.Marshal(result)
					}
					_ = p.send(response)
				}(w)
			} else {
				if !p.acp && w.Method == "serverRequest/resolved" {
					id, _ := json.Marshal(w.Params["requestId"])
					p.mu.Lock()
					pending, exists := p.incoming[string(id)]
					p.mu.Unlock()
					if exists && pending.threadID != "" && pending.threadID == text(w.Params, "threadId") {
						pending.cancel()
					}
				}
				p.notify(w.Method, w.Params)
			}
			continue
		}
		if len(w.ID) == 0 {
			return
		}
		p.mu.Lock()
		ch := p.pending[string(w.ID)]
		p.mu.Unlock()
		if ch != nil {
			select {
			case ch <- w:
			default:
			}
		}
	}
}
func (p *rpcPeer) finish()      { p.once.Do(func() { p.cancel(); close(p.done) }) }
func (p *rpcPeer) Close() error { p.finish(); return p.input.Close() }

type child struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	done   chan struct{}
	mu     sync.Mutex
	err    error
	once   sync.Once
}

func startChild(p Profile, args []string, cwd string, delegatedEnvironment ...string) (*child, error) {
	if err := checkExecutable(p); err != nil {
		return nil, err
	}
	c := exec.Command(p.Executable, args...)
	c.Dir = cwd
	c.Env = append(AgentEnvironment(os.Environ()), delegatedEnvironment...)
	c.Stderr = io.Discard
	isolateProcess(c)
	input, err := c.StdinPipe()
	if err != nil {
		return nil, err
	}
	output, err := c.StdoutPipe()
	if err != nil {
		input.Close()
		return nil, err
	}
	if err = c.Start(); err != nil {
		input.Close()
		output.Close()
		return nil, errors.New("native executable could not start")
	}
	return &child{cmd: c, stdin: input, stdout: output, done: make(chan struct{})}, nil
}

// Wait must be called after stdout has been drained; exec.Wait can otherwise
// close the stdout pipe before the reader consumes the terminal protocol frame.
func (c *child) Wait() error {
	c.once.Do(func() { err := c.cmd.Wait(); c.mu.Lock(); c.err = err; c.mu.Unlock(); close(c.done) })
	<-c.done
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}
func (c *child) Stop() {
	select {
	case <-c.done:
		return
	default:
	}
	_ = terminateProcess(c.cmd, false)
	select {
	case <-c.done:
	case <-time.After(2 * time.Second):
		_ = terminateProcess(c.cmd, true)
	}
}
