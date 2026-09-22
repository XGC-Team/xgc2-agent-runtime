package nativeagent

import (
	"context"
	"encoding/json"
	"errors"
)

type DecisionInput struct {
	Session Session       `json:"session"`
	Request Request       `json:"request"`
	Facts   DecisionFacts `json:"facts"`
}

// DecisionEvaluator is a trusted product-owner hook. It must respect ctx and
// atomically reserve a matching policy use before returning auto or deny. The
// broker owns request resolution; reservation does not mean a provider accepted
// the answer, and a concurrent manual answer may win the request CAS.
type DecisionEvaluator func(context.Context, DecisionInput) (PolicyEvaluation, error)

func (b *Broker) SetDecisionEvaluator(evaluate DecisionEvaluator) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return ErrUnavailable
	}
	b.decisionEvaluator = evaluate
	return nil
}

// NativeDecisionFacts recognizes only adapter-owned structured command/network
// fields with known permission protocols. File previews and arbitrary ACP tool
// labels are not complete admission inputs; they stay manual until their owner
// supplies an exact structured adapter. No prompt, title or permission option
// (including full-access) can identify a ground-station effect.
func NativeDecisionFacts(session Session, request Request) (DecisionFacts, bool) {
	return nativeDecisionFacts(session, request, "")
}

func nativeDecisionFacts(session Session, request Request, runtimeCwd string) (DecisionFacts, bool) {
	facts := DecisionFacts{ExperimentID: session.Scope.Context.ID, ConversationID: session.ID, Workspace: session.Scope.Workspace, TargetID: "local"}
	if session.Scope.Context.Kind != "experiment" || request.Kind != "permission" || len(request.Questions) != 0 || request.Details == nil || request.Details.Truncated {
		return facts, false
	}
	details := request.Details
	cwd := details.Cwd
	if cwd == "" {
		cwd = runtimeCwd
	}
	var parameters any
	switch request.SourceMethod {
	case "item/commandExecution/requestApproval":
		if details.Network != nil {
			if details.Network.Host == "" || details.Network.Protocol == "" {
				return facts, false
			}
			facts.Operation = "native.network.access"
			parameters = struct{ Protocol, Host string }{details.Network.Protocol, details.Network.Host}
		} else {
			if details.Command == "" || cwd == "" {
				return facts, false
			}
			facts.Operation = "native.shell.execute"
			parameters = struct{ Command, Cwd string }{details.Command, cwd}
		}
	case "claude:can_use_tool":
		// Bash's security-relevant request is its exact command in this cwd.
		// Missing cwd uses only the broker's previously validated runtime cwd.
		if details.ToolName != "Bash" || details.Command == "" || cwd == "" {
			return facts, false
		}
		facts.Operation = "native.shell.execute"
		parameters = struct{ Command, Cwd string }{details.Command, cwd}
	default:
		return facts, false
	}
	fingerprint, err := DecisionParametersDigest(parameters)
	if err != nil {
		return facts, false
	}
	facts.ParametersDigest = fingerprint
	return facts, validDecisionFacts(facts)
}

func (b *Broker) evaluatePending(ctx context.Context, session Session, request Request, runtimeCwd string) {
	facts, known := nativeDecisionFacts(session, request, runtimeCwd)
	if !known {
		return
	}
	b.mu.Lock()
	evaluate := b.decisionEvaluator
	if b.closed || evaluate == nil {
		b.mu.Unlock()
		return
	}
	b.wg.Add(1)
	b.mu.Unlock()
	// Deep-copy the retained request before calling a host extension. Changing
	// callback input must never change the options that were shown and journaled.
	encoded, err := json.Marshal(request)
	if err != nil {
		b.wg.Done()
		return
	}
	var copyRequest Request
	if json.Unmarshal(encoded, &copyRequest) != nil {
		b.wg.Done()
		return
	}
	go func() {
		defer b.wg.Done()
		result, err := evaluate(ctx, DecisionInput{Session: session, Request: copyRequest, Facts: facts})
		if err != nil || ctx.Err() != nil || (result.Mode != DecisionAuto && result.Mode != DecisionDeny) ||
			!safeID.MatchString(result.PolicyID) || result.PolicyRevision == "" || !safeID.MatchString(result.RuleID) || !safeID.MatchString(result.Actor.ID) {
			return
		}
		kind := "allow_once"
		if result.Mode == DecisionDeny {
			kind = "reject_once"
		}
		for _, option := range request.Options {
			if option.Kind != kind {
				continue
			}
			actorContext := WithDecisionActor(ctx, result.Actor)
			actorContext = WithDecisionPolicy(actorContext, DecisionPolicy{ID: result.PolicyID, Revision: result.PolicyRevision})
			err = b.AnswerContext(actorContext, session.ID, request.ID, Answer{OptionID: option.ID})
			// An operator can resolve the durable request while policy evaluation
			// is pending. Losing that CAS never overrides or retries their answer.
			if err == nil || errors.Is(err, ErrConflict) || errors.Is(err, ErrStale) {
				return
			}
			return
		}
	}()
}

// DriverDecisionEvaluator is implemented by a product's enrolled host wrapper.
// The provider protocol and model never receive its decision authority.
type DriverDecisionEvaluator interface {
	EvaluateDecision(context.Context, DecisionInput) (PolicyEvaluation, error)
}

// EvaluateDriverDecision delegates only to the currently retained runtime. It
// calls the product wrapper without holding broker/session locks.
func (b *Broker) EvaluateDriverDecision(ctx context.Context, input DecisionInput) (PolicyEvaluation, error) {
	s, err := b.get(input.Session.ID)
	if err != nil {
		return PolicyEvaluation{}, err
	}
	s.mu.Lock()
	pending := s.inputs[input.Request.ID]
	if s.stopping || s.info.RuntimeID != input.Session.RuntimeID || pending == nil || !pending.active || pending.answer != nil {
		s.mu.Unlock()
		return PolicyEvaluation{Mode: DecisionManual, Reason: "request_no_longer_pending"}, nil
	}
	driver := s.driver
	s.mu.Unlock()
	if evaluator, ok := driver.(DriverDecisionEvaluator); ok {
		return evaluator.EvaluateDecision(ctx, input)
	}
	return PolicyEvaluation{Mode: DecisionManual, Reason: "no_product_authority"}, nil
}

// EvaluateInputs rechecks existing durable requests after an operator edits
// policy. It neither accepts rules nor answers and uses the same request CAS.
func (b *Broker) EvaluateInputs(id string) error {
	s, err := b.get(id)
	if err != nil {
		return err
	}
	s.mu.Lock()
	info, cwd := s.info, s.cwd
	requests := []Request{}
	for _, pending := range s.inputs {
		if pending.active && pending.answer == nil {
			requests = append(requests, pending.request)
		}
	}
	s.mu.Unlock()
	for _, request := range requests {
		b.evaluatePending(b.ctx, info, request, cwd)
	}
	return nil
}
