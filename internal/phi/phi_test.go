package phi

import (
	"os"
	"strings"
	"testing"
)

func TestClassify_ExplicitDataPresentLabel_Blocks(t *testing.T) {
	item := WorkItem{Title: "Fix billing report", Labels: []string{LabelDataPresent}}
	d := Classify(item, DefaultPolicy())
	if d.Classification != DataPresent || d.Allowed {
		t.Fatalf("got %+v, want blocked DataPresent", d)
	}
}

func TestClassify_StrongDataSignal_Blocks(t *testing.T) {
	item := WorkItem{Title: "Debug patient sync", Description: "Repro uses a real patient record pulled from prod."}
	d := Classify(item, DefaultPolicy())
	if d.Classification != DataPresent || d.Allowed {
		t.Fatalf("got %+v, want blocked DataPresent", d)
	}
}

func TestClassify_StructuredIdentifierPattern_Blocks(t *testing.T) {
	item := WorkItem{Title: "Fix export bug", Description: "The failing row has SSN 123-45-6789 in it."}
	d := Classify(item, DefaultPolicy())
	if d.Classification != DataPresent || d.Allowed {
		t.Fatalf("got %+v, want blocked DataPresent", d)
	}
}

func TestClassify_StructuredIdentifier_NeverLoggedInReason(t *testing.T) {
	item := WorkItem{Title: "x", Description: "123-45-6789"}
	d := Classify(item, DefaultPolicy())
	if strings.Contains(d.Reason, "123-45-6789") {
		t.Fatalf("reason leaked the matched identifier: %q", d.Reason)
	}
}

func TestClassify_StrongSignal_NeverLoggedInReason(t *testing.T) {
	item := WorkItem{Title: "x", Description: "DOB: 1990-01-01 for the affected record"}
	d := Classify(item, DefaultPolicy())
	if strings.Contains(strings.ToLower(d.Reason), "1990-01-01") {
		t.Fatalf("reason leaked sensitive content: %q", d.Reason)
	}
}

func TestClassify_SensitiveSystem_NoSanitizedEvidence_Blocks(t *testing.T) {
	item := WorkItem{Title: "Fix Open Dental adapter timeout", Description: "adapters/ehr/open_dental_client times out"}
	d := Classify(item, DefaultPolicy())
	if d.Classification != SystemRelated || d.Allowed {
		t.Fatalf("got %+v, want blocked SystemRelated", d)
	}
}

func TestClassify_SensitiveSystem_WithSanitizedEvidence_Allows(t *testing.T) {
	item := WorkItem{
		Title:       "Fix Open Dental adapter timeout",
		Description: "adapters/ehr/open_dental_client times out on synthetic fixture data",
		Labels:      []string{LabelSanitized},
	}
	d := Classify(item, DefaultPolicy())
	if d.Classification != SystemRelated || !d.Allowed {
		t.Fatalf("got %+v, want allowed SystemRelated", d)
	}
}

func TestClassify_SensitivePathIndicator_Blocks(t *testing.T) {
	item := WorkItem{Title: "Audit log query is slow", Description: "SELECT * FROM audit_logs WHERE ..."}
	d := Classify(item, DefaultPolicy())
	if d.Classification != SystemRelated || d.Allowed {
		t.Fatalf("got %+v, want blocked SystemRelated", d)
	}
}

func TestClassify_ExplicitNone_NoSignals_Allows(t *testing.T) {
	item := WorkItem{Title: "Bump lodash to 4.17.21", Labels: []string{LabelNone}}
	d := Classify(item, DefaultPolicy())
	if d.Classification != NonPHI || !d.Allowed {
		t.Fatalf("got %+v, want allowed NonPHI", d)
	}
}

func TestClassify_MetaDiscussionOfPHIPolicy_DoesNotFalsePositive(t *testing.T) {
	// Regression: caught live 2026-07-31 during Phase 4 smoke testing. A
	// bare "phi" keyword in the old default policy matched this exact
	// title and incorrectly classified routine governance/testing work
	// about the PHI guard itself as phi_system_related.
	item := WorkItem{Title: "TEST: PHI guard live smoke test — safe to delete"}
	d := Classify(item, DefaultPolicy())
	if d.Classification != NonPHI || !d.Allowed {
		t.Fatalf("got %+v, want allowed NonPHI — meta-discussion of PHI policy is not PHI-bearing work", d)
	}
}

