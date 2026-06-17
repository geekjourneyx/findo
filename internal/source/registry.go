package source

import "github.com/geekjourneyx/findo/internal/search"

type Info struct {
	Source         search.SourceID     `json:"source"`
	ProviderFamily string              `json:"provider_family"`
	Capabilities   []search.Capability `json:"capabilities"`
	Enabled        bool                `json:"enabled"`
	Configured     bool                `json:"configured"`
}

func StaticInfos() []Info {
	return []Info{
		{Source: search.SourceBochaWeb, ProviderFamily: "bocha", Capabilities: []search.Capability{search.CapabilityWebSearch}, Enabled: true},
		{Source: search.SourceVolcengineAnswer, ProviderFamily: "volcengine", Capabilities: []search.Capability{search.CapabilityAnswer}, Enabled: true},
		{Source: search.SourceZhihuSearch, ProviderFamily: "zhihu", Capabilities: []search.Capability{search.CapabilityWebSearch}, Enabled: true},
		{Source: search.SourceZhihuWeb, ProviderFamily: "zhihu", Capabilities: []search.Capability{search.CapabilityWebSearch}, Enabled: true},
		{Source: search.SourceZhihuHot, ProviderFamily: "zhihu", Capabilities: []search.Capability{search.CapabilityHotlist}, Enabled: true},
	}
}
