package httpapi

import (
	"context"
	"log"
	"net/http"
	"time"

	appwarehouse "github.com/topbase/topbase/internal/app/warehouse"
	"github.com/topbase/topbase/internal/core"
)

func (s *server) listSchedules(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	items, err := s.warehouse.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *server) createSchedule(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	var input struct {
		Name               string `json:"name"`
		QuestionID         string `json:"question_id"`
		QueryID            string `json:"query_id"`
		ModelID            string `json:"model_id"`
		Cron               string `json:"cron"`
		Timezone           string `json:"timezone"`
		MaterializeTo      string `json:"materialize_to"`
		Strategy           string `json:"strategy"`
		DatabaseID         string `json:"database_id"`
		WatermarkField     string `json:"watermark_field"`
		ConfirmSourceWrite bool   `json:"confirm_source_write"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	questionID := input.QuestionID
	if questionID == "" {
		questionID = input.QueryID
	}
	saved, err := s.warehouse.Create(core.Schedule{
		Name: input.Name, QuestionID: questionID, ModelID: input.ModelID, Cron: input.Cron, Timezone: input.Timezone,
		MaterializeTo: input.MaterializeTo, Strategy: input.Strategy, DatabaseID: input.DatabaseID,
		WatermarkField: input.WatermarkField, ConfirmSourceWrite: input.ConfirmSourceWrite,
	}, user.ID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, saved)
}

func (s *server) runSchedule(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	scheduleID := r.PathValue("id")
	leaseName := "warehouse:schedule:" + scheduleID
	acquired, err := s.store.AcquireLease(r.Context(), leaseName, s.instanceID, 6*time.Hour)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "cannot reserve schedule execution"})
		return
	}
	if !acquired {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "schedule is already running on another Topbase instance"})
		return
	}
	defer s.store.ReleaseLease(context.Background(), leaseName, s.instanceID)
	run, err := s.warehouse.Run(r.Context(), scheduleID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error(), "run": run})
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (s *server) listRuns(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	items, err := s.warehouse.ListRuns(r.URL.Query().Get("schedule_id"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *server) listWarehouseTables(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	items, err := s.warehouse.ListTables()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *server) listLineage(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	items, err := s.warehouse.ListLineage(r.PathValue("type"), r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *server) proposeSchedule(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var input struct {
		QuestionID string `json:"question_id"`
		Message    string `json:"message"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	question, err := s.content.GetQuestion(input.QuestionID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, appwarehouse.Propose(question, input.Message))
}

func (s *server) tickWarehouse(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		leader, err := s.store.AcquireLease(ctx, "warehouse:scheduler", s.instanceID, 75*time.Second)
		if err != nil {
			log.Printf("topbase: acquire scheduler lease: %v", err)
			continue
		}
		if !leader {
			continue
		}
		items, err := s.warehouse.List()
		if err != nil {
			log.Printf("topbase: list schedules: %v", err)
			continue
		}
		now := time.Now()
		for _, item := range items {
			if !item.Enabled {
				continue
			}
			if !appwarehouse.Due(item.Cron, item.Timezone, now, item.LastRunAt) {
				continue
			}
			leaseName := "warehouse:schedule:" + item.ID
			acquired, err := s.store.AcquireLease(ctx, leaseName, s.instanceID, 6*time.Hour)
			if err != nil {
				log.Printf("topbase: acquire schedule %s lease: %v", item.ID, err)
				continue
			}
			if !acquired {
				continue
			}
			if _, err := s.warehouse.Run(ctx, item.ID); err != nil {
				log.Printf("topbase: schedule %s: %v", item.ID, err)
			}
			if err := s.store.ReleaseLease(context.Background(), leaseName, s.instanceID); err != nil {
				log.Printf("topbase: release schedule %s lease: %v", item.ID, err)
			}
		}
		s.notify.RunDueSubscriptions(ctx, now)
	}
}
