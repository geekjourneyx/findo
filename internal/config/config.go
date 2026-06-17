package config

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

type Options struct {
	Path       string
	DisableEnv bool
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

func Load(opts Options) (Config, error) {
	cfg := Defaults()
	if opts.Path != "" {
		b, err := os.ReadFile(opts.Path)
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
