---
name: go-crap
description: >-
  Scan this repository with the go-crap commit gate and fix every function
  whose CRAP score is 6 or higher. Use when the user asks to run go-crap,
  lower CRAP scores, or clear the functions that would block a commit.
disable-model-invocation: true
---

# go-crap

Run the same scan the commit hook uses, then fix every function it lists. Create a commit only when the user asked for one.

## Scan

From the repository root:

```bash
python3 .cursor/scripts/go-crap-report.py
```

The script prints one JSON object. Exit 0 means every function scores strictly under 6. A missing score is a failure. The score is `effective_crap`, unless that value is 0, in which case the score is `crap`.

When `ok` is true, say that every function scores under 6 and stop.

When the process exits 2 or 3, report `detail` and stop. Leave the code unchanged after a timeout, a missing `go-crap` binary, or a failed scan.

## Fix only listed functions

Edit only functions in `offenders`. Each object has `file`, `line`, `function`, `score`, `cyclomatic`, `coverage`, and `missing_score`.

Fully covered CRAP equals cyclomatic complexity, so tests can bring a score under 6 only when complexity is 5 or less:

| Complexity | Tests can pass when coverage is above |
| --- | --- |
| 1 | always under 6 |
| 2 | above 0% |
| 3 | about 31% |
| 4 | 50% |
| 5 | about 66% |
| 6 or more | never; split first |

For each offender, highest score first:

1. Complexity is missing or 6 or more: split the function until each resulting function has complexity of 5 or less. Preserve behavior and the existing exported API. Then cover the extracted branches with tests.
2. Complexity is 5 or less: add or extend tests so the uncovered branches of that function run. Match the test style already used in that package. Put them in the package's `*_test.go` file. Scan again before splitting.
3. If the same function is still listed after those tests, split it, then test the resulting functions.

`missing_score` means the report has no CRAP value. Read the function and apply the same complexity rule.

## Repeat

Run the script after each batch of edits. A failing test run comes back as exit 3 with the test output in `detail`; fix those tests, then scan again. Stop when the script exits 0.
