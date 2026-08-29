package appdb

import (
	"context"
	"fmt"
	"time"
)

func (s *Store) AcquireLease(ctx context.Context, name, owner string, ttl time.Duration) (bool, error) {
	if name == "" || owner == "" || ttl <= 0 {
		return false, fmt.Errorf("lease name, owner and positive ttl are required")
	}
	now := time.Now().UTC()
	expires := now.Add(ttl).Format(time.RFC3339Nano)
	updated := now.Format(time.RFC3339Nano)
	result, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO distributed_leases(name, owner, expires_at, updated_at) VALUES(?,?,?,?)`,
		name, owner, expires, updated,
	)
	if err != nil {
		return false, err
	}
	if affected, _ := result.RowsAffected(); affected == 1 {
		return true, nil
	}
	result, err = s.db.ExecContext(ctx,
		`UPDATE distributed_leases SET owner=?, expires_at=?, updated_at=? WHERE name=? AND (expires_at<? OR owner=?)`,
		owner, expires, updated, name, updated, owner,
	)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	return affected == 1, err
}

func (s *Store) ReleaseLease(ctx context.Context, name, owner string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM distributed_leases WHERE name=? AND owner=?`, name, owner)
	return err
}
