import type { SwarmView } from "./api";

const stateColor: Record<string, string> = {
  queued: "#8b949e",
  running: "#2ea043",
  suspended: "#d29922",
  completed: "#1f6feb",
  failed: "#f85149",
};

export function renderHeader(v: SwarmView): string {
  return `<div class="hdr"><b>${v.activeInvestigations}</b> active investigations &middot; <b>${v.workerPods}</b> worker pods</div>`;
}

export function renderTiles(v: SwarmView): string {
  return v.actors
    .map(
      (a) =>
        `<div class="tile" title="${a.name} (${a.state})" style="background:${stateColor[a.state] || "#30363d"}">${a.template || a.name}</div>`,
    )
    .join("");
}
