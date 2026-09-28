package main

import "testing"

func TestSelectBurstCyclesAlerts(t *testing.T) {
	ids := selectBurst(4)
	if len(ids) != 4 {
		t.Fatalf("got %d ids; want 4", len(ids))
	}
	// all ids must be real fixture alerts
	for _, id := range ids {
		if !isKnownAlert(id) {
			t.Errorf("selectBurst produced unknown alert %q", id)
		}
	}
}

func TestSelectSingleHero(t *testing.T) {
	ids := selectSingle("ALERT-1001")
	if len(ids) != 1 || ids[0] != "ALERT-1001" {
		t.Fatalf("got %v; want [ALERT-1001]", ids)
	}
}

func TestSelectSingleUnknown(t *testing.T) {
	if ids := selectSingle("nope"); ids != nil {
		t.Fatalf("got %v; want nil for unknown alert", ids)
	}
}
