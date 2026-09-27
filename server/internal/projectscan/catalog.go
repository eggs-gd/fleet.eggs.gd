package projectscan

// Technology vocabulary shared by repository detection and workspace cards.
// Names match the former scripts/sniff_projects.py catalog so existing
// _registry/repositories.json stays readable by the board.

type techSpec struct {
	category string
}

var technologyCatalog = map[string]techSpec{
	"csharp":         {category: "language"},
	"dart":           {category: "language"},
	"go":             {category: "language"},
	"javascript":     {category: "language"},
	"php":            {category: "language"},
	"python":         {category: "language"},
	"ruby":           {category: "language"},
	"rust":           {category: "language"},
	"typescript":     {category: "language"},
	"flutter":        {category: "framework"},
	"react":          {category: "framework"},
	"svelte":         {category: "framework"},
	"tauri":          {category: "framework"},
	"unity":          {category: "framework"},
	"vite":           {category: "framework"},
	"dotnet":         {category: "runtime"},
	"node":           {category: "runtime"},
	"docker":         {category: "tooling"},
	"docker-compose": {category: "tooling"},
	"npm":            {category: "tooling"},
	"pnpm":           {category: "tooling"},
	"yarn":           {category: "tooling"},
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
