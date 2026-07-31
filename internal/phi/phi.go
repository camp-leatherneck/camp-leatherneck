// Package phi implements the structural PHI (Protected Health Information)
// containment guard for Camp Leatherneck dispatch.
//
// Policy (see ~/gt/directives/lt.md "PHI Containment Contract"): no real PHI
// is ever transmitted to any AI runtime — Claude, Codex, Bedrock, any
// external API, any third-party model, any unaudited sandbox. This package
// is the mechanical enforcement of that policy. It is deliberately NOT
// prose — every dispatch path must call Guard before any bead content is
// handed to an agent session, and Guard fails closed on ambiguity.
//
// This package has no dependency on internal/cmd or any dispatch machinery
// so it can be unit tested in isolation and cannot accidentally participate
// in an import cycle with the code that calls it.
package phi

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Classification is the PHI-exposure classification for a unit of work
// (a bead, a mail message, a task — anything about to be placed in an AI
// runtime's context).
type Classification string

const (
	// DataPresent means the work package itself is asserted or detected to
	// contain real PHI. Must never dispatch to any AI runtime, no exceptions,
	// no override.
	DataPresent Classification = "phi_data_present"

	// SystemRelated means the work touches a PHI-bearing system or
	// integration but the work package itself is asserted to be sanitized
	// (synthetic data, redacted logs, structural description). May dispatch
	// only when sanitization evidence is present.
	SystemRelated Classification = "phi_system_related"

	// NonPHI means no PHI-bearing signal was found. This is the default
	// resolution for ordinary engineering work — the guard does not require
	// every bead to be pre-labeled, or it would block all routine work.
	NonPHI Classification = "non_phi"

	// ClassificationRequired means a signal was ambiguous enough that the
	// guard cannot safely resolve to NonPHI or confidently to DataPresent/
	// SystemRelated. Fails closed: blocks dispatch until a human resolves
	// the ambiguity (label the work explicitly, or escalate).
	ClassificationRequired Classification = "classification_required"
)

// Explicit label vocabulary. Authors/dispatchers may assert a classification
// via bead labels, but labels are never the sole check — see Classify.
const (
	LabelDataPresent = "phi:data-present"
	LabelSanitized   = "phi:sanitized" // evidence a phi_system_related work package has been sanitized
	LabelNone        = "phi:none"      // explicit author affirmation of no PHI
)

// WorkItem is the minimal, dispatch-machinery-agnostic view of a unit of
// work that the guard classifies. Callers (internal/cmd, internal/mail,
// etc.) adapt their own bead/message types into this shape at the call
// site — this package never imports beads/cmd types, by design.
type WorkItem struct {
	Title       string
	Description string
	Labels      []string
}

// Decision is the result of classifying a WorkItem: the classification,
// whether dispatch is allowed, and an auditable (content-free) reason.
type Decision struct {
	Classification Classification
	Allowed        bool
	Reason         string // safe to log — never contains matched sensitive substrings
}

// Policy is the defense-in-depth signal set. A default, embedded policy
// ships with the binary; operators may extend it without a rebuild via
// ~/gt/settings/phi_policy.json (see LoadPolicy).
type Policy struct {
	// SensitiveKeywords are case-insensitive substrings that, if found in
	// title/description, indicate the work touches a known PHI-bearing
	// system or integration (e.g. an EHR adapter). A hit resolves to
	// SystemRelated, not DataPresent — the keyword indicates *context*,
	// not that raw patient data is present in the work package itself.
	SensitiveKeywords []string `json:"sensitive_keywords"`

	// SensitivePathIndicators are case-insensitive substrings matched
	// against title/description that indicate a known PHI-bearing
	// filesystem/resource path. Same resolution as SensitiveKeywords.
	SensitivePathIndicators []string `json:"sensitive_path_indicators"`

	// StrongDataSignalKeywords are case-insensitive substrings that
	// indicate actual patient data, not just system context (e.g. someone
	// pasted a real record into a bead description). A hit resolves to
	// DataPresent — hard block, no override.
	StrongDataSignalKeywords []string `json:"strong_data_signal_keywords"`
}

