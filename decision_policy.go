package agentruntime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"
)

type DecisionMode string

const (
	DecisionManual DecisionMode = "manual"
	DecisionAuto   DecisionMode = "auto"
	DecisionDeny   DecisionMode = "deny"
	// Physical effects always need short, exact grants. The product owner may
	// choose tighter limits; provider full-access options never enter this policy.
	MaxGCSDecisionLifetime    = 5 * time.Minute
	MaxNativeDecisionLifetime = time.Hour
)

// DecisionFacts are supplied by the authority that owns the operation. No
// operation, target or effect is inferred from a model's message or tool title.
// ParametersDigest covers that authority's canonical, complete admission input.
type DecisionFacts struct {
	Operation           string       `json:"operation"`
	ExperimentID        string       `json:"experimentId"`
	ConversationID      string       `json:"conversationId"`
	Workspace           WorkspaceRef `json:"workspace"`
	ExperimentSessionID string       `json:"experimentSessionId,omitempty"`
	TargetID            string       `json:"targetId"`
	ParametersDigest    string       `json:"parametersDigest"`
}

type DecisionRule struct {
	ID            string        `json:"id"`
	Mode          DecisionMode  `json:"mode"`
	Scope         DecisionFacts `json:"scope"`
	IssuedAt      time.Time     `json:"issuedAt"`
	ExpiresAt     time.Time     `json:"expiresAt"`
	RemainingUses uint64        `json:"remainingUses"`
}

// Persistence and atomic use consumption belong to the product backend. This
// snapshot carries no executable credentials and the evaluator performs no I/O.
type DecisionPolicySnapshot struct {
	ID       string         `json:"id"`
	Revision string         `json:"revision"`
	Actor    DecisionActor  `json:"actor"`
	Rules    []DecisionRule `json:"rules"`
}

type PolicyEvaluation struct {
	Mode           DecisionMode  `json:"mode"`
	PolicyID       string        `json:"policyId,omitempty"`
	PolicyRevision string        `json:"policyRevision,omitempty"`
	RuleID         string        `json:"ruleId,omitempty"`
	Actor          DecisionActor `json:"actor"`
	Reason         string        `json:"reason"`
}

func knownDecisionOperation(operation string) bool {
	switch operation {
	case "native.shell.execute", "native.file.read", "native.file.write", "native.network.access",
		"gcs.robot.unlock", "gcs.mode.change", "gcs.workflow.start", "gcs.workflow.stop", "gcs.workflow.confirm",
		"gcs.experiment.start", "gcs.experiment.stop", "gcs.experiment.restart":
		return true
	default:
		return false
	}
}

func validDecisionFacts(facts DecisionFacts) bool {
	if !knownDecisionOperation(facts.Operation) || !safeID.MatchString(facts.ExperimentID) ||
		!safeID.MatchString(facts.TargetID) || !digest.MatchString(facts.ParametersDigest) {
		return false
	}
	// Ordinary GCS commands have no conversation or filesystem workspace. Empty
	// fields represent that exact scope, never a wildcard for agent requests.
	if strings.HasPrefix(facts.Operation, "native.") || facts.ConversationID != "" || facts.Workspace != (WorkspaceRef{}) {
		if !safeID.MatchString(facts.ConversationID) || !safeID.MatchString(facts.Workspace.ID) || facts.Workspace.Revision == "" || len(facts.Workspace.Revision) > 256 || strings.ContainsAny(facts.Workspace.Revision, "\x00\r\n") {
			return false
		}
	}
	if facts.ExperimentSessionID != "" && !safeID.MatchString(facts.ExperimentSessionID) {
		return false
	}
	return true
}

// EvaluateDecisionPolicy is deliberately pure. A matching auto/deny result is
// only a candidate: the owning backend must atomically consume RemainingUses
// against PolicyRevision before returning it to an execution adapter.
// Exact scope equality prevents CLI permission from granting any GCS authority.
func EvaluateDecisionPolicy(now time.Time, facts DecisionFacts, policy DecisionPolicySnapshot) PolicyEvaluation {
	result := PolicyEvaluation{Mode: DecisionManual, Reason: "no_matching_rule"}
	if !validDecisionFacts(facts) {
		result.Reason = "unknown_or_incomplete_operation"
		return result
	}
	if !safeID.MatchString(policy.ID) || policy.Revision == "" || len(policy.Revision) > 256 || !safeID.MatchString(policy.Actor.ID) || len(policy.Rules) > 256 {
		result.Reason = "invalid_policy"
		return result
	}
	priority := 0
	for _, rule := range policy.Rules {
		if !safeID.MatchString(rule.ID) || rule.Scope != facts || rule.RemainingUses == 0 || rule.IssuedAt.IsZero() || rule.IssuedAt.After(now) || !rule.ExpiresAt.After(now) {
			continue
		}
		lifetime := MaxNativeDecisionLifetime
		if strings.HasPrefix(facts.Operation, "gcs.") {
			lifetime = MaxGCSDecisionLifetime
		}
		if rule.ExpiresAt.Sub(rule.IssuedAt) > lifetime {
			continue
		}
		rank := 0
		switch rule.Mode {
		case DecisionAuto:
			rank = 1
		case DecisionManual:
			rank = 2
		case DecisionDeny:
			rank = 3
		}
		if rank <= priority {
			continue
		}
		priority = rank
		result = PolicyEvaluation{Mode: rule.Mode, PolicyID: policy.ID, PolicyRevision: policy.Revision, RuleID: rule.ID, Actor: policy.Actor, Reason: "matched_rule"}
	}
	return result
}

// DecisionParametersDigest is a shared encoding helper, not a source of scope.
// Callers must pass the authoritative canonical admission parameters, never a
// model summary, command label, or a truncated display preview.
func DecisionParametersDigest(parameters any) (string, error) {
	data, err := json.Marshal(parameters)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
