package httpapi

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	appwarehouse "github.com/topbase/topbase/internal/app/warehouse"
	"github.com/topbase/topbase/internal/core"
)

func (s *server) serveWarehousePage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	user, ok := s.currentSessionUser(r)
	if !ok {
		http.Redirect(w, r, "/auth/login/", http.StatusFound)
		return
	}
	if !s.identity.IsAdmin(user.ID) {
		s.serveHTMLStatus(w, r, "warehouse/member.html", http.StatusForbidden)
		return
	}
	s.serveHTML(w, r, "warehouse/index.html")
}

func (s *server) listSchedules(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	if r.URL.Query().Get("include") == "last_run" {
		items, err := s.store.ListScheduleSummaries()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, items)
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
	if errors.Is(err, core.ErrWarehouseTargetInUse) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
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
	release, err := s.reserveWarehouseRun(r.Context(), scheduleID)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "cannot reserve schedule execution"})
		return
	}
	if release == nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "此计划正在运行，请稍后查看结果"})
		return
	}
	defer release()
	run, err := s.warehouse.Run(r.Context(), scheduleID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error(), "run": run})
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (s *server) reserveWarehouseRun(ctx context.Context, scheduleID string) (release func(), err error) {
	// Scheduler leadership is renewable by the node. Each individual run needs
	// a unique owner so that another request on that same node cannot renew or
	// release the in-progress run's lease.
	name := "warehouse:schedule:" + scheduleID
	owner := core.NewID("wh_run")
	acquired, err := s.store.AcquireLease(ctx, name, owner, 6*time.Hour)
	if err != nil || !acquired {
		return nil, err
	}
	return func() {
		if err := s.store.ReleaseLease(context.Background(), name, owner); err != nil {
			log.Printf("topbase: release schedule %s lease: %v", scheduleID, err)
		}
	}, nil
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
			release, err := s.reserveWarehouseRun(ctx, item.ID)
			if err != nil {
				log.Printf("topbase: acquire schedule %s lease: %v", item.ID, err)
				continue
			}
			if release == nil {
				continue
			}
			if _, err := s.warehouse.Run(ctx, item.ID); err != nil {
				log.Printf("topbase: schedule %s: %v", item.ID, err)
			}
			release()
		}
		s.notify.RunDueSubscriptions(ctx, now)
	}
}
