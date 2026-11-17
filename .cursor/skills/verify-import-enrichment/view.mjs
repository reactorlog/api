export const title = "Which enrichment fields an NRC import writes";

const names = {
  reimport: "Stored reactor",
  join: "New reactor on a stored plant",
  create: "New plant",
};

export const state = (v) => {
  if (v.phase === "ready") return "Ready";
  const name = names[v.scenario] ?? v.scenario;
  if (v.phase === "importing") return `${name}, in the transaction`;
  if (v.scenario === "reimport" && v.reactor.design === "empty") return "Stored reactor, design cleared";
  return `${name}, committed`;
};

export const step = (action) => {
  const labels = {
    StartReimport: "open a stored reactor",
    StartJoin: "open a new reactor on a stored plant",
    StartCreate: "open a new plant",
    Reimport: "update the NRC fields",
    Join: "insert the reactor",
    Create: "insert the plant and the reactor",
  };
  return labels[action] ?? null;
};

export const group = (v) => {
  if (v.phase === "committed" && v.reactor.design === "empty" && v.scenario === "reimport") return "Design cleared";
  if (v.phase === "committed") return "Committed";
  if (v.phase === "importing") return "In the transaction";
  return "Ready";
};

export const colors = {
  Committed: "teal",
  "In the transaction": "blue",
  Ready: "tan",
  "Design cleared": "orange",
};
