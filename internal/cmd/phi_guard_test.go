package cmd

import (
	"os"
	"testing"
)

func TestPhiGuardForBead_OrdinaryBead_Allowed(t *testing.T) {
	info := &beadInfo{
		Title:       "Add pagination to the dashboard table",
		Description: "Use the existing cursor pattern from the orders list.",
	}
	if err := phiGuardForBead("", info); err != nil {
		t.Fatalf("expected ordinary bead to pass the PHI guard, got: %v", err)
	}
}

func TestPhiGuardForBead_ExplicitDataPresentLabel_Blocked(t *testing.T) {
	info := &beadInfo{
		Title:  "Fix patient billing export",
		Labels: []string{"phi:data-present"},
	}
	err := phiGuardForBead("", info)
	if err == nil {
		t.Fatal("expected phi:data-present bead to be blocked")
	}
}

func TestPhiGuardForBead_SensitiveSystemNoEvidence_Blocked(t *testing.T) {
	info := &beadInfo{
		Title:       "Open Dental adapter retry logic",
		Description: "adapters/ehr/open_dental_client needs a backoff",
	}
	if err := phiGuardForBead("", info); err == nil {
		t.Fatal("expected phi_system_related bead with no sanitization evidence to be blocked")
	}
}

func TestPhiGuardForBead_SensitiveSystemWithEvidence_Allowed(t *testing.T) {
	info := &beadInfo{
		Title:       "Open Dental adapter retry logic",
		Description: "adapters/ehr/open_dental_client needs a backoff, tested against synthetic fixtures",
		Labels:      []string{"phi:sanitized"},
	}
	if err := phiGuardForBead("", info); err != nil {
		t.Fatalf("expected sanitized phi_system_related bead to pass, got: %v", err)
	}
}

func TestPhiGuardForBead_MalformedPolicyOverride_FailsClosed(t *testing.T) {
	dir := t.TempDir()
	settingsDir := dir + "/settings"
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(settingsDir+"/phi_policy.json", []byte("{broken"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	info := &beadInfo{Title: "Anything at all"}
	if err := phiGuardForBead(dir, info); err == nil {
		t.Fatal("expected a malformed policy override to fail closed (block), got nil error")
	}
}
