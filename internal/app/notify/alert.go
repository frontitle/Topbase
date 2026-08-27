package notify

import (
	"context"
	"fmt"
	"strconv"

	"github.com/topbase/topbase/internal/app/content"
	appquery "github.com/topbase/topbase/internal/app/query"
	"github.com/topbase/topbase/internal/core"
)

type Service struct {
	Content       content.Service
	Dataset       appquery.DatasetService
	Subscriptions core.SubscriptionStore
	Deliver       func(title, body, channel string)
}

func (s Service) RunAlert(ctx context.Context, alertID string) (core.Notification, error) {
	alert, err := s.Content.GetAlert(alertID)
	if err != nil {
		return core.Notification{}, err
	}
	if !alert.Enabled {
		return core.Notification{}, fmt.Errorf("alert is disabled")
	}
	question, err := s.Content.GetQuestion(alert.QuestionID)
	if err != nil {
		return core.Notification{}, err
	}
	if question.QueryIR == nil {
		return core.Notification{}, fmt.Errorf("question has no queryir")
	}
	result, err := s.Dataset.Run(ctx, *question.QueryIR)
	if err != nil {
		return core.Notification{}, err
	}
	fired, body := evaluate(alert, result)
	if !fired {
		return core.Notification{}, fmt.Errorf("alert condition not met")
	}
	note := core.Notification{
		ID: core.NewID("ntf"), UserID: alert.CreatedBy, AlertID: alert.ID,
		Title: alert.Name, Body: body,
	}
	if err := s.Content.RecordNotification(note); err != nil {
		return core.Notification{}, err
	}
	if alert.Once {
		alert.Enabled = false
		_ = s.Content.Alerts.Update(alert)
	}
	return note, nil
}

func evaluate(alert core.Alert, result appquery.DatasetResult) (bool, string) {
	switch alert.Kind {
	case "goal", "progress":
		value, ok := lastNumeric(result.Rows)
		if !ok {
			return false, ""
		}
		if alert.Kind == "goal" && value >= alert.Goal {
			return true, fmt.Sprintf("当前值 %.2f 已达到目标 %.2f", value, alert.Goal)
		}
		if alert.Kind == "progress" && value >= alert.Goal {
			return true, fmt.Sprintf("进度 %.2f 已达到 %.2f", value, alert.Goal)
		}
		return false, ""
	default:
		if len(result.Rows) == 0 {
			return false, ""
		}
		return true, fmt.Sprintf("查询返回 %d 行", len(result.Rows))
	}
}

func lastNumeric(rows [][]any) (float64, bool) {
	if len(rows) == 0 || len(rows[0]) == 0 {
		return 0, false
	}
	cell := rows[len(rows)-1][len(rows[len(rows)-1])-1]
	switch v := cell.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case string:
		n, err := strconv.ParseFloat(v, 64)
		return n, err == nil
	default:
		return 0, false
	}
}
