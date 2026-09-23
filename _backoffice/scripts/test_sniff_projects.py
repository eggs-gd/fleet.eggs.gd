#!/usr/bin/env python3

from pathlib import Path
from tempfile import TemporaryDirectory
import sys
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parent))

import sniff_projects
import generate_workspaces


class TechnologyProfileTests(unittest.TestCase):
    def test_nested_repository_markers_do_not_leak_to_parent(self) -> None:
        with TemporaryDirectory() as temp:
            root = Path(temp)
            parent = root / "Audiophile" / "jivemax"
            child = parent / "jivelite"
            parent.joinpath(".git").mkdir(parents=True)
            child.joinpath(".git").mkdir(parents=True)
            child.joinpath("jivelite.sln").write_text("Microsoft Visual Studio Solution File\n", encoding="utf-8")

            repositories = sniff_projects.build_repositories(root, root / "_registry")
            by_relative = {repo.relative_path: repo for repo in repositories}

            self.assertIn("Audiophile/jivemax", by_relative)
            self.assertIn("Audiophile/jivemax/jivelite", by_relative)
            self.assertEqual(by_relative["Audiophile/jivemax"].effective["languages"], [])
            self.assertEqual(by_relative["Audiophile/jivemax"].effective["runtimes"], [])
            self.assertEqual(by_relative["Audiophile/jivemax/jivelite"].effective["languages"], ["csharp"])
            self.assertEqual(by_relative["Audiophile/jivemax/jivelite"].effective["runtimes"], ["dotnet"])

    def test_effective_profile_composition_applies_overrides(self) -> None:
        detected = {
            "languages": ["javascript"],
            "frameworks": ["react"],
            "runtimes": ["node"],
            "tooling": ["npm"],
            "evidence": [],
        }

        effective = sniff_projects.compose_effective_profile(
            detected,
            repo_override={
                "remove": ["javascript", "react", "node", "npm"],
                "add": ["python"],
                "capabilities": ["documentation"],
            },
            project_override={"add": ["docker"], "capabilities": ["containers"]},
        )

        self.assertEqual(effective["languages"], ["python"])
        self.assertEqual(effective["frameworks"], [])
        self.assertEqual(effective["runtimes"], [])
        self.assertEqual(effective["tooling"], ["docker"])
        self.assertEqual(effective["capabilities"], ["backend", "containers", "documentation"])

    def test_detection_keeps_evidence_for_active_markers(self) -> None:
        with TemporaryDirectory() as temp:
            repo = Path(temp)
            repo.joinpath("package.json").write_text('{"dependencies":{"svelte":"latest"}}', encoding="utf-8")

            detected = sniff_projects.detect_technologies(repo)

            self.assertEqual(detected["languages"], ["javascript"])
            self.assertEqual(detected["frameworks"], ["svelte"])
            paths = [item["path"] for item in detected["evidence"]]
            self.assertEqual(paths, ["package.json", "package.json"])

    def test_workspace_generator_uses_effective_repository_view(self) -> None:
        with TemporaryDirectory() as temp:
            root = Path(temp)
            registry = root / "_registry"
            registry.mkdir()
            registry.joinpath("repositories.json").write_text(
                """{
  "repositories": [
    {"id": "api", "name": "api", "relative_path": "Space/api", "stack": ["python"]},
    {"id": "web", "name": "web", "relative_path": "Space/web", "stack": ["javascript"]}
  ],
  "effective_repositories": [
    {"id": "api", "name": "api", "relative_path": "Space/api", "effective": {"languages": ["python"], "frameworks": [], "runtimes": [], "tooling": [], "capabilities": ["backend"]}},
    {"id": "web", "name": "web", "relative_path": "Space/web", "effective": {"languages": ["typescript"], "frameworks": ["svelte"], "runtimes": ["node"], "tooling": [], "capabilities": ["frontend"]}}
  ]
}""",
                encoding="utf-8",
            )
            registry.joinpath("project-groups.json").write_text(
                """{"groups": [{"kind": "workspace_group", "reason": "same top-level folder", "suggested_project_id": "space", "repository_ids": ["api", "web"]}]}""",
                encoding="utf-8",
            )

            projects = generate_workspaces.build_workspaces(registry)

            self.assertEqual(len(projects), 1)
            self.assertEqual(projects[0]["stack"], ["node", "python", "svelte", "typescript"])


if __name__ == "__main__":
    unittest.main()
