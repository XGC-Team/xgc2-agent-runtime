package agentruntime

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// evaluation is what a host evaluator returns when its policy matched a rule.
func evaluation(mode DecisionMode) PolicyEvaluation {
	return PolicyEvaluation{Mode: mode, PolicyID: "operator-policy", PolicyRevision: "policy-revision-4", RuleID: "one-rule", Actor: DecisionActor{ID: "station-owner"}, Reason: "matched_rule"}
}

type policyDriver struct {
	*conversationDriver
	request Request
	result  chan Answer
}

func (d *policyDriver) Prompt(ctx context.Context, turn, _ string) error {
	answer, err := d.ask(ctx, d.request)
	if err != nil {
		return err
	}
	d.result <- answer
	return d.sink(Event{Kind: "turn.end", TurnID: turn, Status: "completed"})
}

func policyRequestFixture() Request {
	return Request{Kind: "permission", Title: "Provider operation", SourceMethod: "item/commandExecution/requestApproval", Details: &RequestDetails{Command: "printf fixture", Cwd: "/reviewed/project"}, Options: []Option{{ID: "yes", Kind: "allow_once"}, {ID: "no", Kind: "reject_once"}}}
}

func policyBroker(t *testing.T, request Request) (*Broker, Session, <-chan Answer) {
	b, _, _ := conversationBroker(t, false)
	result := make(chan Answer, 1)
	b.factory = func(_ Profile, sink Sink, ask Ask) (Driver, error) {
		return &policyDriver{conversationDriver: &conversationDriver{sink: sink, ask: ask}, request: request, result: result}, nil
	}
	scope := scope("fixture")
	scope.Context = ContextRef{Kind: "experiment", ID: "exp-a"}
	s, err := b.Create(context.Background(), "policy", scope)
	if err != nil {
		t.Fatal(err)
	}
	waitState(t, b, s.ID, "ready")
	s, _ = b.Get(bg, s.ID)
	return b, s, result
}

func TestBrokerPolicyUsesDurableRequestsAndNormalDecisionAudit(t *testing.T) {
	for _, mode := range []DecisionMode{DecisionAuto, DecisionDeny} {
		t.Run(string(mode), func(t *testing.T) {
			b, s, answers := policyBroker(t, policyRequestFixture())
			var reserved atomic.Int32
			if err := b.SetDecisionEvaluator(func(ctx context.Context, input DecisionInput) (PolicyEvaluation, error) {
				// Reentry would deadlock if the broker invoked policy under its locks.
				if _, err := b.Get(bg, input.Session.ID); err != nil {
					t.Error(err)
				}
				pending, err := b.Inputs(bg, input.Session.ID)
				if err != nil || len(pending) != 1 || pending[0].Request.ID != input.Request.ID {
					t.Errorf("policy before durable request: %+v %v", pending, err)
				}
				events, _, err := b.replay(input.Session.ID, 0)
				found := false
				for _, event := range events {
					found = found || event.Kind == "input.request" && event.ItemID == input.Request.ID
				}
				if err != nil || !found {
					t.Error("policy ran before request was journaled")
				}
				// A callback cannot mutate the retained provider option list.
				input.Request.Options[0].ID = "forged-option"
				live, _ := b.get(input.Session.ID)
				live.mu.Lock()
				cwd := live.cwd
				live.mu.Unlock()
				if cwd == "" || input.Cwd != cwd {
					t.Errorf("cwd=%q session cwd=%q", input.Cwd, cwd)
				}
				if reserved.Add(1) != 1 {
					t.Error("policy reserved more than one use")
				}
				return evaluation(mode), ctx.Err()
			}); err != nil {
				t.Fatal(err)
			}
			if _, err := b.Prompt(bg, s.ID, "turn", PromptRequest{Text: "please run"}); err != nil {
				t.Fatal(err)
			}
			select {
			case answer := <-answers:
				expected := "yes"
				if mode == DecisionDeny {
					expected = "no"
				}
				if answer.OptionID != expected {
					t.Fatalf("native answer=%+v", answer)
				}
			case <-time.After(time.Second):
				t.Fatal("policy answer never reached provider")
			}
			waitState(t, b, s.ID, "ready")
			events, _, _ := b.replay(s.ID, 0)
			count := 0
			for _, event := range events {
				if event.Kind == "input.submitted" {
					count++
					if event.Decision == nil || event.Decision.Actor == nil || event.Decision.Actor.ID != "station-owner" || event.Decision.PolicyID != "operator-policy" || event.Decision.PolicyRevision != "policy-revision-4" {
						t.Fatalf("policy receipt=%+v", event.Decision)
					}
				}
			}
			if count != 1 || reserved.Load() != 1 {
				t.Fatalf("submissions=%d reservations=%d", count, reserved.Load())
			}
		})
	}
}

