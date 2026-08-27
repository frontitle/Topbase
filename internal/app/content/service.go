package content

import (
	"fmt"
	"strings"
	"time"

	"github.com/topbase/topbase/internal/core"
)

type Service struct {
	Collections   core.CollectionStore
	Questions     core.QuestionStore
	Dashboards    core.DashboardStore
	Bookmarks     core.BookmarkStore
	Revisions     core.RevisionStore
	Alerts        core.AlertStore
	Notifications core.NotificationStore
	SearchStore   core.SearchStore
	Fields        core.FieldStore
	Models        core.ModelStore
	Metrics       core.MetricStore
	Segments      core.SegmentStore
	Glossary      core.GlossaryStore
}

func (s Service) EnsurePersonalCollection(user core.User) (core.Collection, error) {
	items, err := s.Collections.List()
	if err != nil {
		return core.Collection{}, err
	}
	for _, item := range items {
		if item.PersonalOwnerUserID == user.ID {
			return item, nil
		}
	}
	collection := core.Collection{
		ID: core.NewID("col"), Name: "我的分析", PersonalOwnerUserID: user.ID, Kind: "personal_project", CreatedAt: time.Now().UTC(),
	}
	if err := s.Collections.Create(collection); err != nil {
		return core.Collection{}, err
	}
	return collection, nil
}

func (s Service) CreateQuestion(q core.Question, userID string) (core.Question, error) {
	if strings.TrimSpace(q.Name) == "" {
		return core.Question{}, fmt.Errorf("name is required")
	}
	if q.QueryType == "" {
		if q.NativeSQL != "" {
			q.QueryType = "native"
		} else {
			q.QueryType = "queryir"
		}
	}
	if q.QueryType == "queryir" {
		if q.QueryIR == nil {
			return core.Question{}, fmt.Errorf("queryir is required")
		}
		if err := q.QueryIR.Validate(); err != nil {
			return core.Question{}, err
		}
		q.DatabaseID = q.QueryIR.Source.DatabaseID
	}
	if q.ID == "" {
		q.ID = core.NewID("qst")
	}
	q.CreatedBy = userID
	q.CreatedAt = time.Now().UTC()
	if err := s.Questions.Create(q); err != nil {
		return core.Question{}, err
	}
	s.recordRevision("question", q.ID, userID, q)
	return q, nil
}

func (s Service) ArchiveQuestion(id string) error {
	now := time.Now().UTC()
	return s.Questions.SetArchived(id, &now)
}

func (s Service) RestoreQuestion(id string) error {
	return s.Questions.SetArchived(id, nil)
}

func (s Service) ListQuestions() ([]core.Question, error) {
	return s.Questions.List(false)
}

func (s Service) GetQuestion(id string) (core.Question, error) {
	return s.Questions.ByID(id)
}

func (s Service) UpdateQuestion(patch core.Question) (core.Question, error) {
	existing, err := s.Questions.ByID(patch.ID)
	if err != nil {
		return core.Question{}, err
	}
	if strings.TrimSpace(patch.Name) != "" {
		existing.Name = patch.Name
	}
	existing.Description = patch.Description
	existing.CollectionID = patch.CollectionID
	if patch.ChartSpec != nil {
		existing.ChartSpec = patch.ChartSpec
	}
	if err := s.Questions.Update(existing); err != nil {
		return core.Question{}, err
	}
	return existing, nil
}

func (s Service) ListCollections() ([]core.Collection, error) {
	return s.Collections.List()
}

func (s Service) CreateCollection(name, parentID, ownerID, ownerGroupID string) (core.Collection, error) {
	if strings.TrimSpace(name) == "" {
		return core.Collection{}, fmt.Errorf("name is required")
	}
	kind := "team_project"
	if ownerID != "" {
		kind = "personal_project"
	}
	if parentID != "" {
		parent, err := s.Collections.ByID(parentID)
		if err != nil {
			return core.Collection{}, err
		}
		kind, ownerID, ownerGroupID = parent.Kind, parent.PersonalOwnerUserID, parent.OwnerGroupID
	}
	collection := core.Collection{ID: core.NewID("col"), ParentID: parentID, Name: name, PersonalOwnerUserID: ownerID, OwnerGroupID: ownerGroupID, Kind: kind, CreatedAt: time.Now().UTC()}
	if err := s.Collections.Create(collection); err != nil {
		return core.Collection{}, err
	}
	return collection, nil
}

func (s Service) UpdateCollection(id, name, parentID string) (core.Collection, error) {
	existing, err := s.Collections.ByID(id)
	if err != nil {
		return core.Collection{}, err
	}
	if existing.PersonalOwnerUserID != "" && parentID != existing.ParentID {
		return core.Collection{}, fmt.Errorf("个人数据组不能移动")
	}
	if strings.TrimSpace(name) != "" {
		existing.Name = name
	}
	existing.ParentID = parentID
	if err := s.Collections.Update(existing); err != nil {
		return core.Collection{}, err
	}
	return existing, nil
}

func (s Service) DeleteCollection(id string) error {
	existing, err := s.Collections.ByID(id)
	if err != nil {
		return err
	}
	if existing.PersonalOwnerUserID != "" {
		return fmt.Errorf("个人数据组不能删除")
	}
	questions, err := s.Questions.List(true)
	if err != nil {
		return err
	}
	for _, item := range questions {
		if item.CollectionID == id {
			return fmt.Errorf("请先把数据组里的分析移走或归档")
		}
	}
	items, err := s.Collections.List()
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.ParentID == id {
			return fmt.Errorf("请先移走或删除子数据组")
		}
	}
	return s.Collections.Delete(id)
}
