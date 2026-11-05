package tui

import (
	"sort"
	"strings"

	"wsdash/internal/project"
)

// fuzzyScore matches pattern against target as a case-insensitive
// subsequence. It returns a score that rewards consecutive and word-start
// matches, so "wsh" ranks "wsdash" above a scattered match.
func fuzzyScore(pattern, target string) (int, bool) {
	if pattern == "" {
		return 0, true
	}
	p := []rune(strings.ToLower(pattern))
	t := []rune(strings.ToLower(target))
	if len(p) > len(t) {
		return 0, false
	}
	score := 0
	pi := 0
	prev := -2
	for ti := 0; ti < len(t) && pi < len(p); ti++ {
		if t[ti] != p[pi] {
			continue
		}
		score++
		if ti == prev+1 {
			score += 2
		}
		if ti == 0 || isWordBreak(t[ti-1]) {
			score += 3
		}
		prev = ti
		pi++
	}
	if pi != len(p) {
		return 0, false
	}
	return score, true
}

func isWordBreak(r rune) bool {
	return r == '/' || r == '-' || r == '_' || r == '.' || r == ' '
}

// projectScore matches the query against the fields a user can see. The name
// is weighted highest, then description, tags and path.
func projectScore(query string, p project.Project) (int, bool) {
	best := 0
	fields := []struct {
		value  string
		weight int
	}{
		{p.Name, 1000},
		{p.Description, 300},
		{strings.Join(p.Tags, " "), 200},
		{p.Path, 100},
	}
	found := false
	for _, f := range fields {
		if f.value == "" {
			continue
		}
		score, ok := fuzzyScore(query, f.value)
		if !ok {
			continue
		}
		if !found || score+f.weight > best {
			best = score + f.weight
			found = true
		}
	}
	return best, found
}

// filteredRows is the flat, score-ordered project list used while searching.
// Search sees every project, including scan-only ones hidden by the f filter.
func (m Model) filteredRows() []row {
	type scored struct {
		index int
		score int
	}
	matches := make([]scored, 0, len(m.projects))
	for i := range m.projects {
		score, ok := projectScore(m.query, m.projects[i])
		if ok {
			matches = append(matches, scored{index: i, score: score})
		}
	}
	sort.SliceStable(matches, func(a, b int) bool {
		if matches[a].score != matches[b].score {
			return matches[a].score > matches[b].score
		}
		an := strings.ToLower(m.projects[matches[a].index].Name)
		bn := strings.ToLower(m.projects[matches[b].index].Name)
		return an < bn
	})
	rows := make([]row, len(matches))
	for i, match := range matches {
		rows[i] = row{kind: rowProject, project: match.index}
	}
	return rows
}
