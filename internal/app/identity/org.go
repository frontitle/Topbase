package identity

import (
	"context"
	"fmt"
	"strings"

	"github.com/topbase/topbase/internal/core"
)

func (s Service) ListGroups() ([]core.Group, error) {
	if s.Groups == nil {
		return []core.Group{}, nil
	}
	groups, err := s.Groups.List()
	if err != nil {
		return nil, err
	}
	if s.Users == nil {
		return groups, nil
	}
	users, err := s.Users.List()
	if err != nil {
		return nil, err
	}
	for i := range groups {
		for _, user := range users {
			ok, _ := s.Groups.HasMember(groups[i].ID, user.ID)
			if ok {
				groups[i].MemberIDs = append(groups[i].MemberIDs, user.ID)
			}
		}
	}
	return groups, nil
}

func (s Service) SyncDirectory(ctx context.Context, directory core.OrgDirectory) ([]core.Group, error) {
	if s.Groups == nil {
		return nil, fmt.Errorf("group store is not configured")
	}
	if directory == nil {
		return nil, fmt.Errorf("organization directory credentials are required")
	}
	units, err := directory.ListUnits(ctx)
	if err != nil {
		return nil, err
	}
	saved := []core.Group{}
	for _, unit := range units {
		id := feishuGroupID(unit.ExternalID)
		group := core.Group{ID: id, Name: unit.Name, Kind: "feishu_dept"}
		if err := s.Groups.Upsert(group); err != nil {
			return nil, err
		}
		userIDs := []string{}
		for _, openID := range unit.MemberIDs {
			if s.Users == nil {
				break
			}
			user, err := s.Users.ByFeishuOpenID(openID)
			if err != nil {
				continue
			}
			userIDs = append(userIDs, user.ID)
		}
		if err := s.Groups.ReplaceMembers(id, userIDs); err != nil {
			return nil, err
		}
		saved = append(saved, group)
	}
	return saved, nil
}

func feishuGroupID(externalID string) string {
	clean := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' {
			return r
		}
		return '_'
	}, externalID)
	if clean == "" {
		clean = "dept"
	}
	return "grp_fs_" + clean
}
