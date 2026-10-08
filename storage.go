package agentruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/XGC-Team/xgc2-storage/api"
	xrpc "github.com/XGC-Team/xgc2-xrpc/go"
)

const StorageNamespace = "agent-runtime"
const StorageSchema = "agent-runtime.v1"
const MaxSessionBytes = 32 << 20
const MaxRetainedSessions = 2048
const storageCallTimeout = 10 * time.Second

// StorageClient is implemented by the storage product's SDK client. The host
// owns its transport and its authenticated, instance-bound service reference.
type StorageClient interface {
	Snapshot(context.Context, string, api.SnapshotRequest) (api.SnapshotResponse, error)
	Batch(context.Context, api.BatchRequest) (api.Receipt, error)
	Receipt(context.Context, string, api.ReceiptRequest) (api.Receipt, error)
}

type StorageBinding struct {
	Client     StorageClient
	Scope      api.Scope
	DatabaseID string
	Schema     string
}

// Store has no local persistence, implicit initialization or mutation retries.
// Settings and conversations may use different granted scopes of this schema.
type Store struct {
	StorageBinding
	writer chan struct{}
}

func NewStore(binding StorageBinding) (*Store, error) {
	if binding.Client == nil || binding.Scope.Namespace != StorageNamespace || binding.Scope.User == "" || binding.Scope.Workspace == "" || binding.DatabaseID == "" || binding.Schema != StorageSchema {
		return nil, errors.New("agent-runtime: explicit storage client, database, schema and owner scope required")
	}
	return &Store{StorageBinding: binding, writer: make(chan struct{}, 1)}, nil
}

func (s *Store) begin(ctx context.Context) (context.Context, func(), error) {
	ctx, cancel := context.WithTimeout(ctx, storageCallTimeout)
	select {
	case s.writer <- struct{}{}:
		return ctx, func() { <-s.writer; cancel() }, nil
	case <-ctx.Done():
		cancel()
		return nil, nil, ctx.Err()
	}
}

func revision(value string) bool {
	n, err := strconv.ParseUint(value, 10, 63)
	return err == nil && strconv.FormatUint(n, 10) == value
}

func nextRevision(before, after string) bool {
	prior, e := strconv.ParseUint(before, 10, 63)
	if e != nil || prior == 1<<63-1 {
		return false
	}
	return strconv.FormatUint(prior+1, 10) == after
}

func (s *Store) snapshot(ctx context.Context, queries []api.Query, at *api.Token) (api.SnapshotResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, storageCallTimeout)
	defer cancel()
	out, err := s.Client.Snapshot(ctx, "ar-read-"+randomID(), api.SnapshotRequest{Scope: s.Scope, Queries: queries, At: at})
	if err != nil {
		return out, storageError(err)
	}
	if out.Scope != s.Scope || out.Token.DatabaseID != s.DatabaseID || out.Token.Schema != s.Schema || !revision(out.Token.Revision) || len(out.Results) != len(queries) || (at != nil && out.Token != *at) {
		return api.SnapshotResponse{}, errors.New("agent-runtime: storage snapshot identity mismatch")
	}
	for i, result := range out.Results {
		if result.Collection != queries[i].Collection {
			return api.SnapshotResponse{}, errors.New("agent-runtime: storage collection mismatch")
		}
		if len(queries[i].Keys) > 0 && len(result.Records) != len(queries[i].Keys) {
			return api.SnapshotResponse{}, errors.New("agent-runtime: incomplete exact-key snapshot")
		}
		for j, record := range result.Records {
			if !revision(record.Version) || (len(queries[i].Keys) > 0 && record.Key != queries[i].Keys[j]) || (record.Missing && (record.Version != "0" || record.Deleted || len(record.Data) != 0)) || (!record.Missing && record.Version == "0") {
				return api.SnapshotResponse{}, errors.New("agent-runtime: invalid storage record identity")
			}
		}
	}
	return out, nil
}

func storageError(err error) error {
	var domain *api.Error
	var call *xrpc.CallError
	if (errors.As(err, &domain) && domain.Code == "conflict") || (errors.As(err, &call) && call.Code == "conflict") {
		return errors.Join(ErrConflict, err)
	}
	return err
}

