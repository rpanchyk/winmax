package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"winmax/internal/match"
)

type Match struct {
	Condition string
	Title     string
	Process   string
}

type App struct {
	Name  string `yaml:"name"`
	Match Match  `yaml:"match"`
}

type Config struct {
	Apps []App `yaml:"apps"`
}

func (m *Match) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.MappingNode:
		var raw struct {
			Condition string `yaml:"condition"`
			Title     string `yaml:"title"`
			Process   string `yaml:"process"`
		}
		if err := value.Decode(&raw); err != nil {
			return err
		}
		*m = Match{Condition: raw.Condition, Title: raw.Title, Process: raw.Process}
		return nil
	case yaml.SequenceNode:
		for _, item := range value.Content {
			var part map[string]string
			if err := item.Decode(&part); err != nil {
				return fmt.Errorf("match list item: %w", err)
			}
			for key, val := range part {
				switch strings.ToLower(strings.TrimSpace(key)) {
				case "condition":
					m.Condition = val
				case "title":
					m.Title = val
				case "process":
					m.Process = val
				}
			}
		}
		return nil
	default:
		return fmt.Errorf("match must be a map or a list of fields")
	}
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config %s: %w", path, err)
	}
	return &cfg, nil
}

func (c *Config) validate() error {
	if len(c.Apps) == 0 {
		return fmt.Errorf("apps list is empty")
	}
	for i, app := range c.Apps {
		if strings.TrimSpace(app.Name) == "" {
			return fmt.Errorf("apps[%d].name is empty", i)
		}
		if strings.TrimSpace(app.Match.Title) == "" && strings.TrimSpace(app.Match.Process) == "" {
			return fmt.Errorf("apps[%d].match needs title and/or process", i)
		}
		cond := strings.ToUpper(strings.TrimSpace(app.Match.Condition))
		if cond != "" && cond != match.CondAND && cond != match.CondOR {
			return fmt.Errorf("apps[%d].match.condition must be AND or OR", i)
		}
	}
	return nil
}

func (c *Config) Rules() []match.Rule {
	rules := make([]match.Rule, 0, len(c.Apps))
	for _, app := range c.Apps {
		rules = append(rules, match.Rule{
			Name:      strings.TrimSpace(app.Name),
			Condition: app.Match.Condition,
			Title:     app.Match.Title,
			Process:   app.Match.Process,
		})
	}
	return rules
}

func ResolvePath(explicit string) (string, error) {
	if explicit != "" {
		return filepath.Abs(explicit)
	}
	if env := os.Getenv("WINMAX_CONFIG"); env != "" {
		return filepath.Abs(env)
	}

	candidates := make([]string, 0, 3)
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "config.yml"))
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, "config.yml"))
	}
	if appdata := os.Getenv("LOCALAPPDATA"); appdata != "" {
		candidates = append(candidates, filepath.Join(appdata, "winmax", "config.yml"))
	}

	var firstErr error
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return filepath.Abs(p)
		} else if firstErr == nil {
			firstErr = err
		}
	}
	if firstErr == nil {
		firstErr = fmt.Errorf("config.yml not found")
	}
	return "", firstErr
}
