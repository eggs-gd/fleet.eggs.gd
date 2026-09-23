#!/usr/bin/env python3
"""Generate human-editable Work workspaces from Core _registry files."""

from __future__ import annotations

import argparse
import json
import re
from pathlib import Path


def slugify(value: str) -> str:
    slug = re.sub(r"[^a-zA-Z0-9]+", "-", value).strip("-").lower()
    return slug or "project"


def load_json(path: Path) -> dict:
    return json.loads(path.read_text(encoding="utf-8"))


def load_repositories(registry_dir: Path) -> list[dict]:
    payload = load_json(registry_dir / "repositories.json")
    repositories = payload.get("repositories", [])
    effective_by_id = {
        repo["id"]: repo
        for repo in payload.get("effective_repositories", [])
        if repo.get("id")
    }
    effective_by_path = {
        repo["relative_path"]: repo
        for repo in payload.get("effective_repositories", [])
        if repo.get("relative_path")
    }

    merged = []
    for repo in repositories:
        effective_repo = effective_by_id.get(repo.get("id")) or effective_by_path.get(repo.get("relative_path")) or {}
        if effective_repo:
            combined = {**repo, **{key: value for key, value in effective_repo.items() if value not in (None, "", [])}}
            if repo.get("detected"):
                combined["detected"] = repo["detected"]
            merged.append(combined)
        else:
            merged.append(repo)

    if merged:
        return merged
    return payload.get("effective_repositories", [])


def repo_title(repo: dict) -> str:
    title = repo.get("title") or repo["name"]
    if repo["name"].lower().endswith(("_sonar", "-sonar")):
        return repo["name"]
    if title.lower() in {"sv", "how to use", "screenshots", "installation uri", "my project", "set up instructions"}:
        return repo["name"]
    return title


def display_project_title(project_id: str, repos: list[dict], kind: str) -> str:
    if kind == "workspace_group" and repos:
        return repos[0]["relative_path"].split("/")[0]
    return repo_title(repos[0])


def summarize_project(repos: list[dict], project_id: str, kind: str) -> str:
    if kind == "workspace_group" and repos:
        root = repos[0]["relative_path"].split("/")[0]
        return f"Workspace group containing {len(repos)} repositories under `{root}/`."
    useful = [
        repo
        for repo in repos
        if repo.get("summary_source") == "readme"
        and not repo["summary"].startswith("README present")
        and not repo["summary"].startswith("No README")
    ]
    if useful:
        primary = sorted(useful, key=lambda item: len(item["summary"]))[0]
        return primary["summary"]
    if len(repos) == 1:
        return repos[0].get("summary") or f"Project backed by {repos[0]['relative_path']}."
    return f"Project containing {len(repos)} repositories."


def collect_stack(repos: list[dict]) -> list[str]:
    stack = set()
    for repo in repos:
        effective = repo.get("effective", {})
        if effective:
            stack.update(effective.get("languages", []))
            stack.update(effective.get("frameworks", []))
            stack.update(effective.get("runtimes", []))
            stack.update(effective.get("tooling", []))
        else:
            stack.update(repo.get("stack", []))
    return sorted(stack)


def format_effective_profile(repo: dict) -> str:
    effective = repo.get("effective", {})
    if not effective:
        return ", ".join(repo.get("stack", [])) or "-"
    fields = []
    for label, key in (
        ("languages", "languages"),
        ("frameworks", "frameworks"),
        ("runtimes", "runtimes"),
        ("tooling", "tooling"),
        ("capabilities", "capabilities"),
    ):
        values = effective.get(key, [])
        if values:
            fields.append(f"{label}: {', '.join(values)}")
    return "<br>".join(fields) or "-"


def card_frontmatter(project: dict) -> str:
    lines = ["---"]
    for key in ["id", "title", "kind", "review_status", "status", "source"]:
        lines.append(f"{key}: {json.dumps(project[key], ensure_ascii=False)}")
    lines.append("repositories:")
    for repo in project["repositories"]:
        lines.append(f"  - {json.dumps(repo['relative_path'], ensure_ascii=False)}")
    lines.append("---")
    return "\n".join(lines)


