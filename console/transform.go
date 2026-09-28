package main

// terminalStates are actor states that no longer count as an active investigation.
var terminalStates = map[string]bool{"completed": true, "failed": true}

func SummarizeSwarm(actors []Actor) SwarmView {
	active := 0
	pods := map[string]bool{}
	for _, a := range actors {
		if !terminalStates[a.State] {
			active++
		}
		if a.WorkerPod != "" {
			pods[a.WorkerPod] = true
		}
	}
	return SwarmView{Actors: actors, ActiveInvestigations: active, WorkerPods: len(pods)}
}

func BuildCaseGraph(alertID string, spans []SpanNode) CaseGraph {
	return CaseGraph{AlertID: alertID, Nodes: spans}
}
