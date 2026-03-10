package rest

import (
	"context"

	"kiloforge/internal/adapter/rest/gen"
)

// ListNotifications returns active notifications, optionally filtered by agent_id.
func (h *APIHandler) ListNotifications(_ context.Context, req gen.ListNotificationsRequestObject) (gen.ListNotificationsResponseObject, error) {
	if h.notifSvc == nil {
		return gen.ListNotifications200JSONResponse{Items: []gen.Notification{}}, nil
	}

	agentID := ""
	if req.Params.AgentId != nil {
		agentID = *req.Params.AgentId
	}

	items, err := h.notifSvc.ListActive(agentID)
	if err != nil {
		return gen.ListNotifications500JSONResponse{Error: err.Error()}, nil
	}

	result := make([]gen.Notification, 0, len(items))
	for _, n := range items {
		gn := gen.Notification{
			Id:        n.ID,
			AgentId:   n.AgentID,
			Title:     n.Title,
			Body:      n.Body,
			CreatedAt: n.CreatedAt,
		}
		if n.AcknowledgedAt != nil {
			gn.AcknowledgedAt = n.AcknowledgedAt
		}
		result = append(result, gn)
	}

	return gen.ListNotifications200JSONResponse{Items: result}, nil
}

// AcknowledgeNotification marks a notification as acknowledged.
func (h *APIHandler) AcknowledgeNotification(_ context.Context, req gen.AcknowledgeNotificationRequestObject) (gen.AcknowledgeNotificationResponseObject, error) {
	if h.notifSvc == nil {
		return gen.AcknowledgeNotification404JSONResponse{Error: "notifications not enabled"}, nil
	}

	if err := h.notifSvc.Acknowledge(req.Id); err != nil {
		return gen.AcknowledgeNotification404JSONResponse{Error: err.Error()}, nil
	}

	return gen.AcknowledgeNotification204Response{}, nil
}
