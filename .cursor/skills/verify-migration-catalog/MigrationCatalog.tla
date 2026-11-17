---- MODULE MigrationCatalog ----
\* applyPending walks the catalog in order.
\* A version present in the migrations table is skipped and Result.Applied
\* is false. A missing version runs Up, then recordMigration, and
\* Result.Applied is true. Run never calls Down.
\* Commit atomicity of Up and the record is the separate verify-migrations check.

EXTENDS Naturals, FiniteSets

CONSTANT Broken

Versions == 1..2

VARIABLES phase, applied, initialApplied, index, executed, downCalled
vars == <<phase, applied, initialApplied, index, executed, downCalled>>

TypeOK ==
  /\ phase \in {"ready", "running", "done"}
  /\ applied \subseteq Versions
  /\ initialApplied \subseteq Versions
  /\ index \in 1..3
  /\ executed \subseteq Versions
  /\ downCalled \in BOOLEAN

Init ==
  /\ phase = "ready"
  /\ applied = {}
  /\ initialApplied = {}
  /\ index = 1
  /\ executed = {}
  /\ downCalled = FALSE

Start ==
  \E seed \in SUBSET Versions :
    /\ phase = "ready"
    /\ applied' = seed
    /\ initialApplied' = seed
    /\ phase' = "running"
    /\ index' = 1
    /\ executed' = {}
    /\ downCalled' = FALSE

\* Already recorded. Up is not called. Result.Applied is false.
Skip ==
  /\ phase = "running"
  /\ index \in Versions
  /\ index \in applied
  /\ index' = index + 1
  /\ UNCHANGED <<phase, applied, initialApplied, executed, downCalled>>

\* runMigration calls Up, then recordMigration.
Execute ==
  /\ ~Broken
  /\ phase = "running"
  /\ index \in Versions
  /\ index \notin applied
  /\ applied' = applied \cup {index}
  /\ executed' = executed \cup {index}
  /\ index' = index + 1
  /\ UNCHANGED <<phase, initialApplied, downCalled>>

\* Broken runMigration calls Down instead of Up.
ExecuteDown ==
  /\ Broken
  /\ phase = "running"
  /\ index \in Versions
  /\ index \notin applied
  /\ downCalled' = TRUE
  /\ applied' = applied \cup {index}
  /\ executed' = executed \cup {index}
  /\ index' = index + 1
  /\ UNCHANGED <<phase, initialApplied>>

Finish ==
  /\ phase = "running"
  /\ index = 3
  /\ phase' = "done"
  /\ UNCHANGED <<applied, initialApplied, index, executed, downCalled>>

Done ==
  /\ phase = "done"
  /\ UNCHANGED vars

Next ==
  \/ Start
  \/ Skip
  \/ Execute
  \/ ExecuteDown
  \/ Finish
  \/ Done

Spec == Init /\ [][Next]_vars

\* This call moves only missing versions to applied, by Up, in catalog order.
\* executed is the set of results whose Applied flag is true.
RunOnlyMovesForward ==
  /\ ~downCalled
  /\ phase = "done" =>
       /\ applied = Versions
       /\ executed = Versions \ initialApplied
====
