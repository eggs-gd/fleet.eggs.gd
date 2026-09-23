#!/usr/bin/env python3
"""Discover git repositories under a projects root and write Core _registry files."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import subprocess
from dataclasses import asdict, dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Iterable


DEFAULT_PRUNE_DIRS = {
    ".cache",
    ".mypy_cache",
    ".next",
    ".nuxt",
    ".pytest_cache",
    ".svelte-kit",
    ".turbo",
    ".venv",
    ".yarn",
    "__pycache__",
    "bin",
    "build",
    "dist",
    "deps",
    "external",
    "generated",
    "Library",
    "lib-src",
    "node_modules",
    "obj",
    "Temp",
    "target",
    "third_party",
    "vendor",
    "venv",
}

README_NAMES = ("README.md", "README.MD", "README.txt", "README")
PROTECTION_RULES_FILE = "protection-rules.json"
TECHNOLOGY_OVERRIDES_FILE = "technology-overrides.json"

DELETION_NAME_HINTS = {
    "backup",
    "copy",
    "demo",
    "old",
    "playground",
    "sonar",
    "test",
    "tests",
    "tmp",
}

TECHNOLOGY_CATALOG = {
    "csharp": {"category": "language", "capabilities": ["dotnet"]},
    "dart": {"category": "language", "capabilities": ["mobile"]},
    "go": {"category": "language", "capabilities": ["backend"]},
    "javascript": {"category": "language", "capabilities": ["frontend"]},
    "php": {"category": "language", "capabilities": ["backend"]},
    "python": {"category": "language", "capabilities": ["backend"]},
    "ruby": {"category": "language", "capabilities": ["backend"]},
    "rust": {"category": "language", "capabilities": ["backend"]},
    "typescript": {"category": "language", "capabilities": ["frontend"]},
    "flutter": {"category": "framework", "capabilities": ["mobile"]},
    "react": {"category": "framework", "capabilities": ["frontend"]},
    "svelte": {"category": "framework", "capabilities": ["frontend"]},
    "tauri": {"category": "framework", "capabilities": ["desktop"]},
    "unity": {"category": "framework", "capabilities": ["game-engine"]},
    "vite": {"category": "framework", "capabilities": ["frontend"]},
    "dotnet": {"category": "runtime", "capabilities": ["dotnet"]},
    "node": {"category": "runtime", "capabilities": ["frontend"]},
    "docker": {"category": "tooling", "capabilities": ["containers"]},
    "docker-compose": {"category": "tooling", "capabilities": ["containers"]},
    "npm": {"category": "tooling", "capabilities": ["frontend"]},
    "pnpm": {"category": "tooling", "capabilities": ["frontend"]},
    "yarn": {"category": "tooling", "capabilities": ["frontend"]},
}

STACK_MARKERS = {
    "package.json": ("node", "javascript"),
    "package-lock.json": ("npm",),
    "yarn.lock": ("yarn",),
    "pnpm-lock.yaml": ("pnpm",),
    "pyproject.toml": ("python",),
    "requirements.txt": ("python",),
    "go.mod": ("go",),
    "Cargo.toml": ("rust",),
    "Dockerfile": ("docker",),
    "docker-compose.yml": ("docker-compose",),
    "docker-compose.yaml": ("docker-compose",),
    "composer.json": ("php",),
    "Gemfile": ("ruby",),
    "pubspec.yaml": ("dart", "flutter"),
}


@dataclass
class Repository:
    id: str
    remote_identity: str | None
    name: str
    path: str
    relative_path: str
    remote: str | None
    branch: str | None
    head: str | None
    readme_path: str | None
    title: str
    summary: str
    summary_source: str
    detected: dict[str, Any]
    effective: dict[str, list[str]]
    stack: list[str]
    markers: list[str]
    parent_repository_id: str | None
    nested_repository_ids: list[str]
    first_seen_at: str
    last_seen_at: str


@dataclass
class TechnologyEvidence:
    kind: str
    path: str
    technologies: list[str]
    detail: str
    ignored: bool = False
    ignore_reason: str | None = None


def effective_repository_view(repo: Repository) -> dict[str, Any]:
    return {
        "id": repo.id,
        "remote_identity": repo.remote_identity,
        "name": repo.name,
        "path": repo.path,
        "relative_path": repo.relative_path,
        "remote": repo.remote,
        "branch": repo.branch,
        "summary": repo.summary,
        "effective": repo.effective,
        "evidence": repo.detected.get("evidence", []),
        "parent_repository_id": repo.parent_repository_id,
        "nested_repository_ids": repo.nested_repository_ids,
    }


def run_git(repo: Path, args: list[str]) -> str | None:
    try:
        result = subprocess.run(
            ["git", "-C", str(repo), *args],
            check=False,
            capture_output=True,
            text=True,
        )
    except FileNotFoundError:
        return None
    if result.returncode != 0:
        return None
    value = result.stdout.strip()
    return value or None


def utc_now() -> str:
    return datetime.now(timezone.utc).replace(microsecond=0).isoformat()


def normalize_remote(remote: str) -> str:
    value = remote.strip()
    value = re.sub(r"^ssh://git@", "git@", value)
    value = re.sub(r"^https://github.com/", "github.com:", value)
    value = re.sub(r"^git@github.com:", "github.com:", value)
    value = value.removesuffix(".git")
    return value.lower()


def repo_id(path: Path, remote: str | None) -> str:
    remote_part = normalize_remote(remote) if remote else "no-remote"
    basis = f"repo:{remote_part}|path:{path.resolve()}"
    digest = hashlib.sha1(basis.encode("utf-8")).hexdigest()[:12]
    slug_basis = remote_part.split("/")[-1] if remote else path.name
    slug = slugify(slug_basis)
    return f"{slug}-{digest}"


def remote_identity(remote: str | None) -> str | None:
    if not remote:
        return None
    basis = f"remote:{normalize_remote(remote)}"
    digest = hashlib.sha1(basis.encode("utf-8")).hexdigest()[:12]
    slug_basis = normalize_remote(remote).split("/")[-1]
    return f"{slugify(slug_basis)}-{digest}"


def previous_repo_id(path: Path, remote: str | None) -> str:
    if remote:
        basis = f"remote:{normalize_remote(remote)}"
    else:
        basis = f"path:{path.resolve()}"
    digest = hashlib.sha1(basis.encode("utf-8")).hexdigest()[:12]
    slug_basis = normalize_remote(remote).split("/")[-1] if remote else path.name
    slug = slugify(slug_basis)
    return f"{slug}-{digest}"


def slugify(value: str) -> str:
    slug = re.sub(r"[^a-zA-Z0-9]+", "-", value).strip("-").lower()
    return slug or "repo"


def discover_git_repos(root: Path, prune_dirs: set[str]) -> list[Path]:
    repos: list[Path] = []
    for current, dirs, files in os.walk(root):
        path = Path(current)
        dirs[:] = [d for d in dirs if d not in prune_dirs]
        if ".git" in dirs or ".git" in files:
            repos.append(path)
            dirs[:] = [d for d in dirs if d != ".git"]
    return sorted(repos, key=lambda item: str(item).lower())


def nested_repo_roots(repo: Path, all_repos: Iterable[Path]) -> set[Path]:
    roots: set[Path] = set()
    for candidate in all_repos:
        if candidate == repo:
            continue
        try:
            candidate.relative_to(repo)
        except ValueError:
            continue
        roots.add(candidate)
    return roots


def find_readme(repo: Path) -> Path | None:
    for name in README_NAMES:
        candidate = repo / name
        if candidate.is_file():
            return candidate
    return None


def read_text(path: Path, max_chars: int = 12000) -> str:
    try:
        return path.read_text(encoding="utf-8", errors="replace")[:max_chars]
    except OSError:
        return ""


def summarize_readme(repo: Path) -> tuple[str, str, str, str | None]:
    readme = find_readme(repo)
    if not readme:
        return repo.name, f"No README found. Repository at {repo.name}.", "fallback", None

    text = read_text(readme)
    lines = [line.strip() for line in text.splitlines()]
    title = repo.name
    for line in lines:
        if line.startswith("#"):
            title = clean_markdown(line.lstrip("#").strip()) or repo.name
            break

    summary = ""
    in_code = False
    for line in lines:
        if line.startswith("```"):
            in_code = not in_code
            continue
        if in_code or not line or line.startswith("#") or line.startswith("!") or line.startswith("[!"):
            continue
        if line.startswith(("-", "*", "|", "<", "[<")):
            continue
        summary = clean_markdown(line)
        break

    if not summary:
        summary = f"README present for {title}, but no short prose summary was detected."
        source = "readme-title"
    else:
        source = "readme"
    return title, summary, source, str(readme)


def clean_markdown(value: str) -> str:
    value = re.sub(r"<[^>]+>", " ", value)
    value = re.sub(r"!\[[^\]]*\]\([^)]+\)", " ", value)
    value = re.sub(r"\[([^\]]+)\]\([^)]+\)", r"\1", value)
    value = re.sub(r"`([^`]+)`", r"\1", value)
    value = re.sub(r"\s+", " ", value)
    return value.strip(" #*-|")


def normalize_technology_values(values: Iterable[str]) -> list[str]:
    normalized = []
    seen = set()
    for value in values:
        tech = str(value).strip().lower()
        if not tech or tech not in TECHNOLOGY_CATALOG or tech in seen:
            continue
        normalized.append(tech)
        seen.add(tech)
    return sorted(normalized)


def categorize_technologies(technologies: Iterable[str]) -> dict[str, list[str]]:
    profile = {"languages": [], "frameworks": [], "runtimes": [], "tooling": []}
    buckets = {
        "language": "languages",
        "framework": "frameworks",
        "runtime": "runtimes",
        "tooling": "tooling",
    }
    grouped: dict[str, set[str]] = {field: set() for field in profile}
    for tech in normalize_technology_values(technologies):
        field = buckets.get(TECHNOLOGY_CATALOG[tech]["category"])
        if field:
            grouped[field].add(tech)
    return {field: sorted(values) for field, values in grouped.items()}


def marker_ignore_reason(marker_path: str, ignore_rules: list[dict]) -> str | None:
    marker = Path(marker_path)
    parts = set(marker.parts[:-1])
    noisy = parts & DEFAULT_PRUNE_DIRS
    if noisy:
        return f"ignored source directory: {sorted(noisy)[0]}"
    for rule in ignore_rules:
        prefix = str(rule.get("path_prefix", "")).strip().strip("/")
        if prefix and (marker_path == prefix or marker_path.startswith(f"{prefix}/")):
            return rule.get("reason") or f"ignored by rule {prefix}"
    return None


def add_evidence(
    evidence: list[TechnologyEvidence],
    *,
    kind: str,
    marker_path: str,
    technologies: Iterable[str],
    detail: str,
    ignore_rules: list[dict],
) -> None:
    techs = normalize_technology_values(technologies)
    if not techs:
        return
    reason = marker_ignore_reason(marker_path, ignore_rules)
    evidence.append(
        TechnologyEvidence(
            kind=kind,
            path=marker_path,
            technologies=techs,
            detail=detail,
            ignored=bool(reason),
            ignore_reason=reason,
        )
    )


def detect_technologies(
    repo: Path,
    child_repositories: set[Path] | None = None,
    ignore_rules: list[dict] | None = None,
    max_depth: int = 3,
) -> dict[str, Any]:
    child_repositories = child_repositories or set()
    ignore_rules = ignore_rules or []
    evidence: list[TechnologyEvidence] = []
    for current, dirs, files in os.walk(repo):
        current_path = Path(current)
        rel = Path(current).relative_to(repo)
        depth = len(rel.parts)
        dirs[:] = [
            d
            for d in dirs
            if d not in DEFAULT_PRUNE_DIRS
            and d != ".git"
            and (current_path / d) not in child_repositories
        ]
        if depth >= max_depth:
            dirs[:] = []
        for file_name in files:
            if file_name in STACK_MARKERS:
                marker_path = str((current_path / file_name).relative_to(repo))
                add_evidence(
                    evidence,
                    kind="marker",
                    marker_path=marker_path,
                    technologies=STACK_MARKERS[file_name],
                    detail=f"{file_name} marker",
                    ignore_rules=ignore_rules,
                )
            elif file_name.endswith(".csproj") or file_name.endswith(".sln"):
                marker_path = str((current_path / file_name).relative_to(repo))
                add_evidence(
                    evidence,
                    kind="marker",
                    marker_path=marker_path,
                    technologies=("csharp", "unity" if is_unity_project(repo) else "dotnet"),
                    detail=f"{file_name} marker",
                    ignore_rules=ignore_rules,
                )

    package_json = repo / "package.json"
    if package_json.is_file():
        text = read_text(package_json)
        lower = text.lower()
        package_technologies = []
        if "svelte" in lower:
            package_technologies.append("svelte")
        if "react" in lower:
            package_technologies.append("react")
        if "vite" in lower:
            package_technologies.append("vite")
        if "tauri" in lower:
            package_technologies.append("tauri")
        if "typescript" in lower:
            package_technologies.append("typescript")
        add_evidence(
            evidence,
            kind="package-manifest",
            marker_path="package.json",
            technologies=package_technologies,
            detail="package.json dependency/name scan",
            ignore_rules=ignore_rules,
        )

    if is_unity_project(repo):
        add_evidence(
            evidence,
            kind="layout",
            marker_path=".",
            technologies=("unity",),
            detail="Assets and ProjectSettings directories",
            ignore_rules=ignore_rules,
        )

    active_technologies = {
        tech
        for item in evidence
        if not item.ignored
        for tech in item.technologies
    }
    detected = categorize_technologies(active_technologies)
    detected["evidence"] = [asdict(item) for item in sorted(evidence, key=lambda item: item.path)]
    return detected


def detect_stack(repo: Path, max_depth: int = 3) -> tuple[list[str], list[str]]:
    detected = detect_technologies(repo, max_depth=max_depth)
    stack = sorted(
        {
            *detected.get("languages", []),
            *detected.get("frameworks", []),
            *detected.get("runtimes", []),
            *detected.get("tooling", []),
        }
    )
    markers = sorted({item["path"] for item in detected.get("evidence", []) if not item.get("ignored")})
    return stack, markers


def is_unity_project(repo: Path) -> bool:
    return (repo / "Assets").is_dir() and (repo / "ProjectSettings").is_dir()


def parent_map(repos: Iterable[Path], ids_by_path: dict[Path, str]) -> dict[Path, Path | None]:
    ordered = sorted(repos, key=lambda item: len(item.parts))
    parents: dict[Path, Path | None] = {}
    for repo in ordered:
        parent = None
        for candidate in ordered:
            if candidate == repo:
                continue
            try:
                repo.relative_to(candidate)
            except ValueError:
                continue
            if parent is None or len(candidate.parts) > len(parent.parts):
                parent = candidate
        parents[repo] = parent if parent in ids_by_path else None
    return parents


def load_previous_repositories(registry_dir: Path) -> dict[str, dict]:
    path = registry_dir / "repositories.json"
    if not path.is_file():
        return {}
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return {}
    return {item["id"]: item for item in data.get("repositories", []) if "id" in item}


def load_protection_rules(registry_dir: Path) -> dict:
    path = registry_dir / PROTECTION_RULES_FILE
    if not path.is_file():
        return {"protected_prefixes": [], "protected_repository_paths": [], "protected_remote_groups": []}
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return {"protected_prefixes": [], "protected_repository_paths": [], "protected_remote_groups": []}


def load_technology_overrides(registry_dir: Path) -> dict:
    path = registry_dir / TECHNOLOGY_OVERRIDES_FILE
    if not path.is_file():
        return {"repositories": {}, "projects": {}}
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return {"repositories": {}, "projects": {}}
    return {
        "repositories": data.get("repositories", {}),
        "projects": data.get("projects", {}),
    }


def parse_project_card_overrides(root: Path) -> dict[str, dict]:
    work_dir = root / "Work"
    if not work_dir.is_dir():
        work_dir = root / "core.eggs.gd" / "Work"
    if not work_dir.is_dir():
        return {}
    overrides: dict[str, dict] = {}
    for project_card in sorted(work_dir.glob("*/PROJECT.md")):
        text = read_text(project_card, max_chars=20000)
        if not text.startswith("---\n"):
            continue
        end = text.find("\n---", 4)
        if end < 0:
            continue
        frontmatter = text[4:end]
        project_id = None
        repositories: list[str] = []
        in_repositories = False
        for line in frontmatter.splitlines():
            stripped = line.strip()
            if stripped.startswith("id:"):
                project_id = stripped.split(":", 1)[1].strip().strip('"')
            elif stripped == "repositories:":
                in_repositories = True
            elif in_repositories and stripped.startswith("-"):
                repositories.append(stripped[1:].strip().strip('"'))
            elif stripped and not line.startswith(" "):
                in_repositories = False
        marker = "\n## Technology Overrides\n"
        if marker not in text:
            continue
        section = text.split(marker, 1)[1].split("\n## ", 1)[0]
        override = parse_override_section(section)
        if not override:
            continue
        override["project_id"] = project_id or project_card.parent.name
        for relative_path in repositories:
            overrides[relative_path] = override
    return overrides


def parse_override_section(section: str) -> dict:
    override: dict[str, Any] = {}
    current_list = None
    list_aliases = {
        "add_languages": "add",
        "add_frameworks": "add",
        "add_runtimes": "add",
        "add_tooling": "add",
        "remove_languages": "remove",
        "remove_frameworks": "remove",
        "remove_runtimes": "remove",
        "remove_tooling": "remove",
        "capabilities": "capabilities",
    }
    for line in section.splitlines():
        stripped = line.strip()
        if not stripped:
            continue
        key_match = re.match(r"^[-*]?\s*([a-z_]+):\s*(.*)$", stripped)
        if key_match:
            key, value = key_match.groups()
            current_list = key if key in list_aliases else None
            if current_list and value:
                override.setdefault(list_aliases[current_list], []).extend(parse_inline_values(value))
            elif key == "reason" and value:
                override["reason"] = value.strip().strip('"')
            continue
        if current_list and stripped.startswith("-"):
            override.setdefault(list_aliases[current_list], []).extend(parse_inline_values(stripped[1:]))
    for key in ("add", "remove", "capabilities"):
        if key in override:
            override[key] = normalize_technology_values(override[key]) if key != "capabilities" else sorted(set(override[key]))
    return override


def parse_inline_values(value: str) -> list[str]:
    value = value.strip().strip("[]")
    return [item.strip().strip('"').strip("'") for item in re.split(r"[, ]+", value) if item.strip().strip(",")]


def repository_override(repo: Repository | None, relative_path: str, overrides: dict) -> dict:
    rules = overrides.get("repositories", {})
    candidates = [relative_path]
    if repo:
        candidates.extend([repo.id, repo.remote_identity or ""])
    for key in candidates:
        if key and key in rules:
            return rules[key]
    return {}


def project_override_for(relative_path: str, project_card_overrides: dict, registry_overrides: dict) -> dict:
    card_override = project_card_overrides.get(relative_path, {})
    project_id = card_override.get("project_id")
    registry_project_override = registry_overrides.get("projects", {}).get(project_id, {}) if project_id else {}
    merged = {
        "add": [
            *registry_project_override.get("add", []),
            *card_override.get("add", []),
        ],
        "remove": [
            *registry_project_override.get("remove", []),
            *card_override.get("remove", []),
        ],
        "capabilities": [
            *registry_project_override.get("capabilities", []),
            *card_override.get("capabilities", []),
        ],
    }
    return {key: value for key, value in merged.items() if value}


def compose_effective_profile(
    detected: dict[str, Any],
    *,
    repo_override: dict | None = None,
    project_override: dict | None = None,
    protection_reason: str | None = None,
) -> dict[str, list[str]]:
    technologies = {
        *detected.get("languages", []),
        *detected.get("frameworks", []),
        *detected.get("runtimes", []),
        *detected.get("tooling", []),
    }
    for override in (project_override or {}, repo_override or {}):
        technologies.difference_update(normalize_technology_values(override.get("remove", [])))
        technologies.update(normalize_technology_values(override.get("add", [])))

    effective = categorize_technologies(technologies)
    capabilities = {
        capability
        for tech in technologies
        for capability in TECHNOLOGY_CATALOG.get(tech, {}).get("capabilities", [])
    }
    for override in (project_override or {}, repo_override or {}):
        capabilities.update(str(item).strip().lower() for item in override.get("capabilities", []) if str(item).strip())
    if protection_reason:
        capabilities.add("protected-layout")
    effective["capabilities"] = sorted(capabilities)
    return effective


def is_protected_repository(repo: Repository, rules: dict) -> tuple[bool, str | None]:
    relative_path = repo.relative_path
    for prefix_rule in rules.get("protected_prefixes", []):
        prefix = prefix_rule.get("prefix", "").rstrip("/")
        if relative_path == prefix or relative_path.startswith(f"{prefix}/"):
            return True, prefix_rule.get("reason")
    for path_rule in rules.get("protected_repository_paths", []):
        if relative_path == path_rule.get("path"):
            return True, path_rule.get("reason")
    return False, None


def is_protected_remote_group(items: list[Repository], rules: dict) -> tuple[bool, str | None]:
    paths = sorted(item.relative_path for item in items)
    for group_rule in rules.get("protected_remote_groups", []):
        rule_paths = sorted(group_rule.get("paths", []))
        if paths == rule_paths:
            return True, group_rule.get("reason")
    return False, None


def build_repositories(root: Path, _registry_dir: Path) -> list[Repository]:
    seen_at = utc_now()
    previous = load_previous_repositories(_registry_dir)
    protection_rules = load_protection_rules(_registry_dir)
    technology_overrides = load_technology_overrides(_registry_dir)
    project_overrides = parse_project_card_overrides(root)
    repo_paths = discover_git_repos(root, DEFAULT_PRUNE_DIRS)
    ids_by_path: dict[Path, str] = {}
    remotes_by_path: dict[Path, str | None] = {}

    for path in repo_paths:
        remote = run_git(path, ["remote", "get-url", "origin"])
        remotes_by_path[path] = remote
        ids_by_path[path] = repo_id(path, remote)

    parents = parent_map(repo_paths, ids_by_path)
    nested_by_parent: dict[str, list[str]] = {repo_id_value: [] for repo_id_value in ids_by_path.values()}
    for path, parent in parents.items():
        if parent:
            nested_by_parent[ids_by_path[parent]].append(ids_by_path[path])

    repositories: list[Repository] = []
    for path in repo_paths:
        remote = remotes_by_path[path]
        rid = ids_by_path[path]
        title, summary, summary_source, readme_path = summarize_readme(path)
        previous_entry = previous.get(rid, previous.get(previous_repo_id(path, remote), {}))
        parent = parents[path]
        relative_path = str(path.relative_to(root))
        detected = detect_technologies(
            path,
            child_repositories=nested_repo_roots(path, repo_paths),
            ignore_rules=technology_overrides.get("ignore_paths", []),
        )
        repo = Repository(
            id=rid,
            remote_identity=remote_identity(remote),
            name=path.name,
            path=str(path),
            relative_path=relative_path,
            remote=remote,
            branch=run_git(path, ["branch", "--show-current"]),
            head=run_git(path, ["rev-parse", "--short", "HEAD"]),
            readme_path=readme_path,
            title=title,
            summary=summary,
            summary_source=summary_source,
            detected=detected,
            effective={},
            stack=[],
            markers=[],
            parent_repository_id=ids_by_path[parent] if parent else None,
            nested_repository_ids=sorted(nested_by_parent.get(rid, [])),
            first_seen_at=previous_entry.get("first_seen_at", seen_at),
            last_seen_at=seen_at,
        )
        protected, protected_reason = is_protected_repository(repo, protection_rules)
        repo.effective = compose_effective_profile(
            detected,
            repo_override=repository_override(repo, relative_path, technology_overrides),
            project_override=project_override_for(relative_path, project_overrides, technology_overrides),
            protection_reason=protected_reason if protected else None,
        )
        repo.stack = sorted(
            {
                *repo.effective.get("languages", []),
                *repo.effective.get("frameworks", []),
                *repo.effective.get("runtimes", []),
                *repo.effective.get("tooling", []),
            }
        )
        repo.markers = sorted(
            {
                item["path"]
                for item in detected.get("evidence", [])
                if not item.get("ignored")
            }
        )
        repositories.append(
            repo
        )
    return repositories


def comparable_name(name: str) -> str:
    value = name.lower()
    value = value.removesuffix(".git")
    parts = [part for part in re.split(r"[^a-z0-9]+", value) if part]
    suffixes = {
        "backup",
        "copy",
        "demo",
        "dev",
        "frontend",
        "old",
        "playground",
        "site",
        "sonar",
        "test",
        "tests",
    }
    while len(parts) > 1 and parts[-1] in suffixes:
        parts.pop()
    return "-".join(parts)


def suggest_groups(repositories: list[Repository], root: Path) -> list[dict]:
    groups: list[dict] = []

    by_top_folder: dict[str, list[Repository]] = {}
    for repo in repositories:
        first = Path(repo.relative_path).parts[0]
        by_top_folder.setdefault(first, []).append(repo)
    for folder, items in sorted(by_top_folder.items()):
        if len(items) > 1:
            groups.append(
                {
                    "id": f"parent-{slugify(folder)}",
                    "kind": "workspace_group",
                    "confidence": "high",
                    "decision": "treat_as_group",
                    "reason": "same top-level folder",
                    "suggested_project_id": slugify(folder),
                    "repository_ids": sorted(item.id for item in items),
                }
            )

    by_remote: dict[str, list[Repository]] = {}
    for repo in repositories:
        if repo.remote_identity:
            by_remote.setdefault(repo.remote_identity, []).append(repo)
    for remote_id, items in sorted(by_remote.items()):
        if len(items) > 1:
            groups.append(
                {
                    "id": f"remote-{remote_id}",
                    "kind": "duplicate_candidate",
                    "confidence": "high",
                    "decision": "review",
                    "reason": "same git remote",
                    "suggested_project_id": slugify(items[0].name),
                    "repository_ids": sorted(item.id for item in items),
                }
            )

    by_comparable_name: dict[str, list[Repository]] = {}
    for repo in repositories:
        key = comparable_name(repo.name)
        if len(key) >= 4:
            by_comparable_name.setdefault(key, []).append(repo)

    for key, related in sorted(by_comparable_name.items()):
        if len(related) > 1:
            groups.append(
                {
                    "id": f"name-{slugify(key)}",
                    "kind": "related_candidate",
                    "confidence": "medium",
                    "decision": "review",
                    "reason": "same base repository name",
                    "suggested_project_id": slugify(key),
                    "repository_ids": sorted({item.id for item in related}),
                }
            )

    nested = [repo for repo in repositories if repo.nested_repository_ids]
    for repo in nested:
        groups.append(
            {
                "id": f"nested-{repo.id}",
                "kind": "nested_group",
                "confidence": "medium",
                "decision": "review",
                "reason": "nested repositories",
                "suggested_project_id": slugify(repo.name),
                "repository_ids": sorted([repo.id, *repo.nested_repository_ids]),
            }
        )

    unique: dict[tuple[str, tuple[str, ...]], dict] = {}
    for group in groups:
        ids = tuple(group["repository_ids"])
        if len(ids) < 2:
            continue
        unique[(group["reason"], ids)] = group
    priority = {
        "workspace_group": 0,
        "duplicate_candidate": 1,
        "nested_group": 2,
        "related_candidate": 3,
    }
    return sorted(unique.values(), key=lambda item: (priority.get(item["kind"], 9), item["suggested_project_id"]))


def repo_status(repo: Repository) -> str | None:
    return run_git(Path(repo.path), ["status", "--short"])


def deletion_candidate_score(reasons: list[str]) -> str:
    strong = {"duplicate local checkout", "nested repository"}
    medium = {"temporary name hint", "no git remote"}
    if any(reason in strong for reason in reasons):
        return "high"
    if any(reason in medium for reason in reasons) and len(reasons) >= 2:
        return "medium"
    return "low"


def suggest_deletion_candidates(repositories: list[Repository], protection_rules: dict) -> list[dict]:
    candidates: dict[str, dict] = {}

    def add(repo: Repository, reason: str, detail: str) -> None:
        protected, protected_reason = is_protected_repository(repo, protection_rules)
        if protected:
            return
        status = repo_status(repo)
        entry = candidates.setdefault(
            repo.id,
            {
                "repository_id": repo.id,
                "name": repo.name,
                "path": repo.path,
                "relative_path": repo.relative_path,
                "remote": repo.remote,
                "branch": repo.branch,
                "head": repo.head,
                "decision": "review",
                "reasons": [],
                "details": [],
                "git_status": "dirty" if status else "clean",
            },
        )
        if reason not in entry["reasons"]:
            entry["reasons"].append(reason)
        if detail not in entry["details"]:
            entry["details"].append(detail)

    by_remote: dict[str, list[Repository]] = {}
    for repo in repositories:
        if repo.remote_identity:
            by_remote.setdefault(repo.remote_identity, []).append(repo)
    for items in by_remote.values():
        if len(items) > 1:
            protected, _ = is_protected_remote_group(items, protection_rules)
            if protected:
                continue
            sorted_items = sorted(items, key=lambda item: (len(Path(item.relative_path).parts), item.relative_path))
            keeper = sorted_items[0]
            for repo in sorted_items[1:]:
                add(
                    repo,
                    "duplicate local checkout",
                    f"Shares remote with {keeper.relative_path}; review whether this checkout is still needed.",
                )

    repo_by_id = {repo.id: repo for repo in repositories}
    for repo in repositories:
        if repo.parent_repository_id:
            parent = repo_by_id.get(repo.parent_repository_id)
            parent_path = parent.relative_path if parent else repo.parent_repository_id
            add(
                repo,
                "nested repository",
                f"Repository is nested inside {parent_path}; review whether it is vendored/submodule/intentional.",
            )

        tokens = {part for part in re.split(r"[^a-z0-9]+", repo.name.lower()) if part}
        hints = sorted(tokens & DELETION_NAME_HINTS)
        if hints:
            add(repo, "temporary name hint", f"Name contains cleanup hint(s): {', '.join(hints)}.")

        if not repo.remote:
            add(repo, "no git remote", "Repository has no origin remote.")

        if not repo.readme_path:
            add(repo, "no README", "Repository has no README detected at its root.")

    output = []
    for entry in candidates.values():
        entry["confidence"] = deletion_candidate_score(entry["reasons"])
        output.append(entry)

    priority = {"high": 0, "medium": 1, "low": 2}
    return sorted(output, key=lambda item: (priority[item["confidence"]], item["relative_path"]))


def write_deletion_report(path: Path, candidates: list[dict], generated_at: str) -> None:
    lines = [
        "# Deletion Candidates",
        "",
        f"Generated at: `{generated_at}`",
        "",
        "These are review candidates only. Core never deletes repositories from this report.",
        "",
        f"Candidates: **{len(candidates)}**",
        "",
    ]
    if not candidates:
        lines.append("No deletion candidates detected.")
    else:
        lines.extend(["| Confidence | Path | Git | Reasons | Details |", "|---|---|---|---|---|"])
        for candidate in candidates:
            reasons = ", ".join(candidate["reasons"])
            details = "<br>".join(candidate["details"]).replace("|", "\\|")
            lines.append(
                "| {confidence} | `{path}` | {git_status} | {reasons} | {details} |".format(
                    confidence=candidate["confidence"],
                    path=candidate["relative_path"],
                    git_status=candidate["git_status"],
                    reasons=reasons,
                    details=details,
                )
            )
    path.write_text("\n".join(lines).rstrip() + "\n", encoding="utf-8")


def write_json(path: Path, payload: dict) -> None:
    path.write_text(json.dumps(payload, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")


def write_report(
    path: Path,
    root: Path,
    repositories: list[Repository],
    groups: list[dict],
    deletion_candidates: list[dict],
    generated_at: str,
) -> None:
    lines = [
        "# Bootstrap Report",
        "",
        f"Generated at: `{generated_at}`",
        f"Projects root: `{root}`",
        "",
        f"Discovered repositories: **{len(repositories)}**",
        f"Groups and candidates: **{len(groups)}**",
        f"Deletion review candidates: **{len(deletion_candidates)}**",
        "",
        "## Repositories",
        "",
        "| Name | Path | Branch | Stack | Summary |",
        "|---|---|---|---|---|",
    ]
    for repo in repositories:
        stack = ", ".join(repo.stack) if repo.stack else "-"
        branch = repo.branch or "-"
        summary = repo.summary.replace("|", "\\|")
        lines.append(f"| {repo.title} | `{repo.relative_path}` | `{branch}` | {stack} | {summary} |")

    lines.extend(["", "## Groups And Candidates", ""])
    if not groups:
        lines.append("No grouping suggestions detected.")
    else:
        repo_labels = {repo.id: f"{repo.name} ({repo.relative_path})" for repo in repositories}
        for group in groups:
            names = ", ".join(f"`{repo_labels.get(repo_id, repo_id)}`" for repo_id in group["repository_ids"])
            lines.extend(
                [
                    f"### {group['suggested_project_id']}",
                    "",
                    f"- Kind: {group['kind']}",
                    f"- Confidence: {group['confidence']}",
                    f"- Decision: {group['decision']}",
                    f"- Reason: {group['reason']}",
                    f"- Repositories: {names}",
                    "",
                ]
            )

    path.write_text("\n".join(lines).rstrip() + "\n", encoding="utf-8")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Discover git repositories and write Core _registry files.")
    parser.add_argument("--root", default="~/Projects", help="Projects root to scan.")
    parser.add_argument("--registry-dir", default="_registry", help="Output _registry directory.")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    root = Path(args.root).expanduser().resolve()
    _registry_dir = Path(args.registry_dir).resolve()
    _registry_dir.mkdir(parents=True, exist_ok=True)

    generated_at = utc_now()
    repositories = build_repositories(root, _registry_dir)
    groups = suggest_groups(repositories, root)
    protection_rules = load_protection_rules(_registry_dir)
    deletion_candidates = suggest_deletion_candidates(repositories, protection_rules)

    write_json(
        _registry_dir / "repositories.json",
        {
            "generated_at": generated_at,
            "root": str(root),
            "repositories": [asdict(repo) for repo in repositories],
            "effective_repositories": [effective_repository_view(repo) for repo in repositories],
        },
    )
    write_json(
        _registry_dir / "project-groups.json",
        {
            "generated_at": generated_at,
            "root": str(root),
            "groups": groups,
        },
    )
    write_json(
        _registry_dir / "deletion-candidates.json",
        {
            "generated_at": generated_at,
            "root": str(root),
            "candidates": deletion_candidates,
        },
    )
    write_report(_registry_dir / "bootstrap-report.md", root, repositories, groups, deletion_candidates, generated_at)
    write_deletion_report(_registry_dir / "deletion-candidates.md", deletion_candidates, generated_at)
    print(f"Discovered {len(repositories)} repositories under {root}")
    print(f"Found {len(deletion_candidates)} deletion review candidates")
    print(f"Wrote _registry to {_registry_dir}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
