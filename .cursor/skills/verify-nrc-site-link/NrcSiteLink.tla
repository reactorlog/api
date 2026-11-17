---- MODULE NrcSiteLink ----
\* One NRC import statement.
\* Stored dockets rename their linked site before any new site is inserted.
\* A new docket then joins the site with its name and location, or creates one.
\* List order does not change the result: a new docket listed before a rename
\* still joins the renamed site.
\* Name and location are one key. sites_name_location rejects a second site
\* with that pair. Two stored sites renamed onto one key roll back.
\* The reactor upsert keeps an existing site_id and does not write design.
\* A unique failure aborts the statement; the transaction rolls back.

EXTENDS Naturals, Sequences, FiniteSets

CONSTANT Broken

Sites == {"s1", "s2"}
Dockets == {"d1", "d2"}
Identities == {"a", "b"}

ABSENT == [present |-> FALSE, site |-> "s1", design |-> "empty"]
FREE == [used |-> FALSE, identity |-> "a"]
Reactor(site, design) == [present |-> TRUE, site |-> site, design |-> design]
Used(identity) == [used |-> TRUE, identity |-> identity]

DB(r1, r2, s1, s2) ==
  [reactors |-> [d \in Dockets |-> IF d = "d1" THEN r1 ELSE r2],
   sites |-> [s \in Sites |-> IF s = "s1" THEN s1 ELSE s2]]

Blank == DB(ABSENT, ABSENT, FREE, FREE)
Row(docket, identity) == [docket |-> docket, identity |-> identity]

twoNewSame == [name |-> "two-new-same", mustCommit |-> TRUE, db |-> Blank,
  batch |-> <<Row("d1", "a"), Row("d2", "a")>>]
joinExisting == [name |-> "join-existing", mustCommit |-> TRUE,
  db |-> DB(ABSENT, ABSENT, Used("a"), FREE),
  batch |-> <<Row("d1", "a")>>]
renameKeep == [name |-> "rename-keep", mustCommit |-> TRUE,
  db |-> DB(Reactor("s1", "kept"), Reactor("s1", "kept"), Used("a"), FREE),
  batch |-> <<Row("d1", "b")>>]
renameCollision == [name |-> "rename-collision", mustCommit |-> FALSE,
  db |-> DB(Reactor("s1", "kept"), ABSENT, Used("a"), Used("b")),
  batch |-> <<Row("d1", "b")>>]
renameThenJoin == [name |-> "rename-then-join", mustCommit |-> TRUE,
  db |-> DB(Reactor("s1", "kept"), ABSENT, Used("a"), FREE),
  batch |-> <<Row("d1", "b"), Row("d2", "b")>>]
\* Same rows as rename-then-join, opposite list order. Renames are applied
\* first, so the new docket joins the renamed site and the import commits.
createThenCollide == [name |-> "create-then-collide", mustCommit |-> TRUE,
  db |-> DB(Reactor("s1", "kept"), ABSENT, Used("a"), FREE),
  batch |-> <<Row("d2", "b"), Row("d1", "b")>>]

Scenarios == {twoNewSame, joinExisting, renameKeep, renameCollision,
              renameThenJoin, createThenCollide}

VARIABLES phase, db, snapshot, batch, mustCommit, scenario
vars == <<phase, db, snapshot, batch, mustCommit, scenario>>

TypeOK ==
  /\ phase \in {"ready", "importing", "committed", "aborted"}
  /\ scenario \in {"", "two-new-same", "join-existing", "rename-keep",
                   "rename-collision", "rename-then-join", "create-then-collide"}
  /\ db.reactors \in [Dockets -> [present: BOOLEAN, site: Sites, design: {"empty", "kept"}]]
  /\ db.sites \in [Sites -> [used: BOOLEAN, identity: Identities]]
  /\ snapshot.reactors \in [Dockets -> [present: BOOLEAN, site: Sites, design: {"empty", "kept"}]]
  /\ snapshot.sites \in [Sites -> [used: BOOLEAN, identity: Identities]]
  /\ batch \in Seq([docket: Dockets, identity: Identities])
  /\ mustCommit \in BOOLEAN

Init ==
  /\ phase = "ready"
  /\ db = Blank
  /\ snapshot = Blank
  /\ batch = <<>>
  /\ mustCommit = FALSE
  /\ scenario = ""

SiteWith(database, identity) ==
  {s \in Sites : database.sites[s].used /\ database.sites[s].identity = identity}

Range(seq) == {seq[i] : i \in DOMAIN seq}

