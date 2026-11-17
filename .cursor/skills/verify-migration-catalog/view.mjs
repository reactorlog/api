export const title = "How a migration run walks the catalog";

export const state = (v) => {
  if (v.phase === "ready") return "Ready";
  if (v.phase === "done") return v.downCalled ? "Down ran" : "Catalog applied";
  if (v.index > 2) return "Both versions visited";
  const already = Array.isArray(v.initialApplied) && v.initialApplied.includes(v.index);
  return already ? `Version ${v.index} already applied` : `Version ${v.index} pending`;
};

export const step = (action) => {
  const labels = {
    Start: "start the run",
    Skip: "skip it",
    Execute: "apply it",
    ExecuteDown: "run Down",
    Finish: "finish the catalog",
  };
  return labels[action] ?? null;
};

export const group = (v) => {
  if (v.downCalled) return "Down ran";
  if (v.phase === "done") return "Applied";
  if (v.phase === "running") return "Walking the catalog";
  return "Ready";
};

export const colors = {
  Applied: "teal",
  "Walking the catalog": "blue",
  Ready: "tan",
  "Down ran": "orange",
};
