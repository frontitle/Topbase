package warehouse

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type cronExpr struct {
	minute, hour, day, month, weekday string
}

func ParseCron(expr string) (cronExpr, error) {
	parts := strings.Fields(strings.TrimSpace(expr))
	if len(parts) != 5 {
		return cronExpr{}, fmt.Errorf("cron must have 5 fields")
	}
	return cronExpr{parts[0], parts[1], parts[2], parts[3], parts[4]}, nil
}

func Due(expr, timezone string, now time.Time, last *time.Time) bool {
	parsed, err := ParseCron(expr)
	if err != nil {
		return false
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil || timezone == "" {
		loc, _ = time.LoadLocation("Asia/Shanghai")
	}
	if loc == nil {
		loc = time.UTC
	}
	local := now.In(loc)
	if last != nil {
		prev := last.In(loc)
		if prev.Year() == local.Year() && prev.YearDay() == local.YearDay() && prev.Hour() == local.Hour() && prev.Minute() == local.Minute() {
			return false
		}
	}
	return matchField(parsed.minute, local.Minute()) &&
		matchField(parsed.hour, local.Hour()) &&
		matchField(parsed.day, local.Day()) &&
		matchField(parsed.month, int(local.Month())) &&
		matchField(parsed.weekday, int(local.Weekday()))
}

func matchField(field string, value int) bool {
	if field == "*" {
		return true
	}
	for _, part := range strings.Split(field, ",") {
		n, err := strconv.Atoi(part)
		if err == nil && n == value {
			return true
		}
	}
	return false
}
