package nativeagent

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func policyFixture(facts DecisionFacts, mode DecisionMode, now time.Time) DecisionPolicySnapshot {
	return DecisionPolicySnapshot{ID: "operator-policy", Revision: "policy-revision-4", Actor: DecisionActor{ID: "station-owner"}, Rules: []DecisionRule{{ID: "one-rule", Mode: mode, Scope: facts, IssuedAt: now.Add(-time.Second), ExpiresAt: now.Add(time.Minute), RemainingUses: 1}}}
}

func factsFixture() DecisionFacts {
	return DecisionFacts{Operation: "native.shell.execute", ExperimentID: "exp-a", ConversationID: "conversation-a", Workspace: WorkspaceRef{ID: "project", Revision: "reviewed"}, TargetID: "local", ParametersDigest: strings.Repeat("a", 64)}
}

func TestDecisionPolicyExactScopeAndAuthorityNamespaces(t *testing.T) {
	now := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	facts := factsFixture()
	policy := policyFixture(facts, DecisionAuto, now)
	result := EvaluateDecisionPolicy(now, facts, policy)
	if result.Mode != DecisionAuto || result.Actor.ID != "station-owner" || result.PolicyRevision != policy.Revision || result.RuleID != "one-rule" {
		t.Fatalf("evaluation=%+v", result)
	}
	for name, change := range map[string]func(*DecisionFacts){
		"experiment":           func(f *DecisionFacts) { f.ExperimentID = "exp-b" },
		"conversation":         func(f *DecisionFacts) { f.ConversationID = "conversation-b" },
		"workspace":            func(f *DecisionFacts) { f.Workspace.ID = "another-project" },
		"workspace-revision":   func(f *DecisionFacts) { f.Workspace.Revision = "changed" },
		"experiment-session":   func(f *DecisionFacts) { f.ExperimentSessionID = "new-run" },
		"target":               func(f *DecisionFacts) { f.TargetID = "robot-b" },
		"parameters":           func(f *DecisionFacts) { f.ParametersDigest = strings.Repeat("b", 64) },
		"robot-permission":     func(f *DecisionFacts) { f.Operation = "gcs.robot.unlock" },
		"provider-full-access": func(f *DecisionFacts) { f.Operation = "full-access" },
		"unknown":              func(f *DecisionFacts) { f.Operation = "gcs.anything" },
	} {
		t.Run(name, func(t *testing.T) {
			changed := facts
			change(&changed)
			if got := EvaluateDecisionPolicy(now, changed, policy); got.Mode != DecisionManual {
				t.Fatalf("scope mismatch=%+v", got)
			}
		})
	}
	if policy.Rules[0].RemainingUses != 1 {
		t.Fatal("pure evaluation consumed a persisted rule")
	}
	robot := facts
	robot.Operation = "gcs.robot.unlock"
	robot.ExperimentSessionID = "frozen-run"
	robot.TargetID = "robot-a"
	robotPolicy := policyFixture(robot, DecisionAuto, now)
	if got := EvaluateDecisionPolicy(now, robot, robotPolicy); got.Mode != DecisionAuto {
		t.Fatalf("exact GCS grant=%+v", got)
	}
	if got := EvaluateDecisionPolicy(now, facts, robotPolicy); got.Mode != DecisionManual {
		t.Fatal("robot approval escaped into native execution")
	}
}

func TestDecisionPolicyExpiryUsesAndDenyPrecedence(t *testing.T) {
	now := time.Now().UTC()
	facts := factsFixture()
	for name, change := range map[string]func(*DecisionPolicySnapshot){
		"expired":          func(p *DecisionPolicySnapshot) { p.Rules[0].ExpiresAt = now },
		"exhausted":        func(p *DecisionPolicySnapshot) { p.Rules[0].RemainingUses = 0 },
		"future":           func(p *DecisionPolicySnapshot) { p.Rules[0].IssuedAt = now.Add(time.Minute) },
		"unbounded-ttl":    func(p *DecisionPolicySnapshot) { p.Rules[0].ExpiresAt = now.Add(2 * time.Hour) },
		"operator-missing": func(p *DecisionPolicySnapshot) { p.Actor.ID = "" },
		"revision-missing": func(p *DecisionPolicySnapshot) { p.Revision = "" },
		"wildcard":         func(p *DecisionPolicySnapshot) { p.Rules[0].Scope.ConversationID = "*" },
	} {
		t.Run(name, func(t *testing.T) {
			policy := policyFixture(facts, DecisionAuto, now)
			change(&policy)
			if result := EvaluateDecisionPolicy(now, facts, policy); result.Mode != DecisionManual {
				t.Fatalf("invalid rule=%+v", result)
			}
		})
	}
	policy := policyFixture(facts, DecisionAuto, now)
	manual := policy.Rules[0]
	manual.ID = "manual"
	manual.Mode = DecisionManual
	deny := policy.Rules[0]
	deny.ID = "deny"
	deny.Mode = DecisionDeny
	policy.Rules = append(policy.Rules, manual)
	if result := EvaluateDecisionPolicy(now, facts, policy); result.Mode != DecisionManual || result.RuleID != "manual" {
		t.Fatalf("manual override=%+v", result)
	}
	policy.Rules = append(policy.Rules, deny)
	if result := EvaluateDecisionPolicy(now, facts, policy); result.Mode != DecisionDeny || result.RuleID != "deny" {
		t.Fatalf("deny override=%+v", result)
	}
	robot := facts
	robot.Operation = "gcs.experiment.restart"
	policy = policyFixture(robot, DecisionAuto, now)
	policy.Rules[0].ExpiresAt = now.Add(6 * time.Minute)
	if result := EvaluateDecisionPolicy(now, robot, policy); result.Mode != DecisionManual {
		t.Fatalf("long robot grant=%+v", result)
	}
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
	s, _ = b.Get(s.ID)
	return b, s, result
}

