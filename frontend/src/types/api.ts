export interface Agent {
  id: string;
  name?: string;
  role: string;
  ref: string;
  status: string;
  session_id: string;
  pid: number;
  worktree_dir: string;
  log_file: string;
  started_at: string;
  updated_at: string;
  finished_at?: string;
  suspended_at?: string;
  shutdown_reason?: string;
  resume_error?: string;
  uptime_seconds?: number;
  estimated_cost_usd?: number;
  input_tokens?: number;
  output_tokens?: number;
  cache_read_tokens?: number;
  cache_creation_tokens?: number;
  model?: string;
}

export interface QuotaAgent {
  agent_id: string;
  estimated_cost_usd: number;
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_creation_tokens: number;
}

export interface QuotaResponse {
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_creation_tokens: number;
  estimated_cost_usd: number;
  agent_count: number;
  rate_limited: boolean;
  retry_after_seconds?: number;
  agents?: QuotaAgent[];
}

export interface Track {
  id: string;
  title: string;
  status: string;
  project?: string;
}

export interface TrackDetail {
  id: string;
  title: string;
  status: string;
  type?: string;
  spec?: string;
  plan?: string;
  phases_total?: number;
  phases_completed?: number;
  tasks_total?: number;
  tasks_completed?: number;
  created_at?: string;
  updated_at?: string;
}

export interface Project {
  slug: string;
  repo_name: string;
  origin_remote?: string;
  active: boolean;
}

export interface AddProjectRequest {
  remote_url: string;
  name?: string;
  ssh_key?: string;
}

export interface SSHKeyInfo {
  name: string;
  path: string;
  type: string;
  comment?: string;
}

export interface StatusResponse {
  gitea_url: string;
  agent_counts: Record<string, number>;
  total_agents: number;
  sse_clients: number;
  estimated_cost_usd?: number;
  rate_limited?: boolean;
}

export interface SSEEventData {
  type: string;
  data: unknown;
}

export interface LogResponse {
  agent_id: string;
  lines: string[];
  total: number;
}

export interface SkillDetail {
  name: string;
  modified: boolean;
  changed_files?: string[];
}

export interface SkillsStatus {
  installed_version: string;
  available_version?: string;
  update_available: boolean;
  repo?: string;
  skills: SkillDetail[];
}

export interface TraceSummary {
  trace_id: string;
  root_name: string;
  span_count: number;
  start_time: string;
  end_time: string;
}

export interface SpanEvent {
  name: string;
  timestamp: string;
  attributes?: Record<string, string>;
}

export interface SpanInfo {
  trace_id: string;
  span_id: string;
  parent_id?: string;
  name: string;
  start_time: string;
  end_time: string;
  duration_ms: number;
  status: string;
  attributes?: Record<string, string>;
  events?: SpanEvent[];
}

export interface TraceDetail {
  trace_id: string;
  spans: SpanInfo[];
}

export interface BoardCard {
  track_id: string;
  title: string;
  type?: string;
  column: string;
  position: number;
  agent_id?: string;
  agent_status?: string;
  assigned_worker?: string;
  pr_number?: number;
  trace_id?: string;
  moved_at: string;
  created_at: string;
}

export interface BoardState {
  columns: string[];
  cards: Record<string, BoardCard>;
}

export interface SyncStatus {
  ahead: number;
  behind: number;
  status: "synced" | "ahead" | "behind" | "diverged" | "unknown";
  local_branch: string;
  remote_url?: string;
}

export interface PushRequest {
  remote_branch: string;
  force?: boolean;
}

export interface PushResult {
  success: boolean;
  local_branch: string;
  remote_branch: string;
}

export interface PullRequest {
  remote_branch?: string;
}

export interface PullResult {
  success: boolean;
  new_head: string;
}

export interface ConfigResponse {
  dashboard_enabled: boolean;
}

export interface UpdateConfigRequest {
  dashboard_enabled?: boolean;
}

export interface SpawnInteractiveRequest {
  work_dir?: string;
  model?: string;
}
