package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"kiloforge/internal/core/domain"
)

// ReliabilityStore implements port.ReliabilityStore using SQLite.
type ReliabilityStore struct {
	db *sql.DB
}

// NewReliabilityStore creates a new SQLite-backed reliability store.
func NewReliabilityStore(db *sql.DB) *ReliabilityStore {
	return &ReliabilityStore{db: db}
}

func (s *ReliabilityStore) Insert(event domain.ReliabilityEvent) error {
	var detailJSON sql.NullString
	if len(event.Detail) > 0 {
		b, err := json.Marshal(event.Detail)
		if err != nil {
			return fmt.Errorf("marshal detail: %w", err)
		}
		detailJSON = sql.NullString{String: string(b), Valid: true}
	}

	_, err := s.db.Exec(
		`INSERT INTO reliability_events (id, event_type, severity, agent_id, scope, detail, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		event.ID, event.EventType, event.Severity,
		nullStr(event.AgentID), nullStr(event.Scope),
		detailJSON,
		event.CreatedAt.UTC().Format(time.RFC3339Nano),
	)
	return err
}

func (s *ReliabilityStore) List(filter domain.ReliabilityFilter, opts domain.PageOpts) (domain.Page[domain.ReliabilityEvent], error) {
	opts.Normalize()

	filterWhere, filterArgs := buildFilterWhere(filter)

	// Count total (without cursor).
	var total int
	countQuery := "SELECT COUNT(*) FROM reliability_events"
	if filterWhere != "" {
		countQuery += " WHERE " + filterWhere
	}
	if err := s.db.QueryRow(countQuery, filterArgs...).Scan(&total); err != nil {
		return domain.Page[domain.ReliabilityEvent]{}, fmt.Errorf("count reliability: %w", err)
	}

	// Build query with cursor.
	var whereParts []string
	var args []any
	if filterWhere != "" {
		whereParts = append(whereParts, filterWhere)
		args = append(args, filterArgs...)
	}
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

	query := `SELECT id, event_type, severity, agent_id, scope, detail, created_at
	          FROM reliability_events` + where + ` ORDER BY created_at DESC, id DESC LIMIT ?`
	args = append(args, opts.Limit+1)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return domain.Page[domain.ReliabilityEvent]{}, fmt.Errorf("list reliability: %w", err)
	}
	defer rows.Close()

	var items []domain.ReliabilityEvent
	for rows.Next() {
		ev, err := scanEvent(rows)
		if err != nil {
			return domain.Page[domain.ReliabilityEvent]{}, err
		}
		items = append(items, ev)
	}
	if err := rows.Err(); err != nil {
		return domain.Page[domain.ReliabilityEvent]{}, err
	}

	var nextCursor string
	if len(items) > opts.Limit {
		last := items[opts.Limit-1]
		nextCursor = domain.EncodeCursor(last.CreatedAt.UTC().Format(time.RFC3339Nano), last.ID)
		items = items[:opts.Limit]
	}

	return domain.Page[domain.ReliabilityEvent]{
		Items:      items,
		NextCursor: nextCursor,
		TotalCount: total,
	}, nil
}

func (s *ReliabilityStore) Summary(filter domain.ReliabilityFilter, buckets int) (domain.ReliabilitySummary, error) {
	if buckets <= 0 {
		buckets = 24
	}

	var since, until time.Time
	if filter.Since != nil {
		since = *filter.Since
	}
	if filter.Until != nil {
		until = *filter.Until
	} else {
		until = time.Now().UTC()
	}
	if since.IsZero() {
		since = until.Add(-24 * time.Hour)
	}

	duration := until.Sub(since)
	bucketDuration := duration / time.Duration(buckets)

	// Query all events in the time range with optional type/severity filters.
	filterWhere, filterArgs := buildFilterWhere(domain.ReliabilityFilter{
		EventTypes: filter.EventTypes,
		Severities: filter.Severities,
		Since:      &since,
		Until:      &until,
	})

	query := "SELECT event_type, severity, created_at FROM reliability_events"
	if filterWhere != "" {
		query += " WHERE " + filterWhere
	}
	query += " ORDER BY created_at ASC"

	rows, err := s.db.Query(query, filterArgs...)
	if err != nil {
		return domain.ReliabilitySummary{}, fmt.Errorf("summary query: %w", err)
	}
	defer rows.Close()

	// Initialize buckets.
	summaryBuckets := make([]domain.ReliabilityBucket, buckets)
	for i := range summaryBuckets {
		summaryBuckets[i] = domain.ReliabilityBucket{
			Start:  since.Add(bucketDuration * time.Duration(i)),
			End:    since.Add(bucketDuration * time.Duration(i+1)),
			Counts: make(map[string]int),
		}
	}

	totals := domain.ReliabilityTotals{
		ByType:     make(map[string]int),
		BySeverity: make(map[string]int),
	}

	for rows.Next() {
		var eventType, severity, createdAtStr string
		if err := rows.Scan(&eventType, &severity, &createdAtStr); err != nil {
			return domain.ReliabilitySummary{}, err
		}
		createdAt, err := time.Parse(time.RFC3339Nano, createdAtStr)
		if err != nil {
			createdAt, err = time.Parse(time.RFC3339, createdAtStr)
			if err != nil {
				continue
			}
		}

		totals.Total++
		totals.ByType[eventType]++
		totals.BySeverity[severity]++

		// Place into correct bucket.
		idx := int(createdAt.Sub(since) / bucketDuration)
		if idx >= buckets {
			idx = buckets - 1
		}
		if idx < 0 {
			idx = 0
		}
		summaryBuckets[idx].Counts[eventType]++
	}
	if err := rows.Err(); err != nil {
		return domain.ReliabilitySummary{}, err
	}

	windowStr := formatDuration(duration)
	bucketStr := formatDuration(bucketDuration)

	return domain.ReliabilitySummary{
		Window:         windowStr,
		BucketDuration: bucketStr,
		Buckets:        summaryBuckets,
		Totals:         totals,
	}, nil
}

// buildFilterWhere builds WHERE clause parts from a ReliabilityFilter.
func buildFilterWhere(filter domain.ReliabilityFilter) (string, []any) {
	var parts []string
	var args []any

	if len(filter.EventTypes) > 0 {
		ph := make([]string, len(filter.EventTypes))
		for i, t := range filter.EventTypes {
			ph[i] = "?"
			args = append(args, t)
		}
		parts = append(parts, "event_type IN ("+strings.Join(ph, ",")+")")
	}
	if len(filter.Severities) > 0 {
		ph := make([]string, len(filter.Severities))
		for i, s := range filter.Severities {
			ph[i] = "?"
			args = append(args, s)
		}
		parts = append(parts, "severity IN ("+strings.Join(ph, ",")+")")
	}
	if filter.Since != nil {
		parts = append(parts, "created_at >= ?")
		args = append(args, filter.Since.UTC().Format(time.RFC3339Nano))
	}
	if filter.Until != nil {
		parts = append(parts, "created_at <= ?")
		args = append(args, filter.Until.UTC().Format(time.RFC3339Nano))
	}

	return strings.Join(parts, " AND "), args
}

func scanEvent(rows *sql.Rows) (domain.ReliabilityEvent, error) {
	var ev domain.ReliabilityEvent
	var agentID, scope, detailStr, createdAtStr sql.NullString

	if err := rows.Scan(&ev.ID, &ev.EventType, &ev.Severity,
		&agentID, &scope, &detailStr, &createdAtStr); err != nil {
		return ev, err
	}

	ev.AgentID = agentID.String
	ev.Scope = scope.String

	if detailStr.Valid && detailStr.String != "" {
		var detail map[string]any
		if err := json.Unmarshal([]byte(detailStr.String), &detail); err == nil {
			ev.Detail = detail
		}
	}

	if createdAtStr.Valid {
		if t, err := time.Parse(time.RFC3339Nano, createdAtStr.String); err == nil {
			ev.CreatedAt = t
		} else if t, err := time.Parse(time.RFC3339, createdAtStr.String); err == nil {
			ev.CreatedAt = t
		}
	}

	return ev, nil
}

func nullStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func formatDuration(d time.Duration) string {
	if d >= 24*time.Hour {
		days := int(d / (24 * time.Hour))
		return fmt.Sprintf("%dd", days)
	}
	if d >= time.Hour {
		hours := int(d / time.Hour)
		return fmt.Sprintf("%dh", hours)
	}
	minutes := int(d / time.Minute)
	return fmt.Sprintf("%dm", minutes)
}