// DefaultPolicy is the built-in defense-in-depth signal set. Kept
// intentionally small and specific to Alto's known PHI-bearing systems
// (Open Dental / EHR integration) rather than an attempt at general PII
// detection — see phi.go package doc for the honesty note on scope.
func DefaultPolicy() Policy {
	return Policy{
		SensitiveKeywords: []string{
			"open dental", "opendental", "od adapter", "ehr adapter",
			"patient record", "patient data", "patient chart",
			"protected health information", "phi ", "hipaa",
			"divergent", "dental intelligence",
		},
		SensitivePathIndicators: []string{
			"open_dental_client", "adapters/ehr", "/patients", "/patient/",
			"audit_logs", "portal/images", "portal_images",
		},
		StrongDataSignalKeywords: []string{
			"real patient", "actual patient", "live patient data",
			"phi:", "ssn:", "date of birth:", "dob:", "mrn:",
		},
	}
}

// LoadPolicy loads the operator-editable policy override at
// <townRoot>/settings/phi_policy.json if present, merging its lists onto
// the default policy (additive — operators extend, they don't have to
// restate the defaults). Returns DefaultPolicy() if the file is absent.
// A malformed override file is a hard error (fail closed on policy load,
// consistent with the rest of this package) rather than silently falling
// back to defaults.
func LoadPolicy(townRoot string) (Policy, error) {
	policy := DefaultPolicy()
	if townRoot == "" {
		return policy, nil
	}
	path := townRoot + "/settings/phi_policy.json"
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return policy, nil
		}
		return Policy{}, fmt.Errorf("phi: reading policy override %s: %w", path, err)
	}
	var override Policy
	if err := json.Unmarshal(data, &override); err != nil {
		return Policy{}, fmt.Errorf("phi: parsing policy override %s: %w", path, err)
	}
	policy.SensitiveKeywords = append(policy.SensitiveKeywords, override.SensitiveKeywords...)
	policy.SensitivePathIndicators = append(policy.SensitivePathIndicators, override.SensitivePathIndicators...)
	policy.StrongDataSignalKeywords = append(policy.StrongDataSignalKeywords, override.StrongDataSignalKeywords...)
	return policy, nil
}

// structuredIdentifierPattern is a conservative heuristic for identifier
// shapes that commonly appear in real patient records (SSN-like, MRN-like
// numeric-with-separators runs of a length unlikely in ordinary engineering
// text). This is NOT a general PII/PHI detector — it is one signal among
// several in a defense-in-depth stack, and it is intentionally narrow to
// avoid false-positive-driven bureaucracy on ordinary numeric content
// (bead IDs, ports, versions). Documented limitation, not a claim of
// completeness.
var structuredIdentifierPattern = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)

func containsAny(haystack string, needles []string) (bool, string) {
	lower := strings.ToLower(haystack)
	for _, n := range needles {
		if n == "" {
			continue
		}
		if strings.Contains(lower, strings.ToLower(n)) {
			return true, n
		}
	}
	return false, ""
}

func hasLabel(labels []string, want string) bool {
	for _, l := range labels {
		if strings.EqualFold(strings.TrimSpace(l), want) {
			return true
		}
	}
	return false
}

