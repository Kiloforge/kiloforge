package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
)

var _ port.ReliabilityStore = (*ReliabilityStore)(nil)

// ReliabilityStore persists reliability events to SQLite.
type ReliabilityStore struct {
	db *sql.DB
}

// NewReliabilityStore creates a ReliabilityStore backed by the given database.
func NewReliabilityStore(db *sql.DB) *ReliabilityStore {
	return &ReliabilityStore{db: db}
}

// Insert persists a single reliability event.
func (s *ReliabilityStore) Insert(event domain.ReliabilityEvent) error {
	var detailJSON *string
	if event.Detail != nil {
		b, err := json.Marshal(event.Detail)
		if err != nil {
			return fmt.Errorf("marshal detail: %w", err)
		}
		v := string(b)
		detailJSON = &v
	}
	_, err := s.db.Exec(
		`INSERT INTO reliability_events (id, event_type, severity, agent_id, scope, detail, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		event.ID, string(event.EventType), string(event.Severity),
		nilIfEmpty(event.AgentID), nilIfEmpty(event.Scope),
		detailJSON, event.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("insert reliability event: %w", err)
	}
	return nil
}

// List returns a paginated, filtered list of reliability events.
func (s *ReliabilityStore) List(filter domain.ReliabilityFilter, opts domain.PageOpts) (domain.Page[domain.ReliabilityEvent], error) {
	opts.Normalize()

	var whereParts []string
	var args []any

	if len(filter.EventTypes) > 0 {
		ph := placeholders(len(filter.EventTypes))
		for _, t := range filter.EventTypes {
			args = append(args, string(t))
		}
		whereParts = append(whereParts, "event_type IN ("+ph+")")
	}
	if len(filter.Severities) > 0 {
		ph := placeholders(len(filter.Severities))
		for _, sv := range filter.Severities {
			args = append(args, string(sv))
		}
		whereParts = append(whereParts, "severity IN ("+ph+")")
	}
	if filter.AgentID != "" {
		whereParts = append(whereParts, "agent_id = ?")
		args = append(args, filter.AgentID)
	}
	if filter.Since != nil {
		whereParts = append(whereParts, "created_at >= ?")
		args = append(args, filter.Since.Format(time.RFC3339))
	}
	if filter.Until != nil {
		whereParts = append(whereParts, "created_at < ?")
		args = append(args, filter.Until.Format(time.RFC3339))
	}

	// Save filter parts for count query (no cursor).
	countParts := make([]string, len(whereParts))
	copy(countParts, whereParts)
	countArgs := make([]any, len(args))
	copy(countArgs, args)

	if opts.Cursor != "" {
		cur := domain.DecodeCursor(opts.Cursor)
		if cur.SortVal != "" {
			whereParts = append(whereParts, "(created_at < ? OR (created_at = ? AND id < ?))")
			args = append(args, cur.SortVal, cur.SortVal, cur.ID)
		}
	}

	where := ""
	if len(whereParts) > 0 {
		where = " WHERE " + strings.Join(whereParts, " AND ")
	}
	countWhere := ""
	if len(countParts) > 0 {
		countWhere = " WHERE " + strings.Join(countParts, " AND ")
	}

	var total int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM reliability_events"+countWhere, countArgs...).Scan(&total); err != nil {
		return domain.Page[domain.ReliabilityEvent]{}, fmt.Errorf("count reliability events: %w", err)
	}

	query := `SELECT id, event_type, severity, agent_id, scope, detail, created_at
	          FROM reliability_events` + where + ` ORDER BY created_at DESC, id DESC LIMIT ?`
	args = append(args, opts.Limit+1)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return domain.Page[domain.ReliabilityEvent]{}, fmt.Errorf("list reliability events: %w", err)
	}
	defer rows.Close()

	events := scanReliabilityEvents(rows)
	var nextCursor string
	if len(events) > opts.Limit {
		last := events[opts.Limit-1]
		nextCursor = domain.EncodeCursor(last.CreatedAt.Format(time.RFC3339), last.ID)
		events = events[:opts.Limit]
	}

	return domain.Page[domain.ReliabilityEvent]{
		Items:      events,
		NextCursor: nextCursor,
		TotalCount: total,
	}, nil
}

// Summary returns aggregated event counts bucketed by time.
func (s *ReliabilityStore) Summary(since, until time.Time, bucket string) (domain.ReliabilitySummary, error) {
	truncExpr := "%Y-%m-%dT%H:00:00Z" // hour bucket
	if bucket == "day" {
		truncExpr = "%Y-%m-%dT00:00:00Z"
	}

	query := `SELECT strftime(?, created_at) AS bucket, event_type, COUNT(*) AS cnt
	          FROM reliability_events
	          WHERE created_at >= ? AND created_at < ?
	          GROUP BY bucket, event_type
	          ORDER BY bucket ASC, event_type ASC`

	rows, err := s.db.Query(query, truncExpr, since.Format(time.RFC3339), until.Format(time.RFC3339))
	if err != nil {
		return domain.ReliabilitySummary{}, fmt.Errorf("summary query: %w", err)
	}
	defer rows.Close()

	bucketMap := make(map[string]map[string]int) // timestamp -> eventType -> count
	totals := make(map[string]int)

	for rows.Next() {
		var bucketTS, eventType string
		var cnt int
		if err := rows.Scan(&bucketTS, &eventType, &cnt); err != nil {
			continue
		}
		if bucketMap[bucketTS] == nil {
			bucketMap[bucketTS] = make(map[string]int)
		}
		bucketMap[bucketTS][eventType] += cnt
		totals[eventType] += cnt
	}

	// Build ordered bucket list.
	var buckets []domain.ReliabilityBucket
	// Iterate sorted keys by querying again or sorting map keys.
	keys := sortedKeys(bucketMap)
	for _, k := range keys {
		t, _ := time.Parse(time.RFC3339, k)
		buckets = append(buckets, domain.ReliabilityBucket{
			Timestamp: t,
			Counts:    bucketMap[k],
		})
	}

	return domain.ReliabilitySummary{
		Buckets: buckets,
		Totals:  totals,
	}, nil
}

func scanReliabilityEvents(rows *sql.Rows) []domain.ReliabilityEvent {
	var events []domain.ReliabilityEvent
	for rows.Next() {
		var e domain.ReliabilityEvent
		var agentID, scope, detailStr *string
		var createdAt string
		var evtType, sev string
		if err := rows.Scan(&e.ID, &evtType, &sev, &agentID, &scope, &detailStr, &createdAt); err != nil {
			continue
		}
		e.EventType = domain.ReliabilityEventType(evtType)
		e.Severity = domain.Severity(sev)
		if agentID != nil {
			e.AgentID = *agentID
		}
		if scope != nil {
			e.Scope = *scope
		}
		if detailStr != nil {
			_ = json.Unmarshal([]byte(*detailStr), &e.Detail)
		}
		e.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		events = append(events, e)
	}
	return events
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func placeholders(n int) string {
	if n == 0 {
		return ""
	}
	return strings.Repeat("?,", n)[:2*n-1]
}

func sortedKeys(m map[string]map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Simple insertion sort — bucket count is typically small.
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}
