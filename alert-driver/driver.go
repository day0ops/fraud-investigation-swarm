package main

import (
	"time"

	fx "github.com/day0ops/fraud-investigation-swarm/fixtures"
)

func isKnownAlert(id string) bool {
	_, ok := fx.AlertByID(id)
	return ok
}

func selectSingle(id string) []string {
	if !isKnownAlert(id) {
		return nil
	}
	return []string{id}
}

// selectBurst returns n alert ids, cycling through the fixture alert set so a
// burst larger than the fixture count reuses alerts.
func selectBurst(n int) []string {
	all := fx.AllAlerts()
	if len(all) == 0 || n <= 0 {
		return nil
	}
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, all[i%len(all)].ID)
	}
	return out
}

// pace yields each id with a delay so a burst is submitted at ratePerSec.
func pace(ids []string, ratePerSec float64, fn func(string) error) error {
	var gap time.Duration
	if ratePerSec > 0 {
		gap = time.Duration(float64(time.Second) / ratePerSec)
	}
	for _, id := range ids {
		if err := fn(id); err != nil {
			return err
		}
		if gap > 0 {
			time.Sleep(gap)
		}
	}
	return nil
}