func TestBrokerManualDecisionWinsWhilePolicyIsPending(t *testing.T) {
	b, s, answers := policyBroker(t, policyRequestFixture())
	entered, release := make(chan struct{}), make(chan struct{})
	if err := b.SetDecisionEvaluator(func(ctx context.Context, input DecisionInput) (PolicyEvaluation, error) {
		close(entered)
		select {
		case <-release:
		case <-ctx.Done():
			return PolicyEvaluation{}, ctx.Err()
		}
		return evaluation(DecisionAuto), nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Prompt(bg, s.ID, "race", PromptRequest{Text: "please run"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("policy was not invoked")
	}
	pending, err := b.Inputs(bg, s.ID)
	if err != nil || len(pending) != 1 {
		t.Fatal("missing pending decision")
	}
	if err = b.Answer(WithDecisionActor(context.Background(), DecisionActor{ID: "manual-operator"}), s.ID, pending[0].Request.ID, Answer{OptionID: "no"}); err != nil {
		t.Fatal(err)
	}
	close(release)
	select {
	case answer := <-answers:
		if answer.OptionID != "no" {
			t.Fatalf("policy replaced manual decision: %+v", answer)
		}
	case <-time.After(time.Second):
		t.Fatal("manual answer blocked behind evaluator")
	}
	waitState(t, b, s.ID, "ready")
	b.Close()
	events, _, _ := b.replay(s.ID, 0)
	count := 0
	for _, event := range events {
		if event.Kind == "input.submitted" {
			count++
			if event.Decision.Actor.ID != "manual-operator" || event.Decision.PolicyID != "" {
				t.Fatalf("wrong winner receipt=%+v", event.Decision)
			}
		}
	}
	if count != 1 {
		t.Fatalf("decision CAS produced %d receipts", count)
	}
}

func TestBrokerPolicyCannotInventAnswersOrReviewTruncatedOperations(t *testing.T) {
	for _, kind := range []string{"question", "plan", "truncated"} {
		t.Run(kind, func(t *testing.T) {
			request := policyRequestFixture()
			switch kind {
			case "truncated":
				request.Details.Truncated = true
			default:
				request.Kind = kind
			}
			b, s, _ := policyBroker(t, request)
			var calls atomic.Int32
			b.SetDecisionEvaluator(func(context.Context, DecisionInput) (PolicyEvaluation, error) {
				calls.Add(1)
				return evaluation(DecisionAuto), nil
			})
			if _, err := b.Prompt(bg, s.ID, "manual-only", PromptRequest{Text: "please run"}); err != nil {
				t.Fatal(err)
			}
			waitState(t, b, s.ID, "awaiting-input")
			pending, _ := b.Inputs(bg, s.ID)
			if calls.Load() != 0 {
				t.Fatal("policy evaluated a user question, a plan or a truncated review")
			}
			if err := b.Answer(bg, s.ID, pending[0].Request.ID, Answer{Cancel: true}); err != nil {
				t.Fatal(err)
			}
			waitState(t, b, s.ID, "ready")
		})
	}
	request := policyRequestFixture()
	request.Options = []Option{{ID: "persistent", Kind: "allow_always"}}
	b, s, _ := policyBroker(t, request)
	evaluated := make(chan struct{})
	b.SetDecisionEvaluator(func(context.Context, DecisionInput) (PolicyEvaluation, error) {
		defer close(evaluated)
		return evaluation(DecisionAuto), nil
	})
	b.Prompt(bg, s.ID, "no-public-once", PromptRequest{Text: "please run"})
	select {
	case <-evaluated:
	case <-time.After(time.Second):
		t.Fatal("policy was not evaluated")
	}
	pending, _ := b.Inputs(bg, s.ID)
	if len(pending) != 1 || pending[0].Submitted {
		t.Fatal("policy manufactured unsupported allow_once")
	}
	if err := b.Answer(bg, s.ID, pending[0].Request.ID, Answer{Cancel: true}); err != nil {
		t.Fatal(err)
	}
	waitState(t, b, s.ID, "ready")
}

func TestBrokerPolicyFailureLeavesManualRequestAvailable(t *testing.T) {
	b, s, _ := policyBroker(t, policyRequestFixture())
	evaluated := make(chan struct{})
	b.SetDecisionEvaluator(func(context.Context, DecisionInput) (PolicyEvaluation, error) {
		close(evaluated)
		return PolicyEvaluation{}, errors.New("policy backend unavailable")
	})
	b.Prompt(bg, s.ID, "policy-error", PromptRequest{Text: "please run"})
	select {
	case <-evaluated:
	case <-time.After(time.Second):
		t.Fatal("policy was not evaluated")
	}
	pending, _ := b.Inputs(bg, s.ID)
	if len(pending) != 1 || pending[0].Submitted {
		t.Fatal("unavailable policy resolved the request")
	}
	b.Answer(bg, s.ID, pending[0].Request.ID, Answer{Cancel: true})
	waitState(t, b, s.ID, "ready")
}

func TestBrokerPolicyReevaluatesDurablePendingRequests(t *testing.T) {
	b, s, answers := policyBroker(t, policyRequestFixture())
	if _, err := b.Prompt(bg, s.ID, "pending-then-policy", PromptRequest{Text: "please run"}); err != nil {
		t.Fatal(err)
	}
	waitState(t, b, s.ID, "awaiting-input")
	pending, err := b.Inputs(bg, s.ID)
	if err != nil || len(pending) != 1 || pending[0].Submitted {
		t.Fatalf("pending %+v %v", pending, err)
	}
	var seen atomic.Pointer[Request]
	b.SetDecisionEvaluator(func(_ context.Context, input DecisionInput) (PolicyEvaluation, error) {
		seen.Store(&input.Request)
		return evaluation(DecisionAuto), nil
	})
	if err := b.EvaluateInputs(bg, s.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case answer := <-answers:
		if answer.OptionID != "yes" || seen.Load() == nil || seen.Load().ID != pending[0].Request.ID {
			t.Fatalf("answer %+v for %v", answer, seen.Load())
		}
	case <-time.After(time.Second):
		t.Fatal("policy did not resolve retained request")
	}
	waitState(t, b, s.ID, "ready")
	// A request that already resolved is not evaluated again.
	var calls atomic.Int32
	b.SetDecisionEvaluator(func(context.Context, DecisionInput) (PolicyEvaluation, error) {
		calls.Add(1)
		return evaluation(DecisionAuto), nil
	})
	if err := b.EvaluateInputs(bg, s.ID); err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	if calls.Load() != 0 {
		t.Fatal("completed request was evaluated again")
	}
}
