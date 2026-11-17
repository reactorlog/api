# Project stage

This project is pre-release. Prefer direct refactors and the simplest coherent API.
Do not preserve old APIs with compatibility aliases, forwarding wrappers, or
deprecations unless the user explicitly requests compatibility. Update callers
and tests together when an API changes.

# Agent rules

| Rule | Enforcement |
| --- | --- |
| SQL keywords are lowercase (`select`, not `SELECT`) in `.sql` files and in SQL string literals in `.go` files | `python3 .cursor/scripts/sql-keyword-case.py`; commit hook `.cursor/hooks/sql-case-before-commit.sh` |
