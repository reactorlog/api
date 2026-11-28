export const title = "Which fields an EIA enrich writes";

const names = {
  fill: "Empty enrichment",
  replace: "Prior enrichment",
  skip: "Identical enrichment",
  miss: "Unmatched name",
};

export const state = (v) => {
  if (v.phase === "ready") return "Ready";
  const name = names[v.scenario] ?? v.scenario;
  if (v.phase === "enriching") return `${name}, in the transaction`;
  if (v.site.timezone === "empty") return `${name}, timezone cleared`;
  if (v.scenario === "fill" || v.scenario === "replace") {
    return `${name}, enrichment applied`;
  }
  return `${name}, unchanged`;
};

export const step = (action) => {
  const labels = {
    StartFill: "open a site with empty enrichment",
    StartReplace: "open a site with prior enrichment",
    StartSkip: "open a site whose enrichment already matches",
    StartMiss: "open a site whose name does not match",
    Apply: "update the plant enrichment columns",
    Skip: "skip an identical enrichment row",
    Miss: "match no site by name",
  };
  return labels[action] ?? null;
};

export const group = (v) => {
  if (v.phase === "committed" && v.site.timezone === "empty") return "Timezone cleared";
  if (v.phase === "committed") return "Committed";
  if (v.phase === "enriching") return "In the transaction";
  return "Ready";
};

export const colors = {
  Committed: "teal",
  "In the transaction": "blue",
  Ready: "tan",
  "Timezone cleared": "orange",
};
