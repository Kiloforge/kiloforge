export interface Agent {
  id: string;
  role: string;
  ref: string;
  status: string;
  session_id: string;
  pid: number;
  worktree_dir: string;
  log_file: string;
  started_at: string;
  updated_at: string;
  suspended_at?: string;
  shutdown_reason?: string;
  resume_error?: string;
  uptime_seconds?: number;
  cost_usd?: number;
  input_tokens?: number;
  output_tokens?: number;
}

export interface QuotaAgent {
  agent_id: string;
  cost_usd: number;
  input_tokens: number;
  output_tokens: number;
}

export interface QuotaResponse {
  total_cost_usd: number;
  input_tokens: number;
  output_tokens: number;
  agent_count: number;
  rate_limited: boolean;
  retry_after_seconds?: number;
  agents?: QuotaAgent[];
}

export interface Track {
  id: string;
  title: string;
  status: string;
}

export interface StatusResponse {
  gitea_url: string;
  agent_counts: Record<string, number>;
  total_agents: number;
  sse_clients: number;
  total_cost_usd?: number;
  rate_limited?: boolean;
}

export interface SSEEventData {
  type: string;
  data: Agent | { id: string } | QuotaResponse;
}

export interface LogResponse {
  agent_id: string;
  lines: string[];
  total: number;
}
