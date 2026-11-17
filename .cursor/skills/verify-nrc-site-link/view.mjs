export const title = "How an NRC import links a reactor to a site";

const names = {
  "two-new-same": "Two new reactors, one plant",
  "join-existing": "New reactor, plant already stored",
  "rename-keep": "Stored reactor renamed",
  "rename-collision": "Rename hits another plant",
  "rename-then-join": "Rename, then the new reactor joins",
  "create-then-collide": "New reactor joins the renamed plant",
};

export const state = (v) => {
  if (v.phase === "ready") return "Ready";
  const name = names[v.scenario] ?? v.scenario;
  if (v.phase === "importing") return `${name}, writing`;
  if (v.phase === "committed") return `${name}, committed`;
  return `${name}, rolled back`;
};

export const step = (action) => {
  const labels = {
    Start: "start the import",
    Import: "rename stored plants, then link new reactors",
    Reject: "the name and location already belong to another plant",
  };
  return labels[action] ?? null;
};

export const group = (v) => {
  if (v.phase === "committed") return "Committed";
  if (v.phase === "aborted") return "Rolled back";
  if (v.phase === "importing") return "In the transaction";
  return "Ready";
};

export const colors = {
  Committed: "teal",
  "Rolled back": "orange",
  "In the transaction": "blue",
  Ready: "tan",
};
