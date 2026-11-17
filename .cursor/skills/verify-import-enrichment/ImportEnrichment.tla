---- MODULE ImportEnrichment ----
\* Which columns an NRC import writes, from import.sql.
\* A stored docket updates site name, location, and state, and reactor name,
\* unit, and type. design, timezone, coordinates, and address stay.
\* A new docket that joins a site does not update that site.
\* A new site insert writes name, location, and state, so the other site
\* columns stay null. A new reactor insert omits design.
\* coordinates "half" is one of latitude or longitude set. The schema allows
\* it. This check requires import to leave that pair alone, not to forbid it.

EXTENDS Naturals

CONSTANT Broken

Site(tz, coords, addr) == [timezone |-> tz, coordinates |-> coords, address |-> addr]
Reactor(present, design) == [present |-> present, design |-> design]

EnrichedSite == Site("set", "half", "set")
EnrichedReactor == Reactor(TRUE, "set")
EmptySite == Site("empty", "empty", "empty")
AbsentReactor == Reactor(FALSE, "empty")

VARIABLES phase, scenario, site, reactor, snapshotSite, snapshotReactor, fresh
vars == <<phase, scenario, site, reactor, snapshotSite, snapshotReactor, fresh>>

TypeOK ==
  /\ phase \in {"ready", "importing", "committed"}
  /\ scenario \in {"", "reimport", "join", "create"}
  /\ site.timezone \in {"empty", "set"}
  /\ site.coordinates \in {"empty", "pair", "half"}
  /\ site.address \in {"empty", "set"}
  /\ snapshotSite.timezone \in {"empty", "set"}
  /\ snapshotSite.coordinates \in {"empty", "pair", "half"}
  /\ snapshotSite.address \in {"empty", "set"}
  /\ reactor.present \in BOOLEAN
  /\ reactor.design \in {"empty", "set"}
  /\ snapshotReactor.present \in BOOLEAN
  /\ snapshotReactor.design \in {"empty", "set"}
  /\ fresh \in {"none", "empty"}

Init ==
  /\ phase = "ready"
  /\ scenario = ""
  /\ site = EmptySite
  /\ reactor = AbsentReactor
  /\ snapshotSite = EmptySite
  /\ snapshotReactor = AbsentReactor
  /\ fresh = "none"

StartReimport ==
  /\ phase = "ready"
  /\ phase' = "importing"
  /\ scenario' = "reimport"
  /\ site' = EnrichedSite
  /\ reactor' = EnrichedReactor
  /\ snapshotSite' = EnrichedSite
  /\ snapshotReactor' = EnrichedReactor
  /\ fresh' = "none"

StartJoin ==
  /\ phase = "ready"
  /\ phase' = "importing"
  /\ scenario' = "join"
  /\ site' = EnrichedSite
  /\ reactor' = AbsentReactor
  /\ snapshotSite' = EnrichedSite
  /\ snapshotReactor' = AbsentReactor
  /\ fresh' = "none"

StartCreate ==
  /\ phase = "ready"
  /\ phase' = "importing"
  /\ scenario' = "create"
  /\ site' = EmptySite
  /\ reactor' = AbsentReactor
  /\ snapshotSite' = EmptySite
  /\ snapshotReactor' = AbsentReactor
  /\ fresh' = "none"

\* Docket hit. Broken clears design, which the upsert does not do.
Reimport ==
  /\ phase = "importing"
  /\ scenario = "reimport"
  /\ phase' = "committed"
  /\ \/ /\ Broken
        /\ reactor' = [reactor EXCEPT !.design = "empty"]
     \/ /\ ~Broken
        /\ UNCHANGED reactor
  /\ UNCHANGED <<scenario, site, snapshotSite, snapshotReactor, fresh>>

\* Name and location hit. The site row is not updated. The new reactor has no design.
Join ==
  /\ phase = "importing"
  /\ scenario = "join"
  /\ phase' = "committed"
  /\ fresh' = "empty"
  /\ UNCHANGED <<scenario, site, reactor, snapshotSite, snapshotReactor>>

\* Neither row exists. The inserted site and reactor leave enrichment null.
Create ==
  /\ phase = "importing"
  /\ scenario = "create"
  /\ phase' = "committed"
  /\ fresh' = "empty"
  /\ UNCHANGED <<scenario, site, reactor, snapshotSite, snapshotReactor>>

Done ==
  /\ phase = "committed"
  /\ UNCHANGED vars

Next ==
  \/ StartReimport
  \/ StartJoin
  \/ StartCreate
  \/ Reimport
  \/ Join
  \/ Create
  \/ Done

Spec == Init /\ [][Next]_vars

\* Rows that existed before the import keep design, timezone, coordinates,
\* and address. A reactor this import inserts has no design.
EnrichmentSurvives ==
  phase # "committed" \/
  /\ site = snapshotSite
  /\ reactor = snapshotReactor
  /\ scenario \in {"join", "create"} => fresh = "empty"
====
