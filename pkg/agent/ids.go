package agent

import (
	"fmt"
	"strings"
	"unicode"
)

func sanitizeID(title string) string {
	title = strings.TrimSpace(strings.ToLower(title))
	var b strings.Builder
	prevUS := false
	for _, r := range title {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevUS = false
		case r == '_' || r == '-':
			if !prevUS && b.Len() > 0 {
				b.WriteByte('_')
				prevUS = true
			}
		case unicode.IsSpace(r) || r == '/' || r == '.':
			if !prevUS && b.Len() > 0 {
				b.WriteByte('_')
				prevUS = true
			}
		}
	}
	out := strings.Trim(b.String(), "_-")
	if out == "" {
		out = "task"
	}
	if len(out) > 40 {
		out = strings.Trim(out[:40], "_-")
	}
	return out
}

func uniqueID(base string, taken func(string) bool) string {
	if base == "" || base == "root" {
		base = "task"
	}
	if !taken(base) {
		return base
	}
	for i := 2; i < 1000; i++ {
		id := fmt.Sprintf("%s_%d", base, i)
		if !taken(id) {
			return id
		}
	}
	return fmt.Sprintf("%s_%d", base, 1000)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func looksLikePath(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || strings.ContainsAny(s, " \t\n") {
		return false
	}
	return strings.Contains(s, "/") || strings.Contains(s, ".") || strings.HasSuffix(s, ".go")
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

func defaultDoDCommands(outputs []string, hasGoMod bool) []string {
	cmds := make([]string, 0, len(outputs)+1)
	for _, out := range outputs {
		if looksLikePath(out) {
			cmds = append(cmds, "test -f "+shellQuote(out))
		}
	}
	if hasGoMod {
		cmds = append(cmds, "go test ./...")
	}
	if len(cmds) == 0 {
		cmds = []string{"ls"}
	}
	return cmds
}
