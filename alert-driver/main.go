// Command alert-driver triggers fraud investigations by starting a
// lead-investigator run per alert. --alert runs one (the hero case); --burst
// runs a campaign at --rate per second. The invocation transport is the recipe
// confirmed live in the deployment plan (A2A message/send or kagent run API),
// selected by --endpoint; keep the flag and submit() in sync with that recipe.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	fx "github.com/day0ops/fraud-investigation-swarm/fixtures"
)

func submit(endpoint, alertID string) error {
	a, ok := fx.AlertByID(alertID)
	if !ok {
		return fmt.Errorf("unknown alert %s", alertID)
	}
	// Payload shape follows the recorded recipe: hand the lead investigator the
	// alert id and its trigger context as the initial message.
	body, _ := json.Marshal(map[string]any{
		"alertId":       a.ID,
		"customerId":    a.CustomerID,
		"accountId":     a.AccountID,
		"type":          a.Type,
		"triggerReason": a.TriggerReason,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("submit %s: status %d", alertID, resp.StatusCode)
	}
	log.Printf("submitted investigation for %s", alertID)
	return nil
}

func main() {
	endpoint := flag.String("endpoint", "", "lead-investigator invocation endpoint (from the deployment recipe)")
	alert := flag.String("alert", "", "single alert id to investigate (e.g. ALERT-1001)")
	burst := flag.Int("burst", 0, "number of investigations to fire as a campaign")
	rate := flag.Float64("rate", 2.0, "submissions per second for a burst")
	flag.Parse()

	if *endpoint == "" {
		log.Fatal("--endpoint is required")
	}
	var ids []string
	switch {
	case *alert != "":
		ids = selectSingle(*alert)
	case *burst > 0:
		ids = selectBurst(*burst)
	default:
		log.Fatal("set either --alert or --burst")
	}
	if ids == nil {
		log.Fatal("no alerts selected (unknown alert id?)")
	}
	if err := pace(ids, *rate, func(id string) error { return submit(*endpoint, id) }); err != nil {
		log.Fatalf("driver error: %v", err)
	}
	log.Printf("done: %d investigations submitted", len(ids))
}