Stored(database, rows) ==
  {row \in Range(rows) : database.reactors[row.docket].present}

\* A stored site would take a name and location another site already has,
\* or two stored sites would take the same name and location.
RenameConflict(database, rows) ==
  \/ \E row \in Stored(database, rows) :
       \E other \in SiteWith(database, row.identity) :
         other # database.reactors[row.docket].site
  \/ \E r1, r2 \in Stored(database, rows) :
       /\ database.reactors[r1.docket].site # database.reactors[r2.docket].site
       /\ r1.identity = r2.identity

AfterRename(database, rows) ==
  [database EXCEPT !.sites =
    [s \in Sites |->
      IF \E row \in Stored(database, rows) : database.reactors[row.docket].site = s
      THEN [used |-> TRUE,
            identity |-> (CHOOSE row \in Stored(database, rows) :
                           database.reactors[row.docket].site = s).identity]
      ELSE database.sites[s]]]

\* Place new dockets against the renamed sites. Broken skips the name match,
\* so a taken key tries to insert a second site and the statement fails.
RECURSIVE Place(_, _, _)
Place(database, rows, i) ==
  IF i > Len(rows) THEN [ok |-> TRUE, db |-> database]
  ELSE IF database.reactors[rows[i].docket].present
       THEN Place(database, rows, i + 1)
       ELSE LET occupied == SiteWith(database, rows[i].identity)
                free == {s \in Sites : ~database.sites[s].used}
            IN IF occupied # {} /\ ~Broken
               THEN LET chosen == CHOOSE s \in occupied : TRUE
                        next == [database EXCEPT !.reactors[rows[i].docket] =
                          [present |-> TRUE, site |-> chosen, design |-> "empty"]]
                    IN Place(next, rows, i + 1)
               ELSE IF occupied = {} /\ free # {}
                    THEN LET chosen == CHOOSE s \in free : TRUE
                             next == [database EXCEPT
                               !.sites[chosen] = [used |-> TRUE, identity |-> rows[i].identity],
                               !.reactors[rows[i].docket] =
                                 [present |-> TRUE, site |-> chosen, design |-> "empty"]]
                         IN Place(next, rows, i + 1)
                    ELSE [ok |-> FALSE, db |-> database]

Start ==
  \E chosen \in Scenarios :
    /\ phase = "ready"
    /\ db' = chosen.db
    /\ snapshot' = chosen.db
    /\ batch' = chosen.batch
    /\ mustCommit' = chosen.mustCommit
    /\ scenario' = chosen.name
    /\ phase' = "importing"

Import ==
  /\ phase = "importing"
  /\ ~RenameConflict(db, batch)
  /\ LET placed == Place(AfterRename(db, batch), batch, 1)
     IN /\ placed.ok
        /\ db' = placed.db
        /\ phase' = "committed"
        /\ UNCHANGED <<snapshot, batch, mustCommit, scenario>>

Reject ==
  /\ phase = "importing"
  /\ \/ RenameConflict(db, batch)
     \/ ~Place(IF RenameConflict(db, batch) THEN db ELSE AfterRename(db, batch), batch, 1).ok
  /\ phase' = "aborted"
  /\ db' = snapshot
  /\ UNCHANGED <<snapshot, batch, mustCommit, scenario>>

Done ==
  /\ phase \in {"committed", "aborted"}
  /\ UNCHANGED vars

Next ==
  \/ Start
  \/ Import
  \/ Reject
  \/ Done

Spec == Init /\ [][Next]_vars

UniqueIdentities ==
  \A s, t \in Sites :
    s # t /\ db.sites[s].used /\ db.sites[t].used =>
      db.sites[s].identity # db.sites[t].identity

\* A stored reactor keeps its site and its design. A new docket is stored.
Preserved ==
  /\ \A d \in Dockets :
       snapshot.reactors[d].present =>
         /\ db.reactors[d].present
         /\ db.reactors[d].site = snapshot.reactors[d].site
         /\ db.reactors[d].design = snapshot.reactors[d].design
  /\ \A i \in DOMAIN batch :
       ~snapshot.reactors[batch[i].docket].present =>
         db.reactors[batch[i].docket].present

\* A batch this statement can commit does commit, with each stored reactor
\* still on its site. A batch it rejects leaves the snapshot.
NewDocketJoins ==
  /\ phase = "aborted" => ~mustCommit /\ db = snapshot
  /\ phase = "committed" =>
       mustCommit /\ Preserved /\ UniqueIdentities
====
