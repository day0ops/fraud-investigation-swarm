import { getActors, submitAlerts } from "./api";
import { renderHeader, renderTiles } from "./swarm";

const app = document.querySelector<HTMLDivElement>("#app")!;
app.innerHTML = `
  <style>
    .bar{display:flex;gap:8px;align-items:center;padding:12px}
    button{background:#238636;color:#fff;border:0;border-radius:6px;padding:8px 12px;cursor:pointer}
    .hdr{font-size:20px;padding:0 12px 8px}
    .grid{display:flex;flex-wrap:wrap;gap:6px;padding:12px}
    .tile{width:120px;height:44px;border-radius:6px;display:flex;align-items:center;justify-content:center;font-size:11px;color:#0b0f1a;transition:opacity .4s}
  </style>
  <div class="bar">
    <button id="hero">Submit hero case</button>
    <button id="burst">Simulate fraud campaign</button>
  </div>
  <div id="head"></div>
  <div class="grid" id="grid"></div>
`;

document
  .querySelector("#hero")!
  .addEventListener("click", () =>
    submitAlerts({ mode: "single", alertId: "ALERT-1001" }),
  );
document
  .querySelector("#burst")!
  .addEventListener("click", () => submitAlerts({ mode: "burst", count: 10 }));

async function tick() {
  try {
    const v = await getActors();
    document.querySelector("#head")!.innerHTML = renderHeader(v);
    document.querySelector("#grid")!.innerHTML = renderTiles(v);
  } catch {
    document.querySelector("#head")!.innerHTML =
      `<div class="hdr">waiting for swarm...</div>`;
  }
}
setInterval(tick, 1000);
tick();