func (s *Store) commit(ctx context.Context, token api.Token, mutations []api.Mutation) error {
	ctx, cancel := context.WithTimeout(ctx, storageCallTimeout)
	defer cancel()
	id := "ar-write-" + randomID()
	receipt, err := s.Client.Batch(ctx, api.BatchRequest{Scope: s.Scope, Expected: token, RequestID: id, Mutations: mutations})
	if err != nil {
		// A lost reply is reconciled by the same durable identity. An absent receipt
		// never authorizes resubmission: the original call could still commit.
		var call *xrpc.CallError
		if errors.As(err, &call) && call.Disposition == xrpc.OutcomeUnknown {
			recovery, stop := context.WithTimeout(ctx, storageCallTimeout)
			defer stop()
			var lookup error
			receipt, lookup = s.Client.Receipt(recovery, "ar-receipt-"+randomID(), api.ReceiptRequest{Scope: s.Scope, RequestID: id})
			if lookup != nil {
				return fmt.Errorf("agent-runtime: unresolved storage request %s: %w", id, err)
			}
		} else {
			return storageError(err)
		}
	}
	if receipt.RequestID != id || receipt.Token.DatabaseID != s.DatabaseID || receipt.Token.Schema != s.Schema || !revision(receipt.Token.Revision) || !nextRevision(token.Revision, receipt.Token.Revision) || receipt.Durability != "sqlite-full" || !digest.MatchString(receipt.Digest) || len(receipt.Versions) != len(mutations) {
		return errors.New("agent-runtime: invalid durable storage receipt")
	}
	for i, record := range receipt.Versions {
		if record.Collection != mutations[i].Collection || record.Key != mutations[i].Key || record.Version != receipt.Token.Revision || record.Deleted != mutations[i].Delete || record.Missing {
			return errors.New("agent-runtime: incomplete durable storage receipt")
		}
	}
	return nil
}

type sessionRecord struct {
	Session
	ProfileSnapshot     Profile  `json:"profileSnapshot"`
	Bytes               int64    `json:"bytes"`
	QueueSeq            uint64   `json:"queueSeq,omitempty"`
	SubmittedQueueTurns []string `json:"submittedQueueTurns,omitempty"`
	LastTurnStatus      string   `json:"lastTurnStatus,omitempty"`
}

func eventKey(id string, seq uint64) string { return fmt.Sprintf("%s.%012d", id, seq) }

func (s *Store) create(ctx context.Context, record sessionRecord) error {
	ctx, release, err := s.begin(ctx)
	if err != nil {
		return err
	}
	defer release()
	read, err := s.snapshot(ctx, []api.Query{{Collection: "sessions", Keys: []string{record.ID}}}, nil)
	if err != nil {
		return err
	}
	prior := read.Results[0].Records[0]
	if !prior.Missing {
		return ErrConflict
	}
	raw, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return s.commit(ctx, read.Token, []api.Mutation{{Collection: "sessions", Key: record.ID, ExpectedVersion: prior.Version, Data: raw}})
}

func (s *Store) append(record sessionRecord, previous uint64, event Event) error {
	ctx, release, err := s.begin(context.Background())
	if err != nil {
		return err
	}
	defer release()
	read, err := s.snapshot(ctx, []api.Query{{Collection: "sessions", Keys: []string{record.ID}}}, nil)
	if err != nil {
		return err
	}
	prior := read.Results[0].Records[0]
	var before sessionRecord
	if prior.Missing || prior.Deleted || json.Unmarshal(prior.Data, &before) != nil || before.ID != record.ID || before.LastSeq != previous || before.RuntimeID != record.RuntimeID && event.Kind != "session.runtime" {
		return ErrConflict
	}
	meta, err := json.Marshal(record)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(event)
	if err != nil {
		return err
	}
	mutations := []api.Mutation{{Collection: "sessions", Key: record.ID, ExpectedVersion: prior.Version, Data: meta}, {Collection: "events", Key: eventKey(record.ID, event.Seq), ExpectedVersion: "0", Data: raw}}
	if event.Kind == "item.snapshot" && event.Role == "user" && event.Status == "submitted" {
		prompt, _ := json.Marshal(struct {
			SessionID   string `json:"sessionId"`
			Fingerprint string `json:"fingerprint"`
		}{record.ID, promptFingerprint(event.Text, optionsFromDetails(event.Details))})
		mutations = append(mutations, api.Mutation{Collection: "prompts", Key: event.TurnID, ExpectedVersion: "0", Data: prompt})
	}
	return s.commit(ctx, read.Token, mutations)
}

func (s *Store) records(ctx context.Context) ([]sessionRecord, error) {
	result := []sessionRecord{}
	var at *api.Token
	after := ""
	for {
		read, err := s.snapshot(ctx, []api.Query{{Collection: "sessions", After: after, Limit: 64}}, at)
		if err != nil {
			return nil, err
		}
		if at == nil {
			token := read.Token
			at = &token
		}
		for _, row := range read.Results[0].Records {
			var record sessionRecord
			if json.Unmarshal(row.Data, &record) != nil || row.Key != record.ID || record.SchemaVersion != Schema || !safeID.MatchString(record.ID) || record.Scope.Validate() != nil || record.LastSeq > MaxEvents || record.Bytes < 0 || record.Bytes > MaxSessionBytes || record.QueueSeq > record.LastSeq || len(record.SubmittedQueueTurns) > maxQueuedPrompts || record.MetadataRevision == 0 || record.ProfileSnapshot.ID != record.Scope.ProfileID || record.ProfileSnapshot.Provider != record.Provider {
				return nil, errors.New("agent-runtime: invalid persisted conversation")
			}
			if _, err := commandArgs(record.Provider); err != nil {
				return nil, err
			}
			result = append(result, record)
			if len(result) > MaxRetainedSessions {
				return nil, errors.New("agent-runtime: retained conversation quota exceeded")
			}
		}
		after = read.Results[0].NextAfter
		if after == "" {
			return result, nil
		}
	}
}

