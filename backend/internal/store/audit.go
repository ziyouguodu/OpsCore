package store

import (
	"context"
	"encoding/json"
	"errors"

	"opscore/backend/internal/models"
)

func (s *Store) RecordAudit(ctx context.Context, event models.AuditEvent) error {
	if event.Action == "" || event.ResourceType == "" {
		return errors.New("audit action and resource type are required")
	}
	if event.Outcome != "success" && event.Outcome != "failure" {
		return errors.New("audit outcome must be success or failure")
	}
	detail, err := json.Marshal(event.Detail)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		insert into audit_events(
			actor_user_id, actor_username, action, resource_type, resource_id,
			outcome, detail, ip_address, user_agent
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`, nullableUserID(event.ActorUserID), event.ActorUsername, event.Action, event.ResourceType,
		event.ResourceID, event.Outcome, detail, event.IPAddress, event.UserAgent)
	return err
}

func (s *Store) ListAuditEvents(ctx context.Context, limit int) ([]models.AuditEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `
		select id, coalesce(actor_user_id, 0), actor_username, action, resource_type,
			resource_id, outcome, detail, ip_address, user_agent, created_at
		from audit_events
		order by created_at desc, id desc
		limit $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := make([]models.AuditEvent, 0)
	for rows.Next() {
		var event models.AuditEvent
		var detail []byte
		if err := rows.Scan(&event.ID, &event.ActorUserID, &event.ActorUsername, &event.Action,
			&event.ResourceType, &event.ResourceID, &event.Outcome, &detail, &event.IPAddress,
			&event.UserAgent, &event.CreatedAt); err != nil {
			return nil, err
		}
		if len(detail) > 0 {
			if err := json.Unmarshal(detail, &event.Detail); err != nil {
				return nil, err
			}
		}
		events = append(events, event)
	}
	return events, rows.Err()
}
