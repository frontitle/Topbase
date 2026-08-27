package notify

import (
	"context"
	"fmt"
	"strings"
	"time"

	appwarehouse "github.com/topbase/topbase/internal/app/warehouse"
	"github.com/topbase/topbase/internal/core"
)

func (s Service) CreateSubscription(input core.Subscription, userID string) (core.Subscription, error) {
	if s.Subscriptions == nil {
		return core.Subscription{}, fmt.Errorf("subscriptions are not configured")
	}
	if strings.TrimSpace(input.DashboardID) == "" || strings.TrimSpace(input.Cron) == "" {
		return core.Subscription{}, fmt.Errorf("dashboard_id and cron are required")
	}
	if _, err := appwarehouse.ParseCron(input.Cron); err != nil {
		return core.Subscription{}, err
	}
	if _, err := s.Content.GetDashboard(input.DashboardID); err != nil {
		return core.Subscription{}, err
	}
	if input.Channel == "" {
		input.Channel = "inbox"
	}
	if input.Channel != "inbox" && input.Channel != "feishu" && !strings.HasPrefix(input.Channel, "webhook:") {
		return core.Subscription{}, fmt.Errorf("unsupported channel %q", input.Channel)
	}
	if input.Timezone == "" {
		input.Timezone = "Asia/Shanghai"
	}
	input.ID = core.NewID("sub")
	input.Enabled = true
	input.CreatedBy = userID
	input.CreatedAt = time.Now().UTC()
	if err := s.Subscriptions.Create(input); err != nil {
		return core.Subscription{}, err
	}
	return input, nil
}

func (s Service) ListSubscriptions(dashboardID string) ([]core.Subscription, error) {
	if s.Subscriptions == nil {
		return []core.Subscription{}, nil
	}
	if dashboardID == "" {
		return s.Subscriptions.List()
	}
	return s.Subscriptions.ListByDashboard(dashboardID)
}

func (s Service) RunSubscription(ctx context.Context, id string) (core.Notification, error) {
	if s.Subscriptions == nil {
		return core.Notification{}, fmt.Errorf("subscriptions are not configured")
	}
	sub, err := s.Subscriptions.ByID(id)
	if err != nil {
		return core.Notification{}, err
	}
	board, err := s.Content.GetDashboard(sub.DashboardID)
	if err != nil {
		return core.Notification{}, err
	}
	ok, fail, rows := 0, 0, 0
	for _, card := range board.Cards {
		if card.Type != "question" || card.QuestionID == "" {
			continue
		}
		result, err := s.Dataset.RunDashboardCard(ctx, board, card.ID, map[string]any{}, s.Content.Questions)
		if err != nil {
			fail++
			continue
		}
		ok++
		rows += len(result.Rows)
	}
	body := fmt.Sprintf("%s：成功 %d 张卡，失败 %d，共 %d 行", board.Name, ok, fail, rows)
	note := core.Notification{
		ID: core.NewID("ntf"), UserID: sub.CreatedBy, Title: "仪表盘订阅", Body: body, CreatedAt: time.Now().UTC(),
	}
	if err := s.Content.RecordNotification(note); err != nil {
		return core.Notification{}, err
	}
	now := time.Now().UTC()
	sub.LastRunAt = &now
	_ = s.Subscriptions.Update(sub)
	if s.Deliver != nil {
		s.Deliver(note.Title, note.Body, sub.Channel)
	}
	return note, nil
}

func (s Service) RunDueSubscriptions(ctx context.Context, now time.Time) {
	if s.Subscriptions == nil {
		return
	}
	items, err := s.Subscriptions.List()
	if err != nil {
		return
	}
	for _, item := range items {
		if !item.Enabled {
			continue
		}
		if !appwarehouse.Due(item.Cron, item.Timezone, now, item.LastRunAt) {
			continue
		}
		_, _ = s.RunSubscription(ctx, item.ID)
	}
}
