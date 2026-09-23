# Scripts

Small deterministic utilities for Core.

Scripts should be easy to inspect, run locally, and verify. Prefer simple
standard-library implementations until the project has a clear need for more
infrastructure.

## Project Sniffer

Run:

```bash
python3 _backoffice/scripts/sniff_projects.py --root ~/Projects --registry-dir _registry
```

Outputs:

- `_registry/repositories.json`;
- `_registry/project-groups.json`;
- `_registry/bootstrap-report.md`;
- `_registry/deletion-candidates.json`;
- `_registry/deletion-candidates.md`.

Repository technology data is split into two layers:

- `detected`: raw scanner output grouped as `languages`, `frameworks`,
  `runtimes`, `tooling`, plus evidence records with marker paths.
- `effective`: the controlled-vocabulary profile used by downstream systems
  such as MCP profile composition.

`repositories.json` also includes an `effective_repositories` array. Downstream
agent launch and MCP profile composition should read that view instead of
interpreting raw scanner evidence or legacy `stack` values independently.
Workspace generation merges that top-level effective view back onto raw
repository records before aggregating a workspace stack, so a space reflects
the effective technology tags from all of its repositories while still keeping
raw detection evidence available for audit.

The scanner prunes nested git repositories and common vendored/generated source
folders before classifying a parent repository. Optional deterministic
corrections can live in `_registry/technology-overrides.json`, keyed by
repository path/id/remote identity, or in a project card section named
`## Technology Overrides`.

## Work Workspaces

Run after the project sniffer:

```bash
python3 _backoffice/scripts/generate_workspaces.py --registry-dir _registry --work-dir Work
```
