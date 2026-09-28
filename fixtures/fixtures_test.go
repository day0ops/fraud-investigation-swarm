package fixtures

import "testing"

func TestCustomerByID(t *testing.T) {
	c, ok := CustomerByID("CUST-1001")
	if !ok {
		t.Fatal("CustomerByID(CUST-1001) not found")
	}
	if c.Name != "Elena Rossi" {
		t.Errorf("Name = %q; want Elena Rossi", c.Name)
	}
	if _, ok := CustomerByID("nope"); ok {
		t.Error("CustomerByID(nope) = found; want not found")
	}
}

func TestTransactionsByAccount(t *testing.T) {
	txs := TransactionsByAccount("ACC-1001")
	if len(txs) != 3 {
		t.Fatalf("got %d txs for ACC-1001; want 3", len(txs))
	}
}

func TestScreenSanctionsHit(t *testing.T) {
	hits := ScreenSanctions("Zarwan Holdings LLC")
	if len(hits) != 1 || hits[0].List != "OFAC-SDN" {
		t.Fatalf("got %+v; want one OFAC-SDN hit", hits)
	}
	if len(ScreenSanctions("Supermercato Centrale")) != 0 {
		t.Error("clean counterparty returned a sanctions hit")
	}
}

func TestMatchTypologies(t *testing.T) {
	got := MatchTypologies([]string{"new_device", "sanctioned_counterparty"})
	ids := map[string]bool{}
	for _, ty := range got {
		ids[ty.ID] = true
	}
	if !ids["TYP-ATO"] || !ids["TYP-SANCTIONS"] {
		t.Errorf("got %v; want both TYP-ATO and TYP-SANCTIONS", ids)
	}
}

func TestAllAlertsAndHero(t *testing.T) {
	if len(AllAlerts()) != 6 {
		t.Fatalf("got %d alerts; want 6", len(AllAlerts()))
	}
	a, ok := AlertByID("ALERT-1001")
	if !ok || !a.HeroCase {
		t.Errorf("ALERT-1001 hero = %v, ok = %v; want true/true", a.HeroCase, ok)
	}
}
