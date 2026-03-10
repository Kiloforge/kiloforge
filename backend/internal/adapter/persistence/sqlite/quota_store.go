package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"kiloforge/internal/adapter/agent"
)

// QuotaStore persists quota usage to SQLite and implements dashboard.QuotaReader.
type QuotaStore struct {
	db *sql.DB
}

// NewQuotaStore creates a QuotaStore backed by the given database.
func NewQuotaStore(db *sql.DB) *QuotaStore {
	return &QuotaStore{db: db}
}

// RecordUsage upserts usage for an agent.
func (s *QuotaStore) RecordUsage(u *agent.AgentUsage) error {
	_, err := s.db.Exec(
		`INSERT INTO quota_usage (agent_id, total_cost_usd, input_tokens, output_tokens,
		  cache_read_tokens, cache_creation_tokens, result_count)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(agent_id) DO UPDATE SET
		  total_cost_usd = total_cost_usd + excluded.total_cost_usd,
		  input_tokens = input_tokens + excluded.input_tokens,
		  output_tokens = output_tokens + excluded.output_tokens,
		  cache_read_tokens = cache_read_tokens + excluded.cache_read_tokens,
		  cache_creation_tokens = cache_creation_tokens + excluded.cache_creation_tokens,
		  result_count = result_count + excluded.result_count`,
		u.AgentID, u.TotalCostUSD, u.InputTokens, u.OutputTokens,
		u.CacheReadTokens, u.CacheCreationTokens, u.ResultCount,
	)
	if err != nil {
		return fmt.Errorf("quota store: record usage for %s: %w", u.AgentID, err)
	}
	return nil
}

func (s *QuotaStore) GetAgentUsage(agentID string) *agent.AgentUsage {
	var u agent.AgentUsage
	err := s.db.QueryRow(
		`SELECT agent_id, total_cost_usd, input_tokens, output_tokens,
		        cache_read_tokens, cache_creation_tokens, result_count
		 FROM quota_usage WHERE agent_id = ?`, agentID,
	).Scan(&u.AgentID, &u.TotalCostUSD, &u.InputTokens, &u.OutputTokens,
		&u.CacheReadTokens, &u.CacheCreationTokens, &u.ResultCount)
	if err != nil {
		return nil
	}
	return &u
}

func (s *QuotaStore) GetTotalUsage() agent.TotalUsage {
	var t agent.TotalUsage
	s.db.QueryRow(
		`SELECT COALESCE(SUM(total_cost_usd), 0), COALESCE(SUM(input_tokens), 0),
		        COALESCE(SUM(output_tokens), 0), COALESCE(SUM(cache_read_tokens), 0),
		        COALESCE(SUM(cache_creation_tokens), 0), COUNT(*)
		 FROM quota_usage`,
	).Scan(&t.TotalCostUSD, &t.InputTokens, &t.OutputTokens,
		&t.CacheReadTokens, &t.CacheCreationTokens, &t.AgentCount)
	return t
}

func (s *QuotaStore) IsRateLimited() bool                  { return false }
func (s *QuotaStore) RetryAfter() time.Duration            { return 0 }
func (s *QuotaStore) TokensPerMin(_ time.Duration) float64 { return 0 }
func (s *QuotaStore) CostPerHour(_ time.Duration) float64  { return 0 }
