package main

// terminalStates are actor states that no longer count as an active investigation.
var terminalStates = map[string]bool{"completed": true, "failed": true}

// SummarizeSwarm counts active (non-terminal) actors. workerPods is supplied
// by the caller (from ActorSource.WorkerPoolSize) since actor records carry
// no worker-pod field to derive it from.
func SummarizeSwarm(actors []Actor, workerPods int) SwarmView {
	active := 0
	for _, a := range actors {
		if !terminalStates[a.State] {
			active++
		}
	}
	return SwarmView{Actors: actors, ActiveInvestigations: active, WorkerPods: workerPods}
}

func BuildCaseGraph(alertID string, spans []SpanNode) CaseGraph {
	return CaseGraph{AlertID: alertID, Nodes: spans}
}
