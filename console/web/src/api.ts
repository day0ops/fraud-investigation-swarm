export interface Actor {
  name: string;
  template: string;
  state: string;
  workerPod: string;
}
export interface SwarmView {
  actors: Actor[];
  activeInvestigations: number;
  workerPods: number;
}
export interface SpanNode {
  id: string;
  parentId: string;
  agent: string;
  startedAt: string;
}
export interface CaseGraph {
  alertId: string;
  nodes: SpanNode[];
}

export async function getActors(): Promise<SwarmView> {
  const r = await fetch("/api/actors");
  if (!r.ok) throw new Error(`actors: ${r.status}`);
  return r.json();
}

export async function getCaseGraph(alertId: string): Promise<CaseGraph> {
  const r = await fetch(`/api/case/${encodeURIComponent(alertId)}/spans`);
  if (!r.ok) throw new Error(`case: ${r.status}`);
  return r.json();
}

export async function submitAlerts(body: {
  mode: "single" | "burst";
  alertId?: string;
  count?: number;
}): Promise<void> {
  const r = await fetch("/api/alerts", {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!r.ok) throw new Error(`alerts: ${r.status}`);
}