func (s *Store) readEvents(id string, after, end uint64) ([]Event, error) {
	if end < after || end-after > MaxEvents {
		return nil, ErrCursor
	}
	result := []Event{}
	for seq := after + 1; seq <= end; {
		last := min(end, seq+31)
		keys := make([]string, 0, last-seq+1)
		for n := seq; n <= last; n++ {
			keys = append(keys, eventKey(id, n))
		}
		events, err := s.eventPage(id, keys, seq)
		if err != nil {
			return nil, err
		}
		result = append(result, events...)
		seq = last + 1
	}
	return result, nil
}

func (s *Store) eventPage(id string, keys []string, first uint64) ([]Event, error) {
	read, err := s.snapshot(context.Background(), []api.Query{{Collection: "events", Keys: keys}}, nil)
	var domain *api.Error
	if err != nil && errors.As(err, &domain) && domain.Code == "resource_exhausted" && len(keys) > 1 {
		// Reads can be repartitioned to fit the service's byte budget. No mutation
		// is retried and no long-lived remote snapshot or transaction is held.
		n := len(keys) / 2
		left, e := s.eventPage(id, keys[:n], first)
		if e != nil {
			return nil, e
		}
		right, e := s.eventPage(id, keys[n:], first+uint64(n))
		if e != nil {
			return nil, e
		}
		return append(left, right...), nil
	}
	if err != nil {
		return nil, err
	}
	result := make([]Event, 0, len(keys))
	for i, row := range read.Results[0].Records {
		var event Event
		if row.Missing || row.Deleted || json.Unmarshal(row.Data, &event) != nil || event.SchemaVersion != Schema || event.SessionID != id || event.Seq != first+uint64(i) {
			return nil, errors.New("agent-runtime: persisted event range unavailable")
		}
		result = append(result, event)
	}
	return result, nil
}

func (s *Store) prompt(id, turn string) (string, bool, error) {
	read, err := s.snapshot(context.Background(), []api.Query{{Collection: "prompts", Keys: []string{turn}}}, nil)
	if err != nil {
		return "", false, err
	}
	row := read.Results[0].Records[0]
	if row.Missing {
		return "", false, nil
	}
	var value struct {
		SessionID   string `json:"sessionId"`
		Fingerprint string `json:"fingerprint"`
	}
	if row.Deleted || json.Unmarshal(row.Data, &value) != nil || value.SessionID != id || !digest.MatchString(value.Fingerprint) {
		return "", false, errors.New("agent-runtime: invalid persisted prompt identity")
	}
	return value.Fingerprint, true, nil
}

func (s *Store) profiles(ctx context.Context) ([]Profile, string, error) {
	read, err := s.snapshot(ctx, []api.Query{{Collection: "settings", Keys: []string{"providers"}}}, nil)
	if err != nil {
		return nil, "", err
	}
	row := read.Results[0].Records[0]
	if row.Missing {
		return []Profile{}, row.Version, nil
	}
	if row.Deleted {
		return nil, "", errors.New("agent-runtime: provider configuration was deleted")
	}
	profiles, err := decodeConfigBytes(row.Data)
	return profiles, row.Version, err
}

func (s *Store) saveProfiles(ctx context.Context, expected string, profiles []Profile) error {
	ctx, release, err := s.begin(ctx)
	if err != nil {
		return err
	}
	defer release()
	read, err := s.snapshot(ctx, []api.Query{{Collection: "settings", Keys: []string{"providers"}}}, nil)
	if err != nil {
		return err
	}
	row := read.Results[0].Records[0]
	if row.Version != expected || row.Deleted {
		return ErrConflict
	}
	raw, err := json.Marshal(Config{SchemaVersion: Schema, Profiles: profiles})
	if err != nil {
		return err
	}
	if len(raw) > 64<<10 {
		return errors.New("agent-runtime: provider configuration exceeds capacity")
	}
	return s.commit(ctx, read.Token, []api.Mutation{{Collection: "settings", Key: "providers", ExpectedVersion: expected, Data: raw}})
}