func TestBrokerPolicyUsesDurableRequestsAndNormalDecisionAudit(t *testing.T) {
	for _, mode := range []DecisionMode{DecisionAuto, DecisionDeny} {
		t.Run(string(mode), func(t *testing.T) {
			b, s, answers := policyBroker(t, policyRequestFixture())
			var reserved atomic.Int32
			if err := b.SetDecisionEvaluator(func(ctx context.Context, input DecisionInput) (PolicyEvaluation, error) {
				// Reentry would deadlock if the broker invoked policy under its locks.
				if _, err := b.Get(input.Session.ID); err != nil {
					t.Error(err)
				}
				pending, err := b.Inputs(input.Session.ID)
				if err != nil || len(pending) != 1 || pending[0].Request.ID != input.Request.ID {
					t.Errorf("policy before durable request: %+v %v", pending, err)
				}
				events, _, err := b.Replay(input.Session.ID, 0)
				found := false
				for _, event := range events {
					found = found || event.Kind == "input.request" && event.ItemID == input.Request.ID
				}
				if err != nil || !found {
					t.Error("policy ran before request was journaled")
				}
				// A callback cannot mutate the retained provider option list.
				input.Request.Options[0].ID = "forged-option"
				now := time.Now().UTC()
				result := EvaluateDecisionPolicy(now, input.Facts, policyFixture(input.Facts, mode, now))
				if reserved.Add(1) != 1 {
					t.Error("policy reserved more than one use")
				}
				return result, ctx.Err()
			}); err != nil {
				t.Fatal(err)
			}
			if _, err := b.Prompt(s.ID, "turn", "please run"); err != nil {
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
			events, _, _ := b.Replay(s.ID, 0)
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
		now := time.Now().UTC()
		return EvaluateDecisionPolicy(now, input.Facts, policyFixture(input.Facts, DecisionAuto, now)), nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Prompt(s.ID, "race", "please run"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("policy was not invoked")
	}
	pending, err := b.Inputs(s.ID)
	if err != nil || len(pending) != 1 {
		t.Fatal("missing pending decision")
	}
	if err = b.AnswerContext(WithDecisionActor(context.Background(), DecisionActor{ID: "manual-operator"}), s.ID, pending[0].Request.ID, Answer{OptionID: "no"}); err != nil {
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
	events, _, _ := b.Replay(s.ID, 0)
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

func TestBrokerPolicyCannotInventAnswersOrUnknownEffects(t *testing.T) {
	for _, kind := range []string{"question", "plan", "unknown"} {
		t.Run(kind, func(t *testing.T) {
			request := policyRequestFixture()
			request.Kind = kind
			if kind == "unknown" {
				request.Kind = "permission"
				request.SourceMethod = "unrecognized:tool"
				request.Title = "gcs.robot.unlock native.shell.execute"
			}
			b, s, _ := policyBroker(t, request)
			var calls atomic.Int32
			b.SetDecisionEvaluator(func(context.Context, DecisionInput) (PolicyEvaluation, error) {
				calls.Add(1)
				return PolicyEvaluation{Mode: DecisionAuto}, nil
			})
			if _, err := b.Prompt(s.ID, "manual-only", "please run"); err != nil {
				t.Fatal(err)
			}
			waitState(t, b, s.ID, "awaiting-input")
			pending, _ := b.Inputs(s.ID)
			if calls.Load() != 0 {
				t.Fatal("policy evaluated a user question, plan, or unknown effect")
			}
			if err := b.Answer(s.ID, pending[0].Request.ID, Answer{Cancel: true}); err != nil {
				t.Fatal(err)
			}
			waitState(t, b, s.ID, "ready")
		})
	}
	request := policyRequestFixture()
	request.Options = []Option{{ID: "persistent", Kind: "allow_always"}}
	b, s, _ := policyBroker(t, request)
	evaluated := make(chan struct{})
	b.SetDecisionEvaluator(func(_ context.Context, input DecisionInput) (PolicyEvaluation, error) {
		defer close(evaluated)
		now := time.Now().UTC()
		return EvaluateDecisionPolicy(now, input.Facts, policyFixture(input.Facts, DecisionAuto, now)), nil
	})
	b.Prompt(s.ID, "no-public-once", "please run")
	select {
	case <-evaluated:
	case <-time.After(time.Second):
		t.Fatal("policy was not evaluated")
	}
	pending, _ := b.Inputs(s.ID)
	if len(pending) != 1 || pending[0].Submitted {
		t.Fatal("policy manufactured unsupported allow_once")
	}
	if err := b.Answer(s.ID, pending[0].Request.ID, Answer{Cancel: true}); err != nil {
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
	b.Prompt(s.ID, "policy-error", "please run")
	select {
	case <-evaluated:
	case <-time.After(time.Second):
		t.Fatal("policy was not evaluated")
	}
	pending, _ := b.Inputs(s.ID)
	if len(pending) != 1 || pending[0].Submitted {
		t.Fatal("unavailable policy resolved the request")
	}
	b.Answer(s.ID, pending[0].Request.ID, Answer{Cancel: true})
	waitState(t, b, s.ID, "ready")
}

func TestDecisionPolicyOrdinaryGCSHasExactEmptyConversationScope(t *testing.T) {
	now := time.Now().UTC()
	facts := factsFixture()
	facts.Operation = "gcs.workflow.confirm"
	facts.ConversationID = ""
	facts.Workspace = WorkspaceRef{}
	policy := policyFixture(facts, DecisionAuto, now)
	if got := EvaluateDecisionPolicy(now, facts, policy); got.Mode != DecisionAuto {
		t.Fatalf("ordinary GCS scope %+v", got)
	}
	agent := facts
	agent.ConversationID = "chat"
	agent.Workspace = WorkspaceRef{ID: "project", Revision: "r1"}
	if got := EvaluateDecisionPolicy(now, agent, policy); got.Mode != DecisionManual {
		t.Fatal("empty scope acted as wildcard")
	}
	facts.Operation = "native.shell.execute"
	policy = policyFixture(facts, DecisionAuto, now)
	if got := EvaluateDecisionPolicy(now, facts, policy); got.Mode != DecisionManual {
		t.Fatal("native policy accepted absent conversation/workspace")
	}
}

func TestBrokerPolicyReevaluatesDurablePendingAndExposesCanonicalFacts(t *testing.T) {
	b, s, answers := policyBroker(t, policyRequestFixture())
	if _, err := b.Prompt(s.ID, "pending-then-policy", "please run"); err != nil {
		t.Fatal(err)
	}
	waitState(t, b, s.ID, "awaiting-input")
	pending, err := b.Inputs(s.ID)
	if err != nil || len(pending) != 1 || pending[0].Facts == nil {
		t.Fatalf("facts %+v %v", pending, err)
	}
	facts := *pending[0].Facts
	if facts.ConversationID != s.ID || facts.Workspace != s.Scope.Workspace || facts.Operation != "native.shell.execute" {
		t.Fatalf("facts %+v", facts)
	}
	b.SetDecisionEvaluator(func(_ context.Context, input DecisionInput) (PolicyEvaluation, error) {
		if input.Facts != facts {
			t.Errorf("changed canonical facts")
		}
		return EvaluateDecisionPolicy(time.Now(), input.Facts, policyFixture(facts, DecisionAuto, time.Now())), nil
	})
	if err := b.EvaluateInputs(s.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case answer := <-answers:
		if answer.OptionID != "yes" {
			t.Fatalf("answer %+v", answer)
		}
	case <-time.After(time.Second):
		t.Fatal("policy did not resolve retained request")
	}
	waitState(t, b, s.ID, "ready")
	result, err := b.EvaluateDriverDecision(context.Background(), DecisionInput{Session: s, Request: pending[0].Request, Facts: facts})
	if err != nil || result.Mode != DecisionManual {
		t.Fatalf("completed runtime reused %+v %v", result, err)
	}
}
