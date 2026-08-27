package warehouse

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/topbase/topbase/internal/core"
)

var hourPattern = regexp.MustCompile(`(\d{1,2})\s*点`)

func Propose(question core.Question, message string) core.ScheduleProposal {
	cron := "0 9 * * *"
	rationale := "默认每天 09:00（Asia/Shanghai）写入数仓。"
	msg := strings.TrimSpace(message)
	switch {
	case strings.Contains(msg, "每小时"):
		cron = "0 * * * *"
		rationale = "按描述改为每小时整点运行。"
	case strings.Contains(msg, "每周"):
		cron = "0 9 * * 1"
		rationale = "按描述改为每周一 09:00 运行。"
	default:
		if hour := parseHour(msg); hour >= 0 {
			cron = "0 " + strconv.Itoa(hour) + " * * *"
			rationale = "按描述改为每天 " + strconv.Itoa(hour) + ":00 运行。"
		}
	}
	table := slug(question.Name)
	if table == "" {
		table = "query"
	}
	proposal := core.ScheduleProposal{
		Name: question.Name + " 数仓", QuestionID: question.ID,
		Cron: cron, Timezone: "Asia/Shanghai",
		MaterializeTo: "warehouse.wh_" + table, Strategy: "replace",
		RequiresConfirm: true, Rationale: rationale,
	}
	if strings.Contains(msg, "增量") {
		proposal.Strategy = "incremental"
		proposal.WatermarkField = "created_at"
		proposal.Rationale += " 使用 created_at 增量写入。"
	}
	return proposal
}

func parseHour(message string) int {
	if m := hourPattern.FindStringSubmatch(message); len(m) == 2 {
		n, _ := strconv.Atoi(m[1])
		if n >= 0 && n <= 23 {
			return n
		}
	}
	return -1
}

func slug(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if unicode.IsLetter(r) && r < 128 || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' {
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "result"
	}
	return out
}
