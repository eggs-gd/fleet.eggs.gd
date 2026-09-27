/**
 * Shared, intentionally loose domain shapes mirrored from the Go backend
 * (server/internal/board, taskflow, executionapi). Fields are optional
 * because every consumer already reads them defensively (`?.`) — these
 * types describe what's actually accessed, not a parallel schema.
 */

/** A record whose shape genuinely varies by caller (session, execution, or task fragment). */
export type AnyRecord = Record<string, any>;

export interface TaskComment {
  text?: string;
  author?: string;
  created_at?: string;
}

export interface LaunchEvaluation {
  outcome?: string;
  visibility_mode?: string;
  waiting?: { reason?: string };
  failed_gates?: string[];
  agent?: string;
}

export interface ExecutionState {
  state?: string;
  visibility_mode?: string;
  session_pointer?: string;
  blocking_reason?: string;
  terminal_reason?: string;
  agent?: string;
  backend?: string;
  provider?: string;
  thread_id?: string;
  turn_id?: string;
  thread_title?: string;
  last_event?: string;
  last_activity_at?: string;
  last_event_at?: string;
  last_output_at?: string;
  last_status_change_at?: string;
}

export interface Task {
  id?: string;
  ref?: string;
  status?: string;
  type?: string;
  priority?: number | string;
  title?: string;
  summary?: string;
  updated_at?: string;
  created_at?: string;
  relative_path?: string;
  path?: string;
  project_id?: string;
  project?: string;
  workspace_id?: string;
  assignee?: string;
  execution?: ExecutionState;
  launch?: { agent?: string };
  body?: string;
  launch_evaluation?: LaunchEvaluation;
  comments?: TaskComment[];
  blocked_reason?: string;
  depends_on?: string[];
  repositories?: string[];
}

export interface ToolUsage {
  warning?: string;
  profiles?: string[];
  required?: string[];
  used?: ToolCall[];
  missing?: string[];
}

export interface ToolCall {
  name?: string;
  status?: string;
  called_at?: string;
  summary?: string;
  source?: string;
}

export interface SessionCapabilities {
  can_detect_running_session?: string;
  can_resume_session?: string;
  can_query_thread_status?: string;
  can_show_app_visible_link?: string;
  can_accept_operator_input?: string;
  can_confirm_terminal_outcome?: string;
  notes?: string;
}

export interface Session {
  claim_id?: string;
  task_ref?: string;
  task_id?: string;
  task_path?: string;
  task_title?: string;
  agent?: string;
  assignee?: string;
  provider?: string;
  repository?: string;
  project_id?: string;
  role?: string;
  status?: string;
  execution_status?: string;
  execution_state?: string;
  result?: { outcome?: string; summary?: string };
  provider_controllable?: boolean;
  resume_attempted?: boolean;
  resume_outcome?: string;
  started_at?: string;
  reason?: string;
  blocking_reason?: string;
  last_message?: string;
  last_event?: string;
  visibility_mode?: string;
  last_activity_at?: string;
  resumable?: boolean;
  supersedes_claim_id?: string;
  remote_control_url?: string;
  operator_command?: string;
  background_id?: string;
  last_output_at?: string;
  last_event_at?: string;
  last_status_change_at?: string;
  exited_at?: string;
  claimed_at?: string;
  capabilities?: SessionCapabilities;
  provider_error?: {
    reason?: string;
    detail?: string;
    kind?: string;
    suggested_action?: string;
    retry_policy?: string;
  };
  codex_thread_title?: string;
  thread_title?: string;
  host_name?: string;
  host_id?: string;
  process_id?: string;
  thread_id?: string;
  codex_thread_id?: string;
  cursor_chat_id?: string;
  session_id?: string;
  turn_id?: string;
  codex_turn_id?: string;
  log_path?: string;
  tool_usage?: ToolUsage;
  tool_warning?: string;
}

export interface SessionGroup {
  task_path?: string;
  task_id?: string;
  task_ref?: string;
  sessions?: Session[];
}

export interface Repository {
  id?: string;
  relative_path?: string;
  path?: string;
  name?: string;
  remote?: string;
  branch?: string;
  parent_repository_id?: string;
  nested_repository_ids?: string[];
  effective_tags?: string[];
  detected_tags?: string[];
  technology?: AnyRecord;
  evidence?: unknown[];
}

export interface Project {
  id: string;
  title?: string;
  workspace_id?: string;
  kind?: string;
  source?: string;
  status?: string;
  review_status?: string;
  path?: string;
  relative_path?: string;
  summary?: string;
  tag?: string;
  repositories?: string[];
  technology?: AnyRecord;
}

export interface Workspace {
  id: string;
  title?: string;
  kind?: string;
  tag?: string;
  repositories?: string[];
  workspace_id?: string;
}

export interface AgentCapabilityFlags {
  can_show_app_visible_link?: string | boolean;
  can_accept_operator_input?: string | boolean;
  can_resume_session?: string | boolean;
  can_detect_running_session?: string | boolean;
  can_query_thread_status?: string | boolean;
  can_confirm_terminal_outcome?: string | boolean;
  can_list_sessions?: string | boolean;
}

export interface AgentSessionReuse {
  level?: 'verified' | 'unverified' | 'unsupported' | string;
  can_attempt?: boolean;
  reason?: string;
}

export interface Agent {
  id?: string;
  capabilities?: AgentCapabilityFlags;
  session_reuse?: AgentSessionReuse;
  live_ready?: boolean;
  enabled?: { value?: boolean };
  configured_executable?: { value?: string };
  routing_instructions?: { source?: string; value?: string };
}

export interface SettingsFieldMeta {
  source?: string;
  overridden_by?: string;
  warning?: string;
}

export interface SettingsSnapshot {
  agents?: { providers?: Agent[] };
  projects?: { scan_roots?: string[] };
  general?: {
    session_timeout_config?: SettingsFieldMeta & { value?: string };
    launch_config?: SettingsFieldMeta & { value?: string };
  };
  manager?: { session?: { id?: string; status?: string } | null; provider?: string };
}

export interface SettingsDraftAgent {
  enabled?: boolean;
  executable?: string;
  routingInstructions?: string;
}

export interface SettingsDraft {
  scanRoots?: string[];
  sessionTimeout?: string;
  launch?: string;
  agents?: Record<string, SettingsDraftAgent>;
  manager?: { agent?: string; threadId?: string };
}
