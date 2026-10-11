package provider

import "testing"

func TestClientRenewalConflict(t *testing.T) {
	t.Parallel()
	if got := clientRenewalConflict(0, 0, 0); got != "" {
		t.Fatalf("all disabled: %q", got)
	}
	if got := clientRenewalConflict(0, 0, 1); got != "" {
		t.Fatalf("weekday only: %q", got)
	}
	if got := clientRenewalConflict(7, 0, 1); got == "" {
		t.Fatal("expected conflict with reset")
	}
	if got := clientRenewalConflict(0, 15, 3); got == "" {
		t.Fatal("expected conflict with reset_day")
	}
	if got := clientRenewalConflict(0, 0, 8); got == "" {
		t.Fatal("expected range error")
	}
}
