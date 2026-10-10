package agentruntime

import (
	"context"
	"encoding/json"
)

// DecisionMode is what a host's evaluator decided for a pending request.
type DecisionMode string

const (
	DecisionManual DecisionMode = "manual"
	DecisionAuto   DecisionMode = "auto"
	DecisionDeny   DecisionMode = "deny"
)

// PolicyEvaluation is the answer of a DecisionEvaluator. Only auto and deny
// with a complete audit identity resolve a request; anything else leaves it to
// the operator. The host owns the policy: its rules, scopes, lifetimes and the
// atomic use of a rule are none of this package's business.
type PolicyEvaluation struct {
	Mode           DecisionMode  `json:"mode"`
	PolicyID       string        `json:"policyId,omitempty"`
	PolicyRevision string        `json:"policyRevision,omitempty"`
	RuleID         string        `json:"ruleId,omitempty"`
	Actor          DecisionActor `json:"actor"`
	Reason         string        `json:"reason"`
}

// DecisionInput describes one pending permission request to the evaluator. Cwd
// is the prepared working directory of the session; it never leaves the process.
type DecisionInput struct {
	Session Session `json:"session"`
	Request Request `json:"request"`
	Cwd     string  `json:"-"`
}

// DecisionEvaluator is a trusted host hook. It must respect ctx and atomically
// reserve a matching policy use before returning auto or deny. The broker owns
// request resolution; reservation does not mean a provider accepted the answer,
// and a concurrent manual answer may win the request compare-and-swap.
type DecisionEvaluator func(context.Context, DecisionInput) (PolicyEvaluation, error)

// SetDecisionEvaluator installs the host's evaluator. It sees permission
// requests whose preview is complete, never questions or plans, and its answer
// can only select an offered one-time allow or reject option.
func (b *Broker) SetDecisionEvaluator(evaluate DecisionEvaluator) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return ErrUnavailable
	}
	b.decisionEvaluator = evaluate
	return nil
}

// evaluable reports whether the evaluator may be asked about r.
func evaluable(r Request) bool {
	return r.Kind == "permission" && len(r.Questions) == 0 && (r.Details == nil || !r.Details.Truncated)
}

func (b *Broker) evaluatePending(ctx context.Context, session Session, request Request, runtimeCwd string) {
	if !evaluable(request) {
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
		result, err := evaluate(ctx, DecisionInput{Session: session, Request: copyRequest, Cwd: runtimeCwd})
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
			// An operator can resolve the durable request while the evaluation is
			// pending. Losing that compare-and-swap never overrides or retries
			// their answer.
			_ = b.Answer(actorContext, session.ID, request.ID, Answer{OptionID: option.ID})
			return
		}
	}()
}

// EvaluateInputs asks the evaluator again about the requests that still await
// an answer, for example after the operator changed the host's policy. It
// accepts neither rules nor answers and resolves through the same compare-and-swap.
func (b *Broker) EvaluateInputs(_ context.Context, id string) error {
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
