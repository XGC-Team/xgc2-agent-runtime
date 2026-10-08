package agentruntime

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/XGC-Team/xgc2-storage/api"
	xrpc "github.com/XGC-Team/xgc2-xrpc/go"
)

type interceptStorage struct {
	StorageClient
	batch   func(context.Context, api.BatchRequest) (api.Receipt, error)
	receipt func(context.Context, string, api.ReceiptRequest) (api.Receipt, error)
}

func (s interceptStorage) Batch(ctx context.Context, r api.BatchRequest) (api.Receipt, error) {
	if s.batch != nil {
		return s.batch(ctx, r)
	}
	return s.StorageClient.Batch(ctx, r)
}
func (s interceptStorage) Receipt(ctx context.Context, id string, r api.ReceiptRequest) (api.Receipt, error) {
	if s.receipt != nil {
		return s.receipt(ctx, id, r)
	}
	return s.StorageClient.Receipt(ctx, id, r)
}

func TestStorageLostReplyUsesReceiptWithoutReplayingMutation(t *testing.T) {
	s := testSession(t, "s_lost")
	base := s.store.Client
	batches, lookups := 0, 0
	s.store.Client = interceptStorage{StorageClient: base, batch: func(ctx context.Context, r api.BatchRequest) (api.Receipt, error) {
		batches++
		out, err := base.Batch(ctx, r)
		if err != nil {
			return out, err
		}
		return api.Receipt{}, xrpc.Failure("unavailable", xrpc.OutcomeUnknown, errors.New("reply lost after real COMMIT"))
	}, receipt: func(ctx context.Context, id string, r api.ReceiptRequest) (api.Receipt, error) {
		lookups++
		return base.Receipt(ctx, id, r)
	}}
	if err := s.appendLocked(Event{Kind: "notice", Text: "one committed event"}); err != nil {
		t.Fatal(err)
	}
	if batches != 1 || lookups != 1 || s.info.LastSeq != 1 {
		t.Fatalf("batches=%d lookups=%d seq=%d", batches, lookups, s.info.LastSeq)
	}
	events, err := s.store.readEvents(s.info.ID, 0, 1)
	if err != nil || len(events) != 1 || events[0].Text != "one committed event" {
		t.Fatalf("events=%+v err=%v", events, err)
	}
}

func TestStorageUnresolvedReceiptStopsSessionWithoutAutomaticReplay(t *testing.T) {
	s := testSession(t, "s_unknown")
	base := s.store.Client
	batches := 0
	s.store.Client = interceptStorage{StorageClient: base, batch: func(ctx context.Context, r api.BatchRequest) (api.Receipt, error) {
		batches++
		if _, err := base.Batch(ctx, r); err != nil {
			return api.Receipt{}, err
		}
		return api.Receipt{}, xrpc.Failure("unavailable", xrpc.OutcomeUnknown, errors.New("lost reply"))
	}, receipt: func(context.Context, string, api.ReceiptRequest) (api.Receipt, error) {
		return api.Receipt{}, &api.Error{Code: "not_found", Message: "not visible yet"}
	}}
	err := s.appendLocked(Event{Kind: "notice", Text: "uncertain"})
	if err == nil || !strings.Contains(err.Error(), "ar-write-") || s.info.LastSeq != 0 {
		t.Fatalf("err=%v seq=%d", err, s.info.LastSeq)
	}
	if err = s.appendLocked(Event{Kind: "notice", Text: "must not replay"}); err == nil || batches != 1 {
		t.Fatalf("batches=%d err=%v", batches, err)
	}
	s.store.Client = base
	rows, err := s.store.records(context.Background())
	if err != nil || len(rows) != 1 || rows[0].LastSeq != 1 {
		t.Fatalf("persisted uncertain commit lost: %+v %v", rows, err)
	}
}

func TestStorageIndependentHostsOnlyOneAtomicAppendWins(t *testing.T) {
	a := testSession(t, "s_race")
	other, err := NewStore(a.store.StorageBinding)
	if err != nil {
		t.Fatal(err)
	}
	b := newSession(a.info, other)
	b.profile = a.profile
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, session := range []*liveSession{a, b} {
		wg.Add(1)
		go func(s *liveSession) {
			defer wg.Done()
			<-start
			results <- s.appendLocked(Event{Kind: "notice", Text: "contender"})
		}(session)
	}
	close(start)
	wg.Wait()
	close(results)
	winners, conflicts := 0, 0
	for err := range results {
		if err == nil {
			winners++
		} else if errors.Is(err, ErrConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if winners != 1 || conflicts != 1 {
		t.Fatalf("winners=%d conflicts=%d", winners, conflicts)
	}
	rows, err := a.store.records(context.Background())
	if err != nil || len(rows) != 1 || rows[0].LastSeq != 1 {
		t.Fatalf("state diverged: %+v %v", rows, err)
	}
	if events, err := a.store.readEvents(a.info.ID, 0, 1); err != nil || len(events) != 1 {
		t.Fatalf("events=%+v err=%v", events, err)
	}
}

func TestStorageIdentityAndIncompleteReceiptFailClosed(t *testing.T) {
	s := testSession(t, "s_receipt")
	base := s.store.Client
	s.store.Client = interceptStorage{StorageClient: base, batch: func(ctx context.Context, r api.BatchRequest) (api.Receipt, error) {
		out, err := base.Batch(ctx, r)
		out.Versions = out.Versions[:1]
		return out, err
	}}
	if err := s.appendLocked(Event{Kind: "notice", Text: "bad receipt"}); err == nil || s.info.LastSeq != 0 {
		t.Fatalf("invalid receipt accepted: %v", err)
	}
	wrong, err := NewStore(StorageBinding{Client: base, Scope: s.store.Scope, DatabaseID: "wrong", Schema: StorageSchema})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = NewBroker(wrong, nil, func(context.Context, Create, string, bool) (string, error) {
		t.Fatal("identity mismatch launched workspace")
		return "", nil
	}, nil); err == nil {
		t.Fatal("wrong database accepted")
	}
}

func TestStorageQuotaFailureDoesNotPublishStateOrPrompt(t *testing.T) {
	s := testSession(t, "s_quota")
	s.bytes = MaxSessionBytes
	if err := s.appendLocked(Event{Kind: "item.snapshot", Role: "user", Status: "submitted", Text: "not submitted", TurnID: "t_quota"}); err == nil {
		t.Fatal("quota bypassed")
	}
	if s.info.LastSeq != 0 {
		t.Fatal("failed write published sequence")
	}
	if _, found, err := s.store.prompt(s.info.ID, "t_quota"); err != nil || found {
		t.Fatalf("failed write published prompt: %v %v", found, err)
	}
}