func TestClassify_NoLabelsNoSignals_DefaultsToAllowedNonPHI(t *testing.T) {
	// The overwhelming majority of ordinary engineering beads carry no
	// phi:* label at all. This must not block, or the guard becomes
	// bureaucracy that halts all routine work.
	item := WorkItem{Title: "Add pagination to the dashboard table", Description: "Use the existing cursor pattern from the orders list."}
	d := Classify(item, DefaultPolicy())
	if d.Classification != NonPHI || !d.Allowed {
		t.Fatalf("got %+v, want allowed NonPHI (ordinary work must not require pre-labeling)", d)
	}
}

func TestClassify_UnrecognizedPHILabel_FailsClosed(t *testing.T) {
	item := WorkItem{Title: "x", Labels: []string{"phi:unclear-status"}}
	d := Classify(item, DefaultPolicy())
	if d.Classification != ClassificationRequired || d.Allowed {
		t.Fatalf("got %+v, want blocked ClassificationRequired", d)
	}
}

func TestClassify_MislabeledNone_StillCaughtBySignal(t *testing.T) {
	// Defense in depth: a manual phi:none label must not be a bypass if
	// content clearly signals real data. This is the direct test of
	// "do not rely solely on a manual phi:true label."
	item := WorkItem{
		Title:       "Cleanup task",
		Description: "This is definitely fine. SSN 123-45-6789 in the sample.",
		Labels:      []string{LabelNone},
	}
	d := Classify(item, DefaultPolicy())
	if d.Classification != DataPresent || d.Allowed {
		t.Fatalf("got %+v, want the content signal to override the phi:none label", d)
	}
}

func TestGuard_ReturnsErrorOnlyWhenBlocked(t *testing.T) {
	policy := DefaultPolicy()

	blocked := WorkItem{Title: "x", Labels: []string{LabelDataPresent}}
	if err := Guard(blocked, policy); err == nil {
		t.Fatal("expected Guard to block phi_data_present, got nil error")
	}

	allowed := WorkItem{Title: "Refactor the login form"}
	if err := Guard(allowed, policy); err != nil {
		t.Fatalf("expected Guard to allow ordinary work, got error: %v", err)
	}
}

func TestGuard_ErrorMessage_NeverContainsRawContent(t *testing.T) {
	secret := "SSN 987-65-4321 belongs to the patient"
	item := WorkItem{Title: "x", Description: secret}
	err := Guard(item, DefaultPolicy())
	if err == nil {
		t.Fatal("expected block")
	}
	if strings.Contains(err.Error(), "987-65-4321") {
		t.Fatalf("BlockedError leaked sensitive content: %v", err)
	}
}

func TestLoadPolicy_MissingFile_ReturnsDefault(t *testing.T) {
	p, err := LoadPolicy(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.SensitiveKeywords) == 0 {
		t.Fatal("expected default policy to have sensitive keywords")
	}
}

func TestLoadPolicy_MalformedFile_FailsClosed(t *testing.T) {
	dir := t.TempDir()
	settingsDir := dir + "/settings"
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(settingsDir+"/phi_policy.json", []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if _, err := LoadPolicy(dir); err == nil {
		t.Fatal("expected LoadPolicy to fail closed on malformed override, got nil error")
	}
}

func TestLoadPolicy_OverrideIsAdditive(t *testing.T) {
	dir := t.TempDir()
	settingsDir := dir + "/settings"
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	override := `{"sensitive_keywords": ["totally custom marker"]}`
	if err := os.WriteFile(settingsDir+"/phi_policy.json", []byte(override), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	p, err := LoadPolicy(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, k := range p.SensitiveKeywords {
		if k == "totally custom marker" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected override keyword to be present")
	}
	// Defaults must still be present — override is additive, not replacing.
	defaultFound := false
	for _, k := range p.SensitiveKeywords {
		if k == "open dental" {
			defaultFound = true
		}
	}
	if !defaultFound {
		t.Fatal("expected default keywords to remain after additive override")
	}
}