// Classify determines the PHI classification of a WorkItem using explicit
// labels AND defense-in-depth content signals. Labels are never the sole
// check — a WorkItem labeled phi:none is still scanned for strong-data and
// sensitive-system signals, because a mislabeled or forgotten label must
// not be a bypass (requirement: "do not rely solely on a manual phi:true
// label").
//
// Resolution order (first match wins):
//  1. Explicit phi:data-present label            -> DataPresent
//  2. Strong data-signal content match            -> DataPresent
//  3. Structured-identifier pattern match         -> DataPresent
//  4. Sensitive-system signal + phi:sanitized     -> SystemRelated (allowed)
//  5. Sensitive-system signal, no sanitized label -> SystemRelated (blocked)
//  6. No signals at all                           -> NonPHI (allowed)
//  7. Anything else ambiguous                     -> ClassificationRequired (blocked)
func Classify(item WorkItem, policy Policy) Decision {
	content := item.Title + "\n" + item.Description

	// 1. Explicit assertion of data presence — trust it immediately, no
	// need to scan further; the author is telling us directly.
	if hasLabel(item.Labels, LabelDataPresent) {
		return Decision{
			Classification: DataPresent,
			Allowed:        false,
			Reason:         "blocked: bead explicitly labeled " + LabelDataPresent,
		}
	}

	// 2. Strong data-signal content match (do NOT log the matched text —
	// it may itself be sensitive; log only the fact that a signal fired).
	if hit, _ := containsAny(content, policy.StrongDataSignalKeywords); hit {
		return Decision{
			Classification: DataPresent,
			Allowed:        false,
			Reason:         "blocked: strong PHI data-signal detected in work package content (signal redacted from this log)",
		}
	}

	// 3. Structured-identifier pattern (SSN-shaped, etc.) — again, do not
	// log the match.
	if structuredIdentifierPattern.MatchString(content) {
		return Decision{
			Classification: DataPresent,
			Allowed:        false,
			Reason:         "blocked: structured-identifier pattern detected in work package content (match redacted from this log)",
		}
	}

	// 4/5. Sensitive-system signal (context, not necessarily data).
	sensitiveHit, matchedKeyword := containsAny(content, policy.SensitiveKeywords)
	if !sensitiveHit {
		sensitiveHit, matchedKeyword = containsAny(content, policy.SensitivePathIndicators)
	}
	if sensitiveHit {
		if hasLabel(item.Labels, LabelSanitized) {
			return Decision{
				Classification: SystemRelated,
				Allowed:        true,
				Reason:         fmt.Sprintf("allowed: phi_system_related (matched %q), sanitization evidence present (%s)", matchedKeyword, LabelSanitized),
			}
		}
		return Decision{
			Classification: SystemRelated,
			Allowed:        false,
			Reason:         fmt.Sprintf("blocked: phi_system_related (matched %q) with no sanitization evidence — add %s only after confirming the work package contains no PHI", matchedKeyword, LabelSanitized),
		}
	}

	// 6. Explicit phi:none with no signals at all — clean, allow. (An
	// explicit phi:none with a signal present never reaches here — the
	// signal branches above already fired and won.)
	if hasLabel(item.Labels, LabelNone) {
		return Decision{
			Classification: NonPHI,
			Allowed:        true,
			Reason:         "allowed: non_phi (explicit " + LabelNone + ", no signals detected)",
		}
	}

	// 6b. No labels at all, no signals at all — this is the default state
	// for ordinary engineering work and must not require pre-labeling
	// every bead in the system. Allow.
	if len(item.Labels) == 0 || !hasAnyPHILabel(item.Labels) {
		return Decision{
			Classification: NonPHI,
			Allowed:        true,
			Reason:         "allowed: non_phi (no phi:* label, no defense-in-depth signal detected)",
		}
	}

	// 7. Fail closed: some phi:* label combination we don't recognize, or
	// a state this function's author didn't anticipate. Never guess.
	return Decision{
		Classification: ClassificationRequired,
		Allowed:        false,
		Reason:         "blocked: ambiguous PHI classification — resolve explicitly (phi:none if clean, phi:sanitized if system-related and sanitized, phi:data-present if real PHI) or escalate for classification",
	}
}

func hasAnyPHILabel(labels []string) bool {
	for _, l := range labels {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(l)), "phi:") {
			return true
		}
	}
	return false
}

// Guard is the enforcement entrypoint: classify, and return a non-nil error
// if dispatch must be blocked. Callers must invoke Guard unconditionally,
// before any bead/message content is handed to an agent session, and must
// NOT gate the call behind --force, --agent, or any other override flag —
// the requirement is that runtime overrides cannot bypass the guard, so
// the guard call itself takes no override parameter.
func Guard(item WorkItem, policy Policy) error {
	decision := Classify(item, policy)
	if decision.Allowed {
		return nil
	}
	return &BlockedError{Decision: decision}
}

// BlockedError is returned by Guard when dispatch is blocked. Error()
// returns only the auditable reason (never raw content), satisfying the
// "rejections must be auditable without reproducing sensitive content"
// requirement.
type BlockedError struct {
	Decision Decision
}

func (e *BlockedError) Error() string {
	return fmt.Sprintf("phi guard: dispatch blocked (%s): %s", e.Decision.Classification, e.Decision.Reason)
}
