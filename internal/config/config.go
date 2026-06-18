package config

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Options struct {
	Path               string
	DisableEnv         bool
	DisableDefaultPath bool
}

type Config struct {
	Search     SearchConfig     `yaml:"search" json:"search"`
	Bocha      BochaConfig      `yaml:"bocha" json:"bocha"`
	Volcengine VolcengineConfig `yaml:"volcengine" json:"volcengine"`
	Zhihu      ZhihuConfig      `yaml:"zhihu" json:"zhihu"`
	Output     OutputConfig     `yaml:"output" json:"output"`
}

type SearchConfig struct {
	DefaultSourceIDs []string `yaml:"default_source_ids" json:"default_source_ids"`
	Limit            int      `yaml:"limit" json:"limit"`
	Timeout          string   `yaml:"timeout" json:"timeout"`
	Output           string   `yaml:"output" json:"output"`
	Language         string   `yaml:"language" json:"language"`
}

type BochaConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	APIKey   string `yaml:"api_key" json:"api_key"`
	Endpoint string `yaml:"endpoint" json:"endpoint"`
}

type VolcengineConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	APIKey   string `yaml:"api_key" json:"api_key"`
	Model    string `yaml:"model" json:"model"`
	Endpoint string `yaml:"endpoint" json:"endpoint"`
}

type ZhihuConfig struct {
	Enabled      bool   `yaml:"enabled" json:"enabled"`
	AccessSecret string `yaml:"access_secret" json:"access_secret"`
	EndpointBase string `yaml:"endpoint_base" json:"endpoint_base"`
}

type OutputConfig struct {
	ShowSource      bool `yaml:"show_source" json:"show_source"`
	ShowURL         bool `yaml:"show_url" json:"show_url"`
	ShowPublishedAt bool `yaml:"show_published_at" json:"show_published_at"`
}

func Defaults() Config {
	return Config{
		Search: SearchConfig{
			DefaultSourceIDs: []string{"bocha_web", "volcengine_answer", "zhihu_search"},
			Limit:            10,
			Timeout:          "45s",
			Output:           "table",
			Language:         "zh-CN",
		},
		Bocha: BochaConfig{
			Enabled:  true,
			Endpoint: "https://api.bocha.cn/v1/web-search",
		},
		Volcengine: VolcengineConfig{
			Enabled:  true,
			Model:    "doubao-seed-2-0-lite-260215",
			Endpoint: "https://ark.cn-beijing.volces.com/api/v3/responses",
		},
		Zhihu: ZhihuConfig{
			Enabled:      true,
			EndpointBase: "https://developer.zhihu.com/api/v1/content",
		},
		Output: OutputConfig{
			ShowSource:      true,
			ShowURL:         true,
			ShowPublishedAt: true,
		},
	}
}

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "findo", "config.yaml"), nil
}

func Load(opts Options) (Config, error) {
	cfg := Defaults()
	path := opts.Path
	if path == "" && !opts.DisableDefaultPath {
		defaultPath, err := DefaultPath()
		if err == nil {
			if _, err := os.Stat(defaultPath); err == nil {
				path = defaultPath
			} else if !errors.Is(err, os.ErrNotExist) {
				return cfg, err
			}
		}
	}
	if path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return cfg, err
		}
		if err := yaml.Unmarshal(b, &cfg); err != nil {
			return cfg, err
		}
	}
	if !opts.DisableEnv {
		applyEnv(&cfg)
	}
	if cfg.Search.Limit <= 0 || cfg.Search.Limit > 50 {
		return cfg, errors.New("search.limit must be 1..50")
	}
	return cfg, nil
}

func Init(path string, force bool) (string, error) {
	if path == "" {
		defaultPath, err := DefaultPath()
		if err != nil {
			return "", err
		}
		path = defaultPath
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return "", err
	}
	if !force {
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			if errors.Is(err, os.ErrExist) {
				return "", os.ErrExist
			}
			return "", err
		}
		if _, err := file.WriteString(DefaultYAML()); err != nil {
			_ = file.Close()
			return "", err
		}
		if err := file.Close(); err != nil {
			return "", err
		}
		return path, nil
	}
	if err := os.WriteFile(path, []byte(DefaultYAML()), 0600); err != nil {
		return "", err
	}
	if err := os.Chmod(path, 0600); err != nil {
		return "", err
	}
	return path, nil
}

func DefaultYAML() string {
	return `search:
  default_source_ids:
    - bocha_web
    - volcengine_answer
    - zhihu_search
  limit: 10
  timeout: 45s
  output: table
  language: zh-CN

bocha:
  enabled: true
  api_key: ""
  endpoint: https://api.bocha.cn/v1/web-search

volcengine:
  enabled: true
  api_key: ""
  model: doubao-seed-2-0-lite-260215
  endpoint: https://ark.cn-beijing.volces.com/api/v3/responses

zhihu:
  enabled: true
  access_secret: ""
  endpoint_base: https://developer.zhihu.com/api/v1/content

output:
  show_source: true
  show_url: true
  show_published_at: true
`
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("BOCHA_API_KEY"); v != "" {
		cfg.Bocha.APIKey = v
	}
	if v := os.Getenv("VOLCENGINE_API_KEY"); v != "" {
		cfg.Volcengine.APIKey = v
	} else if v := os.Getenv("ARK_API_KEY"); v != "" {
		cfg.Volcengine.APIKey = v
	}
	if v := os.Getenv("VOLCENGINE_MODEL"); v != "" {
		cfg.Volcengine.Model = v
	}
	if v := os.Getenv("ZHIHU_ACCESS_SECRET"); v != "" {
		cfg.Zhihu.AccessSecret = v
	} else if v := os.Getenv("ZHIHU_API_KEY"); v != "" {
		cfg.Zhihu.AccessSecret = v
	}
}

func (c Config) Redacted() Config {
	c.Bocha.APIKey = redact(c.Bocha.APIKey)
	c.Volcengine.APIKey = redact(c.Volcengine.APIKey)
	c.Zhihu.AccessSecret = redact(c.Zhihu.AccessSecret)
	return c
}

func redact(v string) string {
	if v == "" {
		return ""
	}
	return "***"
}
