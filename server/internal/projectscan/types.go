package projectscan

// Repository is one discovered git checkout.
type Repository struct {
	ID                  string
	RemoteIdentity      string
	Name                string
	Path                string
	RelativePath        string
	ScanRoot            string
	Remote              string
	Branch              string
	Head                string
	ReadmePath          string
	Title               string
	Summary             string
	SummarySource       string
	Detected            Detected
	Effective           Effective
	Stack               []string
	Markers             []string
	ParentRepositoryID  string
	NestedRepositoryIDs []string
	FirstSeenAt         string
	LastSeenAt          string
}

// TechLists is the controlled technology vocabulary, grouped.
type TechLists struct {
	Languages  []string `json:"languages"`
	Frameworks []string `json:"frameworks"`
	Runtimes   []string `json:"runtimes"`
	Tooling    []string `json:"tooling"`
}

// Detected is raw scanner output plus evidence.
type Detected struct {
	TechLists
	Evidence []Evidence `json:"evidence"`
}

// Effective is the profile downstream launch and workspace cards read.
type Effective struct {
	TechLists
}

// Evidence is one marker that contributed technologies.
type Evidence struct {
	Kind         string   `json:"kind"`
	Path         string   `json:"path"`
	Technologies []string `json:"technologies"`
	Detail       string   `json:"detail"`
	Ignored      bool     `json:"ignored"`
	IgnoreReason *string  `json:"ignore_reason"`
}

// Group is a registry suggestion that several repositories belong together.
type Group struct {
	ID                 string   `json:"id"`
	Kind               string   `json:"kind"`
	Confidence         string   `json:"confidence"`
	Decision           string   `json:"decision"`
	Reason             string   `json:"reason"`
	SuggestedProjectID string   `json:"suggested_project_id"`
	RepositoryIDs      []string `json:"repository_ids"`
}

// DeletionCandidate is a review-only cleanup hint. Nothing is deleted from it.
type DeletionCandidate struct {
	RepositoryID string   `json:"repository_id"`
	Name         string   `json:"name"`
	Path         string   `json:"path"`
	RelativePath string   `json:"relative_path"`
	Remote       *string  `json:"remote"`
	Branch       *string  `json:"branch"`
	Head         *string  `json:"head"`
	Decision     string   `json:"decision"`
	Reasons      []string `json:"reasons"`
	Details      []string `json:"details"`
	GitStatus    string   `json:"git_status"`
	Confidence   string   `json:"confidence"`
}

func emptyTech() TechLists {
	return TechLists{
		Languages:  []string{},
		Frameworks: []string{},
		Runtimes:   []string{},
		Tooling:    []string{},
	}
}

func emptyDetected() Detected {
	return Detected{TechLists: emptyTech(), Evidence: []Evidence{}}
}
