---- MODULE EiaEnrichment ----
\* Which columns an EIA enrich writes, from enrich.sql.
\* Updates the site matched by name with plant enrichment when the tuple
\* differs. timezone, location, and reactors stay.
\* name is the match key and is not in the SET list.
\* address_line_2, phone, and email are also not written.
\* state is written: both NRC import and EIA enrich set it.

EXTENDS Naturals

CONSTANT Broken

\* enrichment abstracts the plant tuple written by enrich.sql:
\* eia_plant_code, address_line_1, city, state, zip, latitude, longitude.
Site(enrich, tz, loc) ==
  [enrichment |-> enrich, timezone |-> tz, location |-> loc]
Reactor(present, design) == [present |-> present, design |-> design]

EmptySite == Site("empty", "set", "set")
PriorSite == Site("prior", "set", "set")
IncomingSite == Site("incoming", "set", "set")
KeptReactor == Reactor(TRUE, "set")

VARIABLES phase, scenario, site, reactor, snapshotSite, snapshotReactor
vars == <<phase, scenario, site, reactor, snapshotSite, snapshotReactor>>

TypeOK ==
  /\ phase \in {"ready", "enriching", "committed"}
  /\ scenario \in {"", "fill", "replace", "skip", "miss"}
  /\ site.enrichment \in {"empty", "prior", "incoming"}
  /\ site.timezone \in {"empty", "set"}
  /\ site.location \in {"empty", "set"}
  /\ snapshotSite.enrichment \in {"empty", "prior", "incoming"}
  /\ snapshotSite.timezone \in {"empty", "set"}
  /\ snapshotSite.location \in {"empty", "set"}
  /\ reactor.present \in BOOLEAN
  /\ reactor.design \in {"empty", "set"}
  /\ snapshotReactor.present \in BOOLEAN
  /\ snapshotReactor.design \in {"empty", "set"}

Init ==
  /\ phase = "ready"
  /\ scenario = ""
  /\ site = EmptySite
  /\ reactor = KeptReactor
  /\ snapshotSite = EmptySite
  /\ snapshotReactor = KeptReactor

\* Empty enrichment, protected fields and design already set.
StartFill ==
  /\ phase = "ready"
  /\ phase' = "enriching"
  /\ scenario' = "fill"
  /\ site' = EmptySite
  /\ reactor' = KeptReactor
  /\ snapshotSite' = EmptySite
  /\ snapshotReactor' = KeptReactor

\* Prior plant data differs from the incoming plant.
StartReplace ==
  /\ phase = "ready"
  /\ phase' = "enriching"
  /\ scenario' = "replace"
  /\ site' = PriorSite
  /\ reactor' = KeptReactor
  /\ snapshotSite' = PriorSite
  /\ snapshotReactor' = KeptReactor

\* Incoming plant equals stored enrichment. is distinct from skips the row.
StartSkip ==
  /\ phase = "ready"
  /\ phase' = "enriching"
  /\ scenario' = "skip"
  /\ site' = IncomingSite
  /\ reactor' = KeptReactor
  /\ snapshotSite' = IncomingSite
  /\ snapshotReactor' = KeptReactor

\* Incoming site_name does not match the stored site name.
StartMiss ==
  /\ phase = "ready"
  /\ phase' = "enriching"
  /\ scenario' = "miss"
  /\ site' = PriorSite
  /\ reactor' = KeptReactor
  /\ snapshotSite' = PriorSite
  /\ snapshotReactor' = KeptReactor

\* Name match and distinct enrichment. Broken clears timezone.
Apply ==
  /\ phase = "enriching"
  /\ scenario \in {"fill", "replace"}
  /\ phase' = "committed"
  /\ site' = [
      enrichment |-> "incoming",
      timezone |-> IF Broken THEN "empty" ELSE site.timezone,
      location |-> site.location
    ]
  /\ UNCHANGED <<scenario, reactor, snapshotSite, snapshotReactor>>

\* Name match and identical enrichment. The UPDATE does not run.
Skip ==
  /\ phase = "enriching"
  /\ scenario = "skip"
  /\ phase' = "committed"
  /\ UNCHANGED <<scenario, site, reactor, snapshotSite, snapshotReactor>>

\* No name match. The UPDATE touches no row.
Miss ==
  /\ phase = "enriching"
  /\ scenario = "miss"
  /\ phase' = "committed"
  /\ UNCHANGED <<scenario, site, reactor, snapshotSite, snapshotReactor>>

Done ==
  /\ phase = "committed"
  /\ UNCHANGED vars

Next ==
  \/ StartFill
  \/ StartReplace
  \/ StartSkip
  \/ StartMiss
  \/ Apply
  \/ Skip
  \/ Miss
  \/ Done

Spec == Init /\ [][Next]_vars

\* After commit, plant enrichment is applied on a name match when values
\* differ, skipped when identical or unmatched, and timezone, location, and
\* reactors equal the pre-enrich snapshot.
EnrichmentAppliesAndProtects ==
  phase # "committed" \/
  /\ site.timezone = snapshotSite.timezone
  /\ site.location = snapshotSite.location
  /\ reactor = snapshotReactor
  /\ scenario = "fill" => site.enrichment = "incoming"
  /\ scenario = "replace" => site.enrichment = "incoming"
  /\ scenario = "skip" => site = snapshotSite
  /\ scenario = "miss" => site = snapshotSite
====
