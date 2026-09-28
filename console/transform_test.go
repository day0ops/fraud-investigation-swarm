package main

import "testing"

func TestSummarizeSwarm(t *testing.T) {
	actors := []Actor{
		{Name: "a1", State: "running", WorkerPod: "w1"},
		{Name: "a2", State: "running", WorkerPod: "w1"},
		{Name: "a3", State: "suspended", WorkerPod: "w2"},
		{Name: "a4", State: "completed", WorkerPod: ""},
	}
	v := SummarizeSwarm(actors)
	if v.ActiveInvestigations != 3 {
		t.Errorf("ActiveInvestigations = %d; want 3 (completed excluded)", v.ActiveInvestigations)
	}
	if v.WorkerPods != 2 {
		t.Errorf("WorkerPods = %d; want 2 distinct non-empty pods", v.WorkerPods)
	}
}

func TestBuildCaseGraph(t *testing.T) {
	spans := []SpanNode{
		{ID: "s1", ParentID: "", Agent: "fraud-lead-investigator"},
		{ID: "s2", ParentID: "s1", Agent: "tx-analyst"},
	}
	g := BuildCaseGraph("ALERT-1001", spans)
	if g.AlertID != "ALERT-1001" || len(g.Nodes) != 2 {
		t.Fatalf("got %+v; want 2 nodes for ALERT-1001", g)
	}
}
