package manager

// Failure codes returned by the Manager API.
const (
	FailureAmbiguousProject     = "ambiguous_project"
	FailureAmbiguousRepository  = "ambiguous_repository"
	FailureAmbiguousCommand     = "ambiguous_command"
	FailureUnsafeCommand        = "unsafe_command"
	FailureMalformedModelOutput = "malformed_model_output"
	FailureProviderError        = "provider_error"
	FailureNotFound             = "not_found"
	FailureLLMNotConfigured     = "llm_not_configured"
	FailureSTTNotConfigured     = "stt_not_configured"
	FailureValidation           = "validation_error"
)

// Intent kinds for structured Manager output.
const (
	KindTask           = "task"
	KindStatusChange   = "status_change"
	KindComment        = "comment"
	KindBoardCommand   = "board_command"
	KindQuestion       = "question"
	KindAssigneeChange = "assignee_change"
	KindPriorityChange = "priority_change"
	KindCancel         = "cancel"
)

const (
	BoardShowBoard = "show_board"
	BoardShowTask  = "show_task"
)

const (
	PathDeterministic = "deterministic"
	PathLLM           = "llm"
	PathCommand       = "command"
)

// Intent is the structured Manager output validated against IntentJSONSchema.
type Intent struct {
	Kind          string   `json:"kind"`
	Project       string   `json:"project,omitempty"`
	Repository    string   `json:"repository,omitempty"`
	Repositories  []string `json:"repositories,omitempty"`
	DependsOn     []string `json:"depends_on,omitempty"`
	Title         string   `json:"title,omitempty"`
	Description   string   `json:"description,omitempty"`
	Acceptance    string   `json:"acceptance_criteria,omitempty"`
	Context       string   `json:"context,omitempty"`
	SourceInbox   string   `json:"source_inbox,omitempty"`
	Priority      *int     `json:"priority,omitempty"`
	Status        string   `json:"status,omitempty"`
	Type          string   `json:"type,omitempty"`
	Assignee      string   `json:"assignee,omitempty"`
	Ref           string   `json:"ref,omitempty"`
	Comment       string   `json:"comment,omitempty"`
	CommentAuthor string   `json:"comment_author,omitempty"`
	BoardAction   string   `json:"board_action,omitempty"`
	Query         string   `json:"query,omitempty"`
	Confirm       bool     `json:"confirm,omitempty"`
	RawTranscript string   `json:"raw_transcript,omitempty"`
}

// TextRequest is the body for POST /api/manager/text.
type TextRequest struct {
	Text   string `json:"text"`
	Source string `json:"source,omitempty"`
}

// CommandRequest is the body for POST /api/manager/command.
type CommandRequest struct {
	Intent Intent `json:"intent"`
	Source string `json:"source,omitempty"`
}

// Failure describes a typed Manager failure.
type Failure struct {
	Code        string   `json:"code"`
	Message     string   `json:"message"`
	Suggestions []string `json:"suggestions,omitempty"`
}

func (f Failure) Error() string {
	if f.Message == "" {
		return f.Code
	}
	return f.Code + ": " + f.Message
}

// Result is an action outcome returned to the client.
type Result struct {
	Action string `json:"action"`
	Ref    string `json:"ref,omitempty"`
	Path   string `json:"path,omitempty"`
	Status string `json:"status,omitempty"`
	Detail any    `json:"detail,omitempty"`
	// Warnings are problems the caller should tell the person about even
	// though the operation itself worked.
	Warnings []string `json:"warnings,omitempty"`
}

// Response is the unified Manager API response.
type Response struct {
	OK         bool     `json:"ok"`
	Path       string   `json:"path,omitempty"`
	Intent     *Intent  `json:"intent,omitempty"`
	Result     *Result  `json:"result,omitempty"`
	Failure    *Failure `json:"failure,omitempty"`
	Transcript string   `json:"transcript,omitempty"`
}
