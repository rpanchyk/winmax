package match

import (
	"path/filepath"
	"regexp"
	"strings"
)

const (
	CondAND = "AND"
	CondOR  = "OR"
)

type Rule struct {
	Name      string
	Condition string
	Title     string
	Process   string
}

func (r Rule) String() string {
	cond := r.normalizedCondition()
	return r.displayName() + " [" + cond + " title=" + r.Title + " process=" + r.Process + "]"
}

func (r Rule) displayName() string {
	if r.Name != "" {
		return r.Name
	}
	return "(unnamed)"
}

func (r Rule) normalizedCondition() string {
	switch strings.ToUpper(strings.TrimSpace(r.Condition)) {
	case CondOR:
		return CondOR
	default:
		return CondAND
	}
}

func Any(rules []Rule, title, processPath string) bool {
	_, ok := First(rules, title, processPath)
	return ok
}

func First(rules []Rule, title, processPath string) (Rule, bool) {
	for _, rule := range rules {
		if rule.Match(title, processPath) {
			return rule, true
		}
	}
	return Rule{}, false
}

func (r Rule) Match(title, processPath string) bool {
	titlePat := strings.TrimSpace(r.Title)
	procPat := strings.TrimSpace(r.Process)
	if titlePat == "" && procPat == "" {
		return false
	}

	titleOK := titlePat != "" && wildcard(titlePat, title)
	procOK := procPat != "" && matchProcess(procPat, processPath)

	if r.normalizedCondition() == CondOR && titlePat != "" && procPat != "" {
		return titleOK || procOK
	}
	if titlePat != "" && !titleOK {
		return false
	}
	if procPat != "" && !procOK {
		return false
	}
	return true
}

func matchProcess(pattern, processPath string) bool {
	return wildcard(pattern, processPath) || wildcard(pattern, filepath.Base(processPath))
}

func wildcard(pattern, value string) bool {
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	value = strings.ToLower(value)
	if pattern == "" {
		return false
	}
	if !strings.ContainsAny(pattern, "*?") {
		return strings.Contains(value, pattern)
	}
	re, err := globRegexp(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(value)
}

func globRegexp(pattern string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString("(?i)^")
	for _, r := range pattern {
		switch r {
		case '*':
			b.WriteString(".*")
		case '?':
			b.WriteByte('.')
		default:
			b.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	b.WriteByte('$')
	return regexp.Compile(b.String())
}