def format_repo_table(repos: list[dict]) -> list[str]:
    lines = [
        "| Repository | Branch | Effective Profile | Remote | Summary |",
        "|---|---|---|---|---|",
    ]
    for repo in sorted(repos, key=lambda item: item["relative_path"].lower()):
        stack = format_effective_profile(repo)
        branch = repo.get("branch") or "-"
        remote = repo.get("remote") or "-"
        summary = (repo.get("summary") or "").replace("|", "\\|")
        lines.append(
            f"| `{repo['relative_path']}` | `{branch}` | {stack} | `{remote}` | {summary} |"
        )
    return lines


def format_relationships(project: dict) -> list[str]:
    relationships = project.get("relationships", [])
    if not relationships:
        return ["No related _registry candidates recorded."]
    lines = []
    for item in relationships:
        paths = ", ".join(f"`{path}`" for path in item["repository_paths"])
        lines.append(
            f"- {item['kind']} / {item['reason']} / {item['confidence']}: {paths}"
        )
    return lines


def format_protection_notes(project: dict) -> list[str]:
    notes = project.get("protection_notes", [])
    if not notes:
        return ["No protection rules recorded."]
    return [f"- {note}" for note in notes]


def format_technology_evidence(project: dict) -> list[str]:
    lines = []
    for repo in sorted(project["repositories"], key=lambda item: item["relative_path"].lower()):
        evidence = repo.get("detected", {}).get("evidence", [])
        if not evidence:
            continue
        lines.append(f"### `{repo['relative_path']}`")
        lines.append("")
        lines.append("| Path | Technologies | Detail | Status |")
        lines.append("|---|---|---|---|")
        for item in evidence:
            technologies = ", ".join(item.get("technologies", [])) or "-"
            detail = str(item.get("detail", "")).replace("|", "\\|")
            status = item.get("ignore_reason") if item.get("ignored") else "active"
            lines.append(f"| `{item.get('path', '-')}` | {technologies} | {detail} | {status} |")
        lines.append("")
    if not lines:
        return ["No technology evidence recorded."]
    return lines


def load_protection_rules(registry_dir: Path) -> dict:
    path = registry_dir / "protection-rules.json"
    if not path.is_file():
        return {"protected_prefixes": [], "protected_repository_paths": [], "protected_remote_groups": []}
    return load_json(path)


def protected_reason(path: str, rules: dict) -> str | None:
    for rule in rules.get("protected_prefixes", []):
        prefix = rule.get("prefix", "").rstrip("/")
        if path == prefix or path.startswith(f"{prefix}/"):
            return rule.get("reason")
    for rule in rules.get("protected_repository_paths", []):
        if path == rule.get("path"):
            return rule.get("reason")
    return None


def relationship_is_protected(repo_paths: list[str], rules: dict) -> bool:
    return bool(repo_paths) and all(protected_reason(path, rules) for path in repo_paths)


def build_workspaces(registry_dir: Path) -> list[dict]:
    repositories = load_repositories(registry_dir)
    groups = load_json(registry_dir / "project-groups.json")["groups"]
    protection_rules = load_protection_rules(registry_dir)
    repo_by_id = {repo["id"]: repo for repo in repositories}

    workspace_groups = [group for group in groups if group.get("kind") == "workspace_group"]
    covered_repo_ids = {repo_id for group in workspace_groups for repo_id in group["repository_ids"]}

    workspaces: list[dict] = []
    for group in workspace_groups:
        repos = [repo_by_id[repo_id] for repo_id in group["repository_ids"] if repo_id in repo_by_id]
        project_id = group["suggested_project_id"]
        workspaces.append(
            {
                "id": project_id,
                "title": display_project_title(project_id, repos, "workspace_group"),
                "kind": "workspace_group",
                "review_status": "draft",
                "status": "discovered",
                "source": group["reason"],
                "repositories": repos,
                "summary": summarize_project(repos, project_id, "workspace_group"),
                "stack": collect_stack(repos),
            }
        )

    for repo in repositories:
        if repo["id"] in covered_repo_ids:
            continue
        project_id = slugify(repo["name"])
        workspaces.append(
            {
                "id": project_id,
                "title": repo_title(repo),
                "kind": "standalone_repository",
                "review_status": "draft",
                "status": "discovered",
                "source": "single repository",
                "repositories": [repo],
                "summary": summarize_project([repo], project_id, "standalone_repository"),
                "stack": collect_stack([repo]),
            }
        )

    attach_protection_notes(workspaces, protection_rules)
    attach_relationships(workspaces, groups, repo_by_id, protection_rules)
    return sorted(workspaces, key=lambda item: item["id"])


