package content

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/topbase/topbase/internal/core"
)

func (s Service) CreateDashboard(d core.Dashboard, userID string) (core.Dashboard, error) {
	if strings.TrimSpace(d.Name) == "" {
		d.Name = "新建仪表盘 " + time.Now().Format("01-02 15:04")
	}
	if d.ID == "" {
		d.ID = core.NewID("dsh")
	}
	d.CreatedBy = userID
	d.CreatedAt = time.Now().UTC()
	// Dashboards are private by default. Publishing and embedding are only
	// enabled through their dedicated, permission-checked service methods.
	d.PublicUUID = ""
	d.PublicEmbedEnabled = false
	normalizeDashboard(&d)
	if err := s.Dashboards.Create(d); err != nil {
		return core.Dashboard{}, err
	}
	s.recordRevision("dashboard", d.ID, userID, d)
	return d, nil
}

func (s Service) UpdateDashboard(d core.Dashboard, userID string) (core.Dashboard, error) {
	existing, err := s.Dashboards.ByID(d.ID)
	if err != nil {
		return core.Dashboard{}, err
	}
	if existing.ArchivedAt != nil {
		return core.Dashboard{}, fmt.Errorf("dashboard is archived")
	}
	if strings.TrimSpace(d.Name) == "" {
		d.Name = existing.Name
	}
	if d.PublicUUID == "" {
		d.PublicUUID = existing.PublicUUID
	}
	if d.PublicUUID == "-" {
		d.PublicUUID = ""
	}
	d.CreatedBy = existing.CreatedBy
	d.CreatedAt = existing.CreatedAt
	normalizeDashboard(&d)
	if err := s.Dashboards.Update(d); err != nil {
		return core.Dashboard{}, err
	}
	s.recordRevision("dashboard", d.ID, userID, d)
	return d, nil
}

func (s Service) GetDashboardByPublicUUID(uuid string) (core.Dashboard, error) {
	return s.Dashboards.ByPublicUUID(uuid)
}

func (s Service) EnableDashboardPublicLink(id, userID string) (core.Dashboard, error) {
	d, err := s.Dashboards.ByID(id)
	if err != nil {
		return core.Dashboard{}, err
	}
	if d.PublicUUID == "" {
		d.PublicUUID = core.NewID("pub")
	}
	return s.UpdateDashboard(d, userID)
}

func (s Service) DisableDashboardPublicLink(id, userID string) (core.Dashboard, error) {
	d, err := s.Dashboards.ByID(id)
	if err != nil {
		return core.Dashboard{}, err
	}
	d.PublicUUID = "-"
	d.PublicEmbedEnabled = false
	return s.UpdateDashboard(d, userID)
}

func (s Service) SetDashboardEmbedding(id string, enabled bool, userID string) (core.Dashboard, error) {
	d, err := s.Dashboards.ByID(id)
	if err != nil {
		return core.Dashboard{}, err
	}
	if enabled && d.PublicUUID == "" {
		return core.Dashboard{}, fmt.Errorf("publish the dashboard before enabling embedding")
	}
	d.PublicEmbedEnabled = enabled
	return s.UpdateDashboard(d, userID)
}

func (s Service) DuplicateDashboard(id, userID string) (core.Dashboard, error) {
	src, err := s.GetDashboard(id)
	if err != nil {
		return core.Dashboard{}, err
	}
	tabMap := map[string]string{}
	cardMap := map[string]string{}
	copy := src
	copy.ID = core.NewID("dsh")
	copy.Name = strings.TrimSpace(src.Name) + " 副本"
	copy.PublicUUID = ""
	copy.PublicEmbedEnabled = false
	copy.ArchivedAt = nil
	copy.CreatedBy = userID
	copy.CreatedAt = time.Now().UTC()
	copy.Tabs = make([]core.DashboardTab, len(src.Tabs))
	for i, tab := range src.Tabs {
		nid := core.NewID("tab")
		tabMap[tab.ID] = nid
		tab.ID = nid
		tab.DashboardID = copy.ID
		copy.Tabs[i] = tab
	}
	copy.Cards = make([]core.DashboardCard, len(src.Cards))
	for i, card := range src.Cards {
		nid := core.NewID("crd")
		cardMap[card.ID] = nid
		card.ID = nid
		card.DashboardID = copy.ID
		if mapped, ok := tabMap[card.TabID]; ok {
			card.TabID = mapped
		}
		copy.Cards[i] = card
	}
	copy.Filters = make([]core.DashboardFilter, len(src.Filters))
	for i, filter := range src.Filters {
		filter.ID = core.NewID("flt")
		filter.DashboardID = copy.ID
		for j, mapping := range filter.Mappings {
			if mapped, ok := cardMap[mapping.CardID]; ok {
				filter.Mappings[j].CardID = mapped
			}
		}
		copy.Filters[i] = filter
	}
	normalizeDashboard(&copy)
	if err := s.Dashboards.Create(copy); err != nil {
		return core.Dashboard{}, err
	}
	s.recordRevision("dashboard", copy.ID, userID, copy)
	return copy, nil
}

func (s Service) GetDashboard(id string) (core.Dashboard, error) {
	return s.Dashboards.ByID(id)
}

func (s Service) ListDashboards() ([]core.Dashboard, error) {
	return s.Dashboards.List(false)
}

func (s Service) ArchiveDashboard(id string) error {
	now := time.Now().UTC()
	return s.Dashboards.SetArchived(id, &now)
}

func (s Service) RestoreDashboard(id string) error {
	return s.Dashboards.SetArchived(id, nil)
}

func normalizeDashboard(d *core.Dashboard) {
	if len(d.Tabs) == 0 {
		d.Tabs = []core.DashboardTab{{ID: core.NewID("tab"), Name: "概述", Position: 0}}
	}
	for i := range d.Tabs {
		if d.Tabs[i].ID == "" {
			d.Tabs[i].ID = core.NewID("tab")
		}
		d.Tabs[i].DashboardID = d.ID
		d.Tabs[i].Position = i
	}
	defaultTab := d.Tabs[0].ID
	for i := range d.Cards {
		if d.Cards[i].ID == "" {
			d.Cards[i].ID = core.NewID("crd")
		}
		if d.Cards[i].Type == "" {
			d.Cards[i].Type = "question"
		}
		if d.Cards[i].TabID == "" {
			d.Cards[i].TabID = defaultTab
		}
		if d.Cards[i].Layout.W == 0 {
			d.Cards[i].Layout.W = 4
		}
		if d.Cards[i].Layout.H == 0 {
			d.Cards[i].Layout.H = 4
		}
		d.Cards[i].DashboardID = d.ID
	}
	for i := range d.Filters {
		if d.Filters[i].ID == "" {
			d.Filters[i].ID = core.NewID("flt")
		}
		d.Filters[i].DashboardID = d.ID
	}
}

func (s Service) recordRevision(targetType, targetID, actorID string, snapshot any) {
	if s.Revisions == nil {
		return
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return
	}
	_ = s.Revisions.Create(core.Revision{
		ID: core.NewID("rev"), TargetType: targetType, TargetID: targetID,
		ActorID: actorID, Snapshot: string(raw), CreatedAt: time.Now().UTC(),
	})
}

func (s Service) ListRevisions(targetType, targetID string) ([]core.Revision, error) {
	if s.Revisions == nil {
		return []core.Revision{}, nil
	}
	return s.Revisions.List(targetType, targetID)
}
