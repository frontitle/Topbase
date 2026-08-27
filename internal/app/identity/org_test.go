package identity

import (
	"context"
	"strings"
	"testing"

	"github.com/topbase/topbase/internal/core"
)

type memUsers struct{ items map[string]core.User }

func (m *memUsers) Create(user core.User) error {
	if m.items == nil {
		m.items = map[string]core.User{}
	}
	m.items[user.ID] = user
	return nil
}
func (m *memUsers) ByEmail(string) (core.User, error) { return core.User{}, core.ErrNotFound }
func (m *memUsers) ByID(id string) (core.User, error) {
	user, ok := m.items[id]
	if !ok {
		return core.User{}, core.ErrNotFound
	}
	return user, nil
}
func (m *memUsers) ByFeishuOpenID(openID string) (core.User, error) {
	for _, user := range m.items {
		if user.FeishuOpenID == openID {
			return user, nil
		}
	}
	return core.User{}, core.ErrNotFound
}
func (m *memUsers) List() ([]core.User, error)       { return nil, nil }
func (m *memUsers) SetActive(string, bool) error     { return nil }
func (m *memUsers) SetPassword(string, string) error { return nil }

type memGroups struct {
	items   map[string]core.Group
	members map[string][]string
}

func (m *memGroups) Create(group core.Group) error { return m.Upsert(group) }
func (m *memGroups) AddMember(groupID, userID string) error {
	if m.members == nil {
		m.members = map[string][]string{}
	}
	m.members[groupID] = append(m.members[groupID], userID)
	return nil
}
func (m *memGroups) List() ([]core.Group, error) {
	out := []core.Group{}
	for _, item := range m.items {
		out = append(out, item)
	}
	return out, nil
}
func (m *memGroups) Upsert(group core.Group) error {
	if m.items == nil {
		m.items = map[string]core.Group{}
	}
	m.items[group.ID] = group
	return nil
}
func (m *memGroups) ReplaceMembers(groupID string, userIDs []string) error {
	if m.members == nil {
		m.members = map[string][]string{}
	}
	copied := append([]string{}, userIDs...)
	m.members[groupID] = copied
	return nil
}
func (m *memGroups) HasMember(groupID, userID string) (bool, error) {
	for _, id := range m.members[groupID] {
		if id == userID {
			return true, nil
		}
	}
	return false, nil
}
func (m *memGroups) GroupsForUser(userID string) ([]core.Group, error) {
	items := []core.Group{}
	for groupID, members := range m.members {
		for _, id := range members {
			if id == userID {
				if group, ok := m.items[groupID]; ok {
					items = append(items, group)
				}
				break
			}
		}
	}
	return items, nil
}

type fakeDirectory struct{ units []core.OrgUnit }

func (f fakeDirectory) ListUnits(context.Context) ([]core.OrgUnit, error) { return f.units, nil }

func TestSyncDirectoryCreatesFeishuGroups(t *testing.T) {
	users := &memUsers{items: map[string]core.User{
		"usr_1": {ID: "usr_1", Name: "Ada", FeishuOpenID: "ou_ada"},
	}}
	groups := &memGroups{items: map[string]core.Group{
		"grp_all_users": {ID: "grp_all_users", Name: "所有人", Kind: "all_users"},
		"grp_admins":    {ID: "grp_admins", Name: "管理员", Kind: "admins"},
	}}
	svc := Service{Users: users, Groups: groups}
	saved, err := svc.SyncDirectory(context.Background(), fakeDirectory{units: []core.OrgUnit{
		{ExternalID: "od_sales", Name: "销售部", MemberIDs: []string{"ou_ada", "ou_missing"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(saved) != 1 || saved[0].Kind != "feishu_dept" || !strings.HasPrefix(saved[0].ID, "grp_fs_") {
		t.Fatalf("%+v", saved)
	}
	if groups.items["grp_all_users"].Kind != "all_users" || groups.items["grp_admins"].Kind != "admins" {
		t.Fatal("built-in groups must stay unchanged")
	}
	if got := groups.members[saved[0].ID]; len(got) != 1 || got[0] != "usr_1" {
		t.Fatalf("members %+v", got)
	}
}

func TestSyncDirectoryRequiresCredentials(t *testing.T) {
	svc := Service{Groups: &memGroups{}}
	if _, err := svc.SyncDirectory(context.Background(), nil); err == nil {
		t.Fatal("expected credentials error")
	}
}
