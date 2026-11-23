---
name: evidence-led-debugging
description: >-
  Investigate bugs, regressions, hangs, crashes, incorrect behavior, or performance
  failures when the root cause is uncertain. Use for unfamiliar code, flaky tests,
  and incidents where reading the code suggests plausible but unverified causes.
---

# evidence-led debugging

Establish what happened before explaining why. Source code describes possible behavior; a plausible explanation is not evidence that it occurred.

## procedure

1. **Capture the failure.** Record the exact symptom, expected behavior, command or request, environment, version/commit, and relevant timestamps. Preserve raw errors and diagnostic outputs. If the symptom is not observable, say so.
2. **Find a control.** Compare the broken case (variable) with a working case, last known good version, or a failure with a different signature. Keep inputs, versions, load, flags, and environment as similar as possible; identify remaining confounders. If no control exists, state that limitation.
3. **Keep facts separate from theories.** Maintain a small evidence ledger: `F` = directly observed fact (include command, log, trace, test, or source); `H` = hypothesis; `P` = prediction that distinguishes hypotheses. Treat evidence quality and instrumentation as fallible. Never describe inferred behavior as measured behavior.
4. **Choose discriminating observations.** Ask what measurement would be different if each hypothesis were true. Run the lowest-risk, highest-information test first. Measure both control and variable when possible. Prefer logs, traces, counters, process state, database state, network observations, and targeted instrumentation to repeated code-only speculation.
5. **Falsify aggressively.** Every surviving hypothesis must explain all relevant high-confidence facts. Explicitly reject a hypothesis when a predicted signal is absent or a fact contradicts it. If none fits, collect new signals from a different layer rather than generating a long list of guesses.
6. **Reproduce faithfully.** Prefer a failing test with the original bug's distinctive signals, not merely a matching error message. Document differences between reproduction and incident. If reproduction is impossible, preserve the observed constraints and state the resulting uncertainty.
7. **Verify causality and the fix.** Where feasible, change one causal variable, confirm the predicted signal changes, then apply the smallest justified fix. Re-run the original reproduction and an appropriate control. Add regression coverage for the violated invariant or resulting state, not just that a function returned without error.
8. **Report only warranted confidence.** Distinguish `verified root cause`, `probable cause`, and `unconfirmed hypothesis`. Include commands run and results, or explicitly say what could not be run. Do not claim tests passed without executing them.

## escalation and safety

- Read source to locate instrumentation points and identify invariants; do not use code inspection alone as proof of runtime behavior.
- When ordinary diagnostics are inconclusive, move to another observable layer (e.g. request logs → traces → process stacks → database/network state). Avoid unnecessary instrumentation if existing evidence suffices.
- For consequential or ambiguous fixes, ask an independent reviewer to find contradictions in the evidence and alternative causes, **not** to agree with the proposed patch.
- Never alter production data, enable invasive instrumentation, expose secrets, or execute destructive diagnostics without explicit authorization. Use read-only access and redact sensitive details by default.
- Scale the process to the bug: obvious compiler errors need a compiler result and verification, not an incident report.

## concise debugging report

Use these fields when the investigation is nontrivial; omit empty fields rather than filling them with guesses.

```text
symptom: <exact observed failure>
control: <working comparison, or unavailable>
facts: F1 <observation + where verified>; F2 ...
hypothesis: H1 <cause> predicts P1 <distinguishing signal>
test: <experiment + observed outcome>
conclusion: <verified | probable | unconfirmed, with reason>
fix: <change, if justified>
verification: <commands/tests run + results; remaining gaps>
```

Reference: Cindy Sridharan, *How to effectively debug when you no longer understand the code*, https://copyconstruct.medium.com/how-to-effectively-debug-when-you-no-longer-understand-the-code-bd3dcb2c5f7b
