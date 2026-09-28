package main

type Actor struct {
	Name     string `json:"name"`
	Template string `json:"template"`
	State    string `json:"state"`
}

type SwarmView struct {
	Actors               []Actor `json:"actors"`
	ActiveInvestigations int     `json:"activeInvestigations"`
	WorkerPods           int     `json:"workerPods"`
}

type SpanNode struct {
	ID        string `json:"id"`
	ParentID  string `json:"parentId"`
	Agent     string `json:"agent"`
	StartedAt string `json:"startedAt"`
}

type CaseGraph struct {
	AlertID string     `json:"alertId"`
	Nodes   []SpanNode `json:"nodes"`
}