def attach_protection_notes(projects: list[dict], rules: dict) -> None:
    for project in projects:
        notes = []
        seen = set()
        for repo in project["repositories"]:
            reason = protected_reason(repo["relative_path"], rules)
            if reason and reason not in seen:
                notes.append(reason)
                seen.add(reason)
        if notes:
            project["protection_notes"] = notes


def attach_relationships(projects: list[dict], groups: list[dict], repo_by_id: dict[str, dict], protection_rules: dict) -> None:
    project_by_repo_id = {}
    for project in projects:
        for repo in project["repositories"]:
            project_by_repo_id[repo["id"]] = project

    for group in groups:
        if group.get("kind") == "workspace_group":
            continue
        repo_ids = [repo_id for repo_id in group["repository_ids"] if repo_id in repo_by_id]
        project_ids = {project_by_repo_id[repo_id]["id"] for repo_id in repo_ids if repo_id in project_by_repo_id}
        relationship = {
            "kind": group.get("kind", "candidate"),
            "reason": group.get("reason", "unknown"),
            "confidence": group.get("confidence", "unknown"),
            "repository_paths": [repo_by_id[repo_id]["relative_path"] for repo_id in repo_ids],
        }
        if relationship_is_protected(relationship["repository_paths"], protection_rules):
            continue
        for project in projects:
            if project["id"] in project_ids:
                project.setdefault("relationships", []).append(relationship)


def render_workspace(project: dict) -> str:
    stack = ", ".join(project["stack"]) or "-"
    lines = [
        card_frontmatter(project),
        "",
        f"# {project['title']}",
        "",
        project["summary"],
        "",
        "## Registry",
        "",
        f"- Project id: `{project['id']}`",
        f"- Kind: `{project['kind']}`",
        f"- Review status: `{project['review_status']}`",
        f"- Status: `{project['status']}`",
        f"- Source: `{project['source']}`",
        f"- Stack: {stack}",
        "",
        "## Repositories",
        "",
        *format_repo_table(project["repositories"]),
        "",
        "## Relationships",
        "",
        *format_relationships(project),
        "",
        "## Protection",
        "",
        *format_protection_notes(project),
        "",
        "## Technology Evidence",
        "",
        *format_technology_evidence(project),
        "",
        "## Notes",
        "",
        "- Generated from _registry. Review and edit before treating this as canonical workspace metadata.",
    ]
    return "\n".join(lines).rstrip() + "\n"


def write_index(work_dir: Path, projects: list[dict]) -> None:
    lines = [
        "# Workspaces",
        "",
        "Human-editable Work workspaces generated from `_registry/`.",
        "",
        "| Workspace | Kind | Repositories | Status |",
        "|---|---|---:|---|",
    ]
    for project in projects:
        path = f"{project['id']}/PROJECT.md"
        lines.append(
            f"| [{project['title']}]({path}) | `{project['kind']}` | {len(project['repositories'])} | `{project['review_status']}` |"
        )
    work_dir.joinpath("INDEX.md").write_text("\n".join(lines) + "\n", encoding="utf-8")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Generate Work workspaces from Core _registry files.")
    parser.add_argument("--registry-dir", default="_registry", help="Registry directory.")
    parser.add_argument("--work-dir", default="Work", help="Work workspace output directory.")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    registry_dir = Path(args.registry_dir).resolve()
    work_dir = Path(args.work_dir).resolve()
    work_dir.mkdir(parents=True, exist_ok=True)

    projects = build_workspaces(registry_dir)
    for project in projects:
        project_dir = work_dir / project["id"]
        project_dir.mkdir(parents=True, exist_ok=True)
        for child in ("tasks", "notes", "decisions", "inbox"):
            project_dir.joinpath(child).mkdir(parents=True, exist_ok=True)
        project_dir.joinpath("PROJECT.md").write_text(render_workspace(project), encoding="utf-8")
    write_index(work_dir, projects)
    print(f"Generated {len(projects)} Work workspaces in {work_dir}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
