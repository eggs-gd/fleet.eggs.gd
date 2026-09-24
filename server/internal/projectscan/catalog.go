package projectscan

// Technology vocabulary shared by repository detection and workspace cards.
// Names match the former scripts/sniff_projects.py catalog so existing
// _registry/repositories.json stays readable by the board.

type techSpec struct {
	category     string
	capabilities []string
}

var technologyCatalog = map[string]techSpec{
	"csharp":         {category: "language", capabilities: []string{"dotnet"}},
	"dart":           {category: "language", capabilities: []string{"mobile"}},
	"go":             {category: "language", capabilities: []string{"backend"}},
	"javascript":     {category: "language", capabilities: []string{"frontend"}},
	"php":            {category: "language", capabilities: []string{"backend"}},
	"python":         {category: "language", capabilities: []string{"backend"}},
	"ruby":           {category: "language", capabilities: []string{"backend"}},
	"rust":           {category: "language", capabilities: []string{"backend"}},
	"typescript":     {category: "language", capabilities: []string{"frontend"}},
	"flutter":        {category: "framework", capabilities: []string{"mobile"}},
	"react":          {category: "framework", capabilities: []string{"frontend"}},
	"svelte":         {category: "framework", capabilities: []string{"frontend"}},
	"tauri":          {category: "framework", capabilities: []string{"desktop"}},
	"unity":          {category: "framework", capabilities: []string{"game-engine"}},
	"vite":           {category: "framework", capabilities: []string{"frontend"}},
	"dotnet":         {category: "runtime", capabilities: []string{"dotnet"}},
	"node":           {category: "runtime", capabilities: []string{"frontend"}},
	"docker":         {category: "tooling", capabilities: []string{"containers"}},
	"docker-compose": {category: "tooling", capabilities: []string{"containers"}},
	"npm":            {category: "tooling", capabilities: []string{"frontend"}},
	"pnpm":           {category: "tooling", capabilities: []string{"frontend"}},
	"yarn":           {category: "tooling", capabilities: []string{"frontend"}},
}

var stackMarkers = map[string][]string{
	"package.json":        {"node", "javascript"},
	"package-lock.json":   {"npm"},
	"yarn.lock":           {"yarn"},
	"pnpm-lock.yaml":      {"pnpm"},
	"pyproject.toml":      {"python"},
	"requirements.txt":    {"python"},
	"go.mod":              {"go"},
	"Cargo.toml":          {"rust"},
	"Dockerfile":          {"docker"},
	"docker-compose.yml":  {"docker-compose"},
	"docker-compose.yaml": {"docker-compose"},
	"composer.json":       {"php"},
	"Gemfile":             {"ruby"},
	"pubspec.yaml":        {"dart", "flutter"},
}

var pruneDirs = map[string]struct{}{
	".cache": {}, ".mypy_cache": {}, ".next": {}, ".nuxt": {}, ".pytest_cache": {},
	".svelte-kit": {}, ".turbo": {}, ".venv": {}, ".yarn": {}, "__pycache__": {},
	"bin": {}, "build": {}, "dist": {}, "deps": {}, "external": {}, "generated": {},
	"Library": {}, "lib-src": {}, "node_modules": {}, "obj": {}, "Temp": {},
	"target": {}, "third_party": {}, "vendor": {}, "venv": {},
}

var deletionNameHints = map[string]struct{}{
	"backup": {}, "copy": {}, "demo": {}, "old": {}, "playground": {},
	"sonar": {}, "test": {}, "tests": {}, "tmp": {},
}

var readmeNames = []string{"README.md", "README.MD", "README.txt", "README"}
