package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// FileName is the project-level ForgeKit config file.
const FileName = "forge.yaml"

// Config holds ForgeKit project settings.
type Config struct {
	Architecture ArchitectureConfig `yaml:"architecture"`
	Project      ProjectConfig        `yaml:"project"`
}

type ProjectConfig struct {
	Language string `yaml:"language"`
	Module   string `yaml:"module"`
}

type ArchitectureConfig struct {
	Rules []ArchitectureRule `yaml:"rules"`
}

type ArchitectureRule struct {
	From  string `yaml:"from"`
	To    string `yaml:"to"`
	Allow bool   `yaml:"allow"`
}

// Global holds user-wide defaults (~/.config/forge/config.yaml).
type Global struct {
	Author string `yaml:"author"`
}

// Merge applies priority: CLI > env > project > global > defaults.
type Merge struct {
	Author string
}

// DefaultArchitectureRules returns ForgeKit reference layer rules.
func DefaultArchitectureRules() []ArchitectureRule {
	return []ArchitectureRule{
		{From: "domain", To: "infrastructure", Allow: false},
		{From: "domain", To: "transport", Allow: false},
		{From: "application", To: "transport", Allow: false},
		{From: "application", To: "infrastructure", Allow: true},
		{From: "transport", To: "application", Allow: true},
		{From: "transport", To: "infrastructure", Allow: true},
	}
}

// LoadProject reads forge.yaml from project root if present.
func LoadProject(root string) (Config, error) {
	path := filepath.Join(root, FileName)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Config{
			Architecture: ArchitectureConfig{Rules: DefaultArchitectureRules()},
		}, nil
	}
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if len(cfg.Architecture.Rules) == 0 {
		cfg.Architecture.Rules = DefaultArchitectureRules()
	}
	return cfg, nil
}

// LoadGlobal reads ~/.config/forge/config.yaml when available.
func LoadGlobal() Global {
	home, err := os.UserHomeDir()
	if err != nil {
		return Global{}
	}
	path := filepath.Join(home, ".config", "forge", "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return Global{}
	}
	var g Global
	_ = yaml.Unmarshal(data, &g)
	return g
}

// ResolveAuthor picks author from merge priority.
func ResolveAuthor(cliAuthor string, projectDir string) string {
	if cliAuthor != "" {
		return cliAuthor
	}
	if v := os.Getenv("FORGE_AUTHOR"); v != "" {
		return v
	}
	g := LoadGlobal()
	if g.Author != "" {
		return g.Author
	}
	return ""
}
