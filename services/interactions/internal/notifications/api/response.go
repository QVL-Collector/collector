package api

import (
	"time"

	"collector/services/interactions/internal/notifications"
)

type notificationResponse struct {
	ID        string    `json:"id"`
	ArticleID string    `json:"articleId"`
	Revision  int       `json:"revision"`
	Issue     string    `json:"issue"`
	Reason    *string   `json:"motif"`
	DecidedAt time.Time `json:"decideLe"`
}

type notificationPageResponse struct {
	Elements []notificationResponse `json:"elements"`
	Page     int                    `json:"page"`
	Size     int                    `json:"taille"`
	Total    int                    `json:"total"`
}

func toNotificationPageResponse(page notifications.Page, request notifications.PageRequest) notificationPageResponse {
	elements := make([]notificationResponse, len(page.Notifications))
	for i, notification := range page.Notifications {
		elements[i] = toNotificationResponse(notification)
	}
	return notificationPageResponse{
		Elements: elements,
		Page:     request.Number,
		Size:     request.Size,
		Total:    page.Total,
	}
}

func toNotificationResponse(notification notifications.Notification) notificationResponse {
	return notificationResponse{
		ID:        notification.ID.String(),
		ArticleID: notification.Article.String(),
		Revision:  int(notification.Revision),
		Issue:     string(notification.Issue),
		Reason:    reasonCode(notification.Reason),
		DecidedAt: notification.DecidedAt.UTC(),
	}
}

func reasonCode(reason *notifications.Reason) *string {
	if reason == nil {
		return nil
	}
	code := string(*reason)
	return &code
}
