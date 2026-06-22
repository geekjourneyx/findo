package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/geekjourneyx/tanso/internal/config"
	"github.com/geekjourneyx/tanso/internal/output"
	"github.com/geekjourneyx/tanso/internal/search"
	"github.com/geekjourneyx/tanso/internal/skillcontent"
	sourcepkg "github.com/geekjourneyx/tanso/internal/source"
	"github.com/geekjourneyx/tanso/internal/source/bocha"
	"github.com/geekjourneyx/tanso/internal/source/volcengine"
	"github.com/geekjourneyx/tanso/internal/source/zhihu"
	"github.com/geekjourneyx/tanso/internal/tansoerr"
)

const (
	ExitOK              = 0
	ExitPartial         = 1
	ExitInvalidArgument = 2
	ExitConfig          = 3
	ExitCredential      = 4
	ExitSource          = 5
	ExitTimeout         = 6
	ExitNoResults       = 7
	ExitInternal        = 9
)

type parsed struct {
	Command      string
	Positionals  []string
	JSON         bool
	Markdown     bool
	Table        bool
	Raw          bool
	Filter       string
	SearchDB     string
	Timeout      string
	SourceIDs    []string
	Limit        int
	LimitSet     bool
	ConfigPath   string
	Path         string
	Force        bool
	Help         bool
	Version      bool
	NoColor      bool
	Verbose      bool
	UnknownFlags []string
}

func Run(args []string, version string, stdout, stderr io.Writer) int {
	p, err := parse(args)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return ExitInvalidArgument
	}
	if p.Version && p.Command == "" {
		return runVersion(p, version, stdout, stderr)
	}
	if p.Help {
		return runHelp(p, stdout, stderr)
	}
	if p.Filter != "" && !isZhihuWebCommand(p) {
		_, _ = fmt.Fprintln(stderr, "--filter is only valid for tanso zhihu web")
		return ExitInvalidArgument
	}
	if p.SearchDB != "" && !isZhihuWebCommand(p) {
		_, _ = fmt.Fprintln(stderr, "--search-db is only valid for tanso zhihu web")
		return ExitInvalidArgument
	}
	if err := validateOutputModes(p); err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return ExitInvalidArgument
	}
	if err := validateSourceFlag(p); err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return ExitInvalidArgument
	}
	if err := validateTimeout(p); err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return ExitInvalidArgument
	}
	if p.LimitSet && (p.Limit <= 0 || p.Limit > 50) {
		_, _ = fmt.Fprintln(stderr, "--limit must be 1..50")
		return ExitInvalidArgument
	}
	if (p.Path != "" || p.Force) && !isConfigInitCommand(p) {
		_, _ = fmt.Fprintln(stderr, "--path and --force are only valid for tanso config init")
		return ExitInvalidArgument
	}
	if len(args) == 0 || p.Command == "help" {
		return runHelp(p, stdout, stderr)
	}
	if p.Command == "version" {
		return runVersion(p, version, stdout, stderr)
	}
	if p.Command == "sources" {
		if err := validateSources(p); err != nil {
			_, _ = fmt.Fprintln(stderr, err.Error())
			return ExitInvalidArgument
		}
		if p.JSON {
			cfg, err := config.Load(config.Options{Path: p.ConfigPath})
			if err != nil {
				return writeCommandError(stdout, stderr, p, tansoerr.ConfigInvalid, err.Error(), ExitConfig)
			}
			_ = output.WriteJSON(stdout, map[string]any{"version": version, "sources": sourcepkg.Infos(cfg)})
			return ExitOK
		}
		writeSourcesText(stdout)
		return ExitOK
	}
	if p.Command == "init" {
		p.Command = "config"
		p.Positionals = append([]string{"init"}, p.Positionals...)
		return runConfig(p, stdout, stderr)
	}
	if p.Command == "skills" {
		return runSkills(p, version, stdout, stderr)
	}
	if p.Command == "config" {
		return runConfig(p, stdout, stderr)
	}
	if isRetrievalCommand(p) {
		return runRetrieval(p, version, stdout, stderr)
	}
	if isGenericCommand(p) {
		return runGenericRetrieval(p, version, stdout, stderr)
	}

	_, _ = fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
	return ExitInvalidArgument
}

func parse(args []string) (parsed, error) {
	p := parsed{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			p.Help = true
		case "--version":
			p.Version = true
		case "--json":
			p.JSON = true
		case "--markdown":
			p.Markdown = true
		case "--table":
			p.Table = true
		case "--raw":
			p.Raw = true
		case "--filter":
			if i+1 >= len(args) {
				return p, fmt.Errorf("--filter requires a value")
			}
			p.Filter = args[i+1]
			i++
		case "--search-db":
			if i+1 >= len(args) {
				return p, fmt.Errorf("--search-db requires a value")
			}
			p.SearchDB = args[i+1]
			i++
		case "--limit":
			if i+1 >= len(args) {
				return p, fmt.Errorf("--limit requires a value")
			}
			limit, err := strconv.Atoi(args[i+1])
			if err != nil {
				return p, fmt.Errorf("--limit must be an integer")
			}
			p.Limit = limit
			p.LimitSet = true
			i++
		case "--timeout":
			if i+1 >= len(args) {
				return p, fmt.Errorf("--timeout requires a value")
			}
			p.Timeout = args[i+1]
			i++
		case "--source":
			if i+1 >= len(args) {
				return p, fmt.Errorf("--source requires a value")
			}
			for _, item := range strings.Split(args[i+1], ",") {
				item = strings.TrimSpace(item)
				if item != "" {
					p.SourceIDs = append(p.SourceIDs, item)
				}
			}
			i++
		case "--config":
			if i+1 >= len(args) {
				return p, fmt.Errorf("--config requires a value")
			}
			p.ConfigPath = args[i+1]
			i++
		case "--path":
			if i+1 >= len(args) {
				return p, fmt.Errorf("--path requires a value")
			}
			p.Path = args[i+1]
			i++
		case "--force":
			p.Force = true
		case "--no-color":
			p.NoColor = true
		case "--verbose":
			p.Verbose = true
		default:
			if len(args[i]) > 0 && args[i][0] == '-' {
				p.UnknownFlags = append(p.UnknownFlags, args[i])
				continue
			}
			if p.Command == "" {
				p.Command = args[i]
			} else {
				p.Positionals = append(p.Positionals, args[i])
			}
		}
	}
	return p, nil
}

func runVersion(p parsed, version string, stdout, stderr io.Writer) int {
	if err := validateVersion(p); err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return ExitInvalidArgument
	}
	if p.JSON {
		_, _ = fmt.Fprintf(stdout, `{"version":%q}`+"\n", version)
		return ExitOK
	}
	_, _ = fmt.Fprintf(stdout, "tanso %s\n", version)
	return ExitOK
}

func runHelp(p parsed, stdout, stderr io.Writer) int {
	if err := validateHelp(p); err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return ExitInvalidArgument
	}
	topic := helpTopic(p)
	_, _ = io.WriteString(stdout, helpText(topic))
	return ExitOK
}

func helpTopic(p parsed) string {
	if p.Command == "help" {
		return strings.Join(p.Positionals, " ")
	}
	parts := []string{}
	if p.Command != "" {
		parts = append(parts, p.Command)
	}
	parts = append(parts, p.Positionals...)
	return strings.Join(parts, " ")
}

func helpText(topic string) string {
	switch topic {
	case "bocha":
		return `Usage:
  tanso bocha <query> [--json|--markdown|--table|--raw] [--limit 1..50] [--timeout duration] [--config path]

Search broad web evidence through Bocha.
`
	case "volc", "volc answer":
		return `Usage:
  tanso volc <query> [--json|--markdown|--table|--raw] [--limit 1..50] [--timeout duration] [--config path]
  tanso volc answer <query> [flags]

Return a web-grounded answer through Volcengine Ark.
`
	case "zhihu":
		return `Usage:
  tanso zhihu <query> [--json|--markdown|--table|--raw] [--limit 1..50] [--timeout duration] [--config path]
  tanso zhihu web <query> [--filter string] [--search-db all|realtime|static] [flags]
  tanso zhihu hot [--json|--markdown|--table|--raw] [--limit 1..50] [flags]

Search Zhihu discussions, Zhihu-backed web results, or the current hotlist.
`
	case "zhihu web":
		return `Usage:
  tanso zhihu web <query> [--filter string] [--search-db all|realtime|static] [--json|--markdown|--table|--raw] [--limit 1..50] [--timeout duration] [--config path]

Search the web through Zhihu's global search API.
`
	case "zhihu hot", "hot zhihu":
		return `Usage:
  tanso zhihu hot [--json|--markdown|--table|--raw] [--limit 1..50] [--timeout duration] [--config path]

Return the current Zhihu hotlist.
`
	case "sources":
		return `Usage:
  tanso sources [--json] [--config path]

List source inventory. In JSON output, configured means local credential material is present; it is not a live authentication check.
`
	case "config":
		return `Usage:
  tanso config init [--path path] [--force]
  tanso config path
  tanso config show --json [--config path]

Create, locate, or inspect the resolved Tanso config.
`
	case "config init", "init":
		return `Usage:
  tanso config init [--path path] [--force]
  tanso init [--path path] [--force]

Create the default config file. Parent directories are created automatically.
`
	case "config path":
		return `Usage:
  tanso config path

Print the resolved config path.
`
	case "config show":
		return `Usage:
  tanso config show --json [--config path]

Print the merged config with secrets redacted.
`
	case "skills":
		return `Usage:
  tanso skills list [--json]
  tanso skills read <name>[/<path>] [path] [--json]

Read Agent Skill instructions bundled with the installed CLI.
`
	case "skills list":
		return `Usage:
  tanso skills list [--json]

List bundled Agent Skills.
`
	case "skills read":
		return `Usage:
  tanso skills read <name>[/<path>] [path] [--json]

Read bundled Agent Skill content.
`
	case "version":
		return `Usage:
  tanso version [--json]
  tanso --version

Print the CLI version.
`
	default:
		return `Usage:
  tanso <query> [--json|--markdown|--table|--raw] [--source source_id] [--limit 1..50] [--timeout duration] [--config path]
  tanso all <query> [flags]
  tanso bocha <query> [flags]
  tanso volc <query> [flags]
  tanso zhihu <query> [flags]
  tanso zhihu web <query> [--filter string] [--search-db all|realtime|static] [flags]
  tanso zhihu hot [flags]
  tanso sources [--json]
  tanso config <init|path|show>
  tanso skills <list|read>
  tanso version

Global flags:
  --json          machine-readable JSON
  --markdown      Markdown output for research reports
  --table         tabular human output
  --raw           raw JSON envelope for provider debugging
  --limit int     requested per-source limit, 1..50
  --timeout dur   total command timeout, for example 10s
  --config path   config file path
  --source id     source id for generic query; repeat or comma-separate
  --no-color      accepted for script compatibility
  --verbose       accepted for script compatibility
  --help, -h      show help
  --version       show version

Source IDs:
  bocha_web, volcengine_answer, zhihu_search, zhihu_web, zhihu_hot
`
	}
}

func runSkills(p parsed, version string, stdout, stderr io.Writer) int {
	if err := validateSkills(p); err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return ExitInvalidArgument
	}
	reader, err := newSkillReader()
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return ExitInternal
	}

	action := p.Positionals[0]
	switch action {
	case "list":
		skills, err := reader.List()
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err.Error())
			return ExitInternal
		}
		if err := output.WriteJSON(stdout, map[string]any{
			"version": version,
			"skills":  skills,
			"count":   len(skills),
		}); err != nil {
			_, _ = fmt.Fprintln(stderr, err.Error())
			return ExitInternal
		}
		return ExitOK
	case "read":
		name, relpath := skillcontent.SplitTarget(p.Positionals[1])
		if len(p.Positionals) == 3 {
			relpath = p.Positionals[2]
		}
		result, err := reader.Read(name, relpath)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err.Error())
			return ExitInvalidArgument
		}
		if p.JSON {
			if err := output.WriteJSON(stdout, map[string]any{
				"version":  version,
				"skill":    result.Skill,
				"path":     result.Path,
				"content":  result.Content,
				"guidance": result.Guidance,
			}); err != nil {
				_, _ = fmt.Fprintln(stderr, err.Error())
				return ExitInternal
			}
			return ExitOK
		}
		if _, err := io.WriteString(stdout, result.Content); err != nil {
			_, _ = fmt.Fprintln(stderr, err.Error())
			return ExitInternal
		}
		return ExitOK
	default:
		_, _ = fmt.Fprintf(stderr, "unknown skills command: %s\n", action)
		return ExitInvalidArgument
	}
}

func newSkillReader() (*skillcontent.Reader, error) {
	fsys, err := skillcontent.OpenFS()
	if err != nil {
		return nil, fmt.Errorf("skill content not available: %w", err)
	}
	return skillcontent.New(fsys), nil
}

func runConfig(p parsed, stdout, stderr io.Writer) int {
	if err := validateConfig(p); err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return ExitInvalidArgument
	}
	action := p.Positionals[0]
	switch action {
	case "init":
		path, err := config.Init(p.Path, p.Force)
		if err != nil {
			if errors.Is(err, os.ErrExist) {
				_, _ = fmt.Fprintln(stderr, "config already exists; use --force to overwrite")
				return ExitConfig
			}
			_, _ = fmt.Fprintln(stderr, err.Error())
			return ExitConfig
		}
		_, _ = fmt.Fprintf(stdout, "created config: %s\n", path)
		return ExitOK
	case "path":
		path, err := config.ResolvePath(config.Options{})
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err.Error())
			return ExitConfig
		}
		_, _ = fmt.Fprintln(stdout, path)
		return ExitOK
	case "show":
		cfg, err := config.Load(config.Options{Path: p.ConfigPath})
		if err != nil {
			return writeCommandError(stdout, stderr, p, tansoerr.ConfigInvalid, err.Error(), ExitConfig)
		}
		if err := output.WriteJSON(stdout, cfg.Redacted()); err != nil {
			_, _ = fmt.Fprintln(stderr, err.Error())
			return ExitInternal
		}
		return ExitOK
	default:
		_, _ = fmt.Fprintf(stderr, "unknown config command: %s\n", action)
		return ExitInvalidArgument
	}
}

func runRetrieval(p parsed, version string, stdout, stderr io.Writer) int {
	if err := rejectUnknownFlags(p); err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return ExitInvalidArgument
	}
	cfg, err := config.Load(config.Options{Path: p.ConfigPath})
	if err != nil {
		return writeCommandError(stdout, stderr, p, tansoerr.ConfigInvalid, err.Error(), ExitConfig)
	}
	plan, err := retrievalPlan(p, cfg)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return ExitInvalidArgument
	}

	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout(p, cfg))
	defer cancel()

	start := time.Now()
	results, callErr := plan.run(ctx)
	duration := time.Since(start).Milliseconds()
	status := sourceStatus(plan.source, plan.effectiveLimit, duration, results, callErr)
	errorsOut := errorsFor(callErr)
	overall, _, exit := search.Decide([]search.SourceStatus{status})

	env := search.Envelope{
		Version: version,
		Query: search.Query{
			Text:    plan.text,
			Mode:    plan.mode,
			Sources: []search.SourceID{plan.source},
			Limit:   plan.requestedLimit,
		},
		Status:       overall,
		Results:      results,
		SourceStatus: []search.SourceStatus{status},
		Errors:       errorsOut,
	}
	if err := writeEnvelope(stdout, stderr, env, p); err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return ExitInternal
	}
	return exit
}

func runGenericRetrieval(p parsed, version string, stdout, stderr io.Writer) int {
	if err := rejectUnknownFlags(p); err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return ExitInvalidArgument
	}
	if p.Filter != "" || p.SearchDB != "" {
		_, _ = fmt.Fprintln(stderr, "--filter and --search-db are only valid for tanso zhihu web")
		return ExitInvalidArgument
	}
	cfg, err := config.Load(config.Options{Path: p.ConfigPath})
	if err != nil {
		return writeCommandError(stdout, stderr, p, tansoerr.ConfigInvalid, err.Error(), ExitConfig)
	}
	text, sourceIDs, err := genericQueryAndSources(p, cfg)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return ExitInvalidArgument
	}
	runners, err := genericRunners(sourceIDs, cfg, p)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return ExitInvalidArgument
	}

	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout(p, cfg))
	defer cancel()

	results := []search.Result{}
	statuses := []search.SourceStatus{}
	errorsOut := []tansoerr.Error{}
	for _, runner := range runners {
		sourceStart := time.Now()
		sourceResults, callErr := runner.run(ctx)
		results = append(results, sourceResults...)
		status := sourceStatus(runner.source, runner.effectiveLimit, time.Since(sourceStart).Milliseconds(), sourceResults, callErr)
		statuses = append(statuses, status)
		errorsOut = append(errorsOut, errorsFor(callErr)...)
	}
	overall, _, exit := search.Decide(statuses)

	env := search.Envelope{
		Version: version,
		Query: search.Query{
			Text:    text,
			Mode:    queryModeFor(runners),
			Sources: sourceIDs,
			Limit:   requestedLimit(p, cfg),
		},
		Status:       overall,
		Results:      results,
		SourceStatus: statuses,
		Errors:       errorsOut,
	}
	if err := writeEnvelope(stdout, stderr, env, p); err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return ExitInternal
	}
	return exit
}

type retrieval struct {
	text           string
	source         search.SourceID
	mode           search.QueryMode
	requestedLimit int
	effectiveLimit int
	run            func(context.Context) ([]search.Result, error)
}

type sourceRunner struct {
	source         search.SourceID
	mode           search.QueryMode
	effectiveLimit int
	run            func(context.Context) ([]search.Result, error)
}

func retrievalPlan(p parsed, cfg config.Config) (retrieval, error) {
	limit := cfg.Search.Limit
	if p.LimitSet {
		limit = p.Limit
	}

	switch p.Command {
	case "bocha":
		text, err := singleQuery(p.Positionals, "tanso bocha")
		if err != nil {
			return retrieval{}, err
		}
		client := bocha.New(cfg.Bocha.APIKey, cfg.Bocha.Endpoint)
		return retrieval{
			text:           text,
			source:         search.SourceBochaWeb,
			mode:           search.QueryModeSearch,
			requestedLimit: limit,
			effectiveLimit: limit,
			run: func(ctx context.Context) ([]search.Result, error) {
				return client.Search(ctx, search.SearchQuery{Text: text, Limit: limit, Language: cfg.Search.Language})
			},
		}, nil
	case "volc":
		args := p.Positionals
		if len(args) > 0 && args[0] == "answer" {
			args = args[1:]
		}
		text, err := singleQuery(args, "tanso volc")
		if err != nil {
			return retrieval{}, err
		}
		client := volcengine.Client{Endpoint: cfg.Volcengine.Endpoint, APIKey: cfg.Volcengine.APIKey, Model: cfg.Volcengine.Model}
		return retrieval{
			text:           text,
			source:         search.SourceVolcengineAnswer,
			mode:           search.QueryModeAnswer,
			requestedLimit: limit,
			effectiveLimit: limit,
			run: func(ctx context.Context) ([]search.Result, error) {
				return client.Answer(ctx, search.AnswerQuery{Text: text, Limit: limit, Language: cfg.Search.Language})
			},
		}, nil
	case "zhihu":
		args := p.Positionals
		if len(args) > 0 && args[0] == "hot" {
			if len(args) != 1 {
				return retrieval{}, fmt.Errorf("usage: tanso zhihu hot")
			}
			return zhihuHotRetrieval(cfg, limit), nil
		}
		global := len(args) > 0 && args[0] == "web"
		if global {
			args = args[1:]
		}
		text, err := singleQuery(args, "tanso zhihu")
		if err != nil {
			return retrieval{}, err
		}
		client := zhihu.Client{EndpointBase: cfg.Zhihu.EndpointBase, AccessSecret: cfg.Zhihu.AccessSecret}
		source := search.SourceZhihuSearch
		effective := clamp(limit, 1, 10)
		run := func(ctx context.Context) ([]search.Result, error) {
			return client.Search(ctx, search.SearchQuery{Text: text, Limit: limit, Language: cfg.Search.Language})
		}
		if global {
			source = search.SourceZhihuWeb
			effective = clamp(limit, 1, 20)
			run = func(ctx context.Context) ([]search.Result, error) {
				return client.GlobalSearch(ctx, search.SearchQuery{Text: text, Limit: limit, Language: cfg.Search.Language, Filter: p.Filter, SearchDB: p.SearchDB})
			}
		}
		return retrieval{
			text:           text,
			source:         source,
			mode:           search.QueryModeSearch,
			requestedLimit: limit,
			effectiveLimit: effective,
			run:            run,
		}, nil
	case "hot":
		if len(p.Positionals) != 1 || p.Positionals[0] != "zhihu" {
			return retrieval{}, fmt.Errorf("usage: tanso zhihu hot")
		}
		return zhihuHotRetrieval(cfg, limit), nil
	default:
		return retrieval{}, fmt.Errorf("unknown command: %s", p.Command)
	}
}

func zhihuHotRetrieval(cfg config.Config, limit int) retrieval {
	client := zhihu.Client{EndpointBase: cfg.Zhihu.EndpointBase, AccessSecret: cfg.Zhihu.AccessSecret}
	return retrieval{
		source:         search.SourceZhihuHot,
		mode:           search.QueryModeHotlist,
		requestedLimit: limit,
		effectiveLimit: clamp(limit, 1, 30),
		run: func(ctx context.Context) ([]search.Result, error) {
			return client.Hotlist(ctx, search.HotlistQuery{Limit: limit, Language: cfg.Search.Language})
		},
	}
}

func genericQueryAndSources(p parsed, cfg config.Config) (string, []search.SourceID, error) {
	var queryParts []string
	if p.Command == "all" {
		queryParts = p.Positionals
	} else {
		queryParts = append([]string{p.Command}, p.Positionals...)
	}
	text, err := singleQuery(queryParts, "tanso")
	if err != nil {
		return "", nil, err
	}

	if len(p.SourceIDs) > 0 {
		sourceIDs := make([]search.SourceID, 0, len(p.SourceIDs))
		for _, raw := range p.SourceIDs {
			sourceID, err := parseGenericSource(raw)
			if err != nil {
				return "", nil, err
			}
			sourceIDs = append(sourceIDs, sourceID)
		}
		return text, sourceIDs, nil
	}
	if p.Command == "all" {
		return text, allQuerySources(cfg), nil
	}
	sourceIDs := make([]search.SourceID, 0, len(cfg.Search.DefaultSourceIDs))
	for _, raw := range cfg.Search.DefaultSourceIDs {
		sourceID, err := parseGenericSource(raw)
		if err != nil {
			return "", nil, err
		}
		sourceIDs = append(sourceIDs, sourceID)
	}
	if len(sourceIDs) == 0 {
		return "", nil, fmt.Errorf("no default sources configured")
	}
	return text, sourceIDs, nil
}

func allQuerySources(cfg config.Config) []search.SourceID {
	sourceIDs := []search.SourceID{}
	if cfg.Bocha.Enabled {
		sourceIDs = append(sourceIDs, search.SourceBochaWeb)
	}
	if cfg.Volcengine.Enabled {
		sourceIDs = append(sourceIDs, search.SourceVolcengineAnswer)
	}
	if cfg.Zhihu.Enabled {
		sourceIDs = append(sourceIDs, search.SourceZhihuSearch, search.SourceZhihuWeb)
	}
	return sourceIDs
}

func parseGenericSource(raw string) (search.SourceID, error) {
	switch raw {
	case "bocha", string(search.SourceBochaWeb):
		return search.SourceBochaWeb, nil
	case "volc", "volcengine", string(search.SourceVolcengineAnswer):
		return search.SourceVolcengineAnswer, nil
	case "zhihu", string(search.SourceZhihuSearch):
		return search.SourceZhihuSearch, nil
	case string(search.SourceZhihuWeb):
		return search.SourceZhihuWeb, nil
	default:
		return "", fmt.Errorf("unsupported generic source: %s", raw)
	}
}

func genericRunners(sourceIDs []search.SourceID, cfg config.Config, p parsed) ([]sourceRunner, error) {
	runners := make([]sourceRunner, 0, len(sourceIDs))
	for _, sourceID := range sourceIDs {
		runner, err := genericRunner(sourceID, cfg, p)
		if err != nil {
			return nil, err
		}
		runners = append(runners, runner)
	}
	if len(runners) == 0 {
		return nil, fmt.Errorf("no sources selected")
	}
	return runners, nil
}

func genericRunner(sourceID search.SourceID, cfg config.Config, p parsed) (sourceRunner, error) {
	limit := requestedLimit(p, cfg)
	switch sourceID {
	case search.SourceBochaWeb:
		text := genericText(p)
		client := bocha.New(cfg.Bocha.APIKey, cfg.Bocha.Endpoint)
		return sourceRunner{
			source:         search.SourceBochaWeb,
			mode:           search.QueryModeSearch,
			effectiveLimit: limit,
			run: func(ctx context.Context) ([]search.Result, error) {
				return client.Search(ctx, search.SearchQuery{Text: text, Limit: limit, Language: cfg.Search.Language})
			},
		}, nil
	case search.SourceVolcengineAnswer:
		text := genericText(p)
		client := volcengine.Client{Endpoint: cfg.Volcengine.Endpoint, APIKey: cfg.Volcengine.APIKey, Model: cfg.Volcengine.Model}
		return sourceRunner{
			source:         search.SourceVolcengineAnswer,
			mode:           search.QueryModeAnswer,
			effectiveLimit: limit,
			run: func(ctx context.Context) ([]search.Result, error) {
				return client.Answer(ctx, search.AnswerQuery{Text: text, Limit: limit, Language: cfg.Search.Language})
			},
		}, nil
	case search.SourceZhihuSearch:
		text := genericText(p)
		client := zhihu.Client{EndpointBase: cfg.Zhihu.EndpointBase, AccessSecret: cfg.Zhihu.AccessSecret}
		return sourceRunner{
			source:         search.SourceZhihuSearch,
			mode:           search.QueryModeSearch,
			effectiveLimit: clamp(limit, 1, 10),
			run: func(ctx context.Context) ([]search.Result, error) {
				return client.Search(ctx, search.SearchQuery{Text: text, Limit: limit, Language: cfg.Search.Language})
			},
		}, nil
	case search.SourceZhihuWeb:
		text := genericText(p)
		client := zhihu.Client{EndpointBase: cfg.Zhihu.EndpointBase, AccessSecret: cfg.Zhihu.AccessSecret}
		return sourceRunner{
			source:         search.SourceZhihuWeb,
			mode:           search.QueryModeSearch,
			effectiveLimit: clamp(limit, 1, 20),
			run: func(ctx context.Context) ([]search.Result, error) {
				return client.GlobalSearch(ctx, search.SearchQuery{Text: text, Limit: limit, Language: cfg.Search.Language})
			},
		}, nil
	default:
		return sourceRunner{}, fmt.Errorf("source %s is not valid for generic query", sourceID)
	}
}

func genericText(p parsed) string {
	if p.Command == "all" {
		return strings.Join(p.Positionals, " ")
	}
	return strings.Join(append([]string{p.Command}, p.Positionals...), " ")
}

func queryModeFor(runners []sourceRunner) search.QueryMode {
	if len(runners) == 1 {
		return runners[0].mode
	}
	return search.QueryModeMixed
}

func requestedLimit(p parsed, cfg config.Config) int {
	if p.LimitSet {
		return p.Limit
	}
	return cfg.Search.Limit
}

func sourceStatus(source search.SourceID, effectiveLimit int, durationMS int64, results []search.Result, err error) search.SourceStatus {
	status := search.SourceStatusOK
	var ferr *tansoerr.Error
	if err != nil {
		converted := toTansoError(source, err)
		ferr = &converted
		status = statusForError(converted)
	}
	return search.SourceStatus{
		Source:         source,
		Status:         status,
		Results:        len(results),
		EffectiveLimit: effectiveLimit,
		DurationMS:     durationMS,
		Error:          ferr,
	}
}

func errorsFor(err error) []tansoerr.Error {
	if err == nil {
		return []tansoerr.Error{}
	}
	var ferr tansoerr.Error
	if errors.As(err, &ferr) {
		return []tansoerr.Error{ferr}
	}
	return []tansoerr.Error{{Code: tansoerr.InternalError, Message: err.Error(), Retryable: false}}
}

func toTansoError(source search.SourceID, err error) tansoerr.Error {
	var ferr tansoerr.Error
	if errors.As(err, &ferr) {
		return ferr
	}
	return tansoerr.Error{Code: tansoerr.InternalError, Message: err.Error(), Source: string(source)}
}

func statusForError(err tansoerr.Error) search.SourceStatusValue {
	switch err.Code {
	case tansoerr.CredentialMissing:
		return search.SourceStatusSkipped
	case tansoerr.SourceTimeout:
		return search.SourceStatusTimeout
	case tansoerr.SourceUnauthorized:
		return search.SourceStatusUnauthorized
	case tansoerr.SourceRateLimited:
		return search.SourceStatusRateLimited
	default:
		return search.SourceStatusError
	}
}

func writeEnvelope(stdout, stderr io.Writer, env search.Envelope, p parsed) error {
	if p.JSON || p.Raw {
		return output.WriteJSON(stdout, env)
	}
	if env.Status != search.StatusOK {
		if err := writeHumanErrors(stderr, env); err != nil {
			return err
		}
		if env.Status == search.StatusError {
			return nil
		}
	}
	if p.Markdown {
		return output.WriteMarkdown(stdout, env)
	}
	return output.WriteTable(stdout, env)
}

func writeHumanErrors(stderr io.Writer, env search.Envelope) error {
	for _, item := range env.Errors {
		source := ""
		if item.Source != "" {
			source = " [" + item.Source + "]"
		}
		if _, err := fmt.Fprintf(stderr, "error%s: %s: %s\n", source, item.Code, item.Message); err != nil {
			return err
		}
	}
	if len(env.Errors) == 0 {
		_, err := fmt.Fprintln(stderr, "error: command returned no results")
		return err
	}
	return nil
}

func writeCommandError(stdout, stderr io.Writer, p parsed, code, message string, exit int) int {
	if p.JSON {
		_ = output.WriteJSON(stdout, map[string]any{
			"status": "error",
			"errors": []tansoerr.Error{{
				Code:      code,
				Message:   message,
				Retryable: false,
			}},
		})
		return exit
	}
	_, _ = fmt.Fprintln(stderr, message)
	return exit
}

func singleQuery(args []string, usage string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("missing query for %s", usage)
	}
	return strings.Join(args, " "), nil
}

func parseTimeout(value string) time.Duration {
	timeout, err := time.ParseDuration(value)
	if err != nil || timeout <= 0 {
		return 12 * time.Second
	}
	return timeout
}

func commandTimeout(p parsed, cfg config.Config) time.Duration {
	if p.Timeout != "" {
		return parseTimeout(p.Timeout)
	}
	return parseTimeout(cfg.Search.Timeout)
}

func validateTimeout(p parsed) error {
	if p.Timeout == "" {
		return nil
	}
	timeout, err := time.ParseDuration(p.Timeout)
	if err != nil || timeout <= 0 {
		return fmt.Errorf("--timeout must be a positive duration")
	}
	return nil
}

func validateHelp(p parsed) error {
	if err := rejectUnknownFlags(p); err != nil {
		return err
	}
	if p.JSON || p.Markdown || p.Table || p.Raw {
		return fmt.Errorf("output flags are not valid for tanso help")
	}
	return nil
}

func validateVersion(p parsed) error {
	if err := rejectUnknownFlags(p); err != nil {
		return err
	}
	if p.Markdown || p.Table || p.Raw {
		return fmt.Errorf("only --json is valid for tanso version")
	}
	if len(p.Positionals) > 0 {
		return fmt.Errorf("unexpected argument for tanso version: %s", p.Positionals[0])
	}
	return nil
}

func validateSources(p parsed) error {
	if err := rejectUnknownFlags(p); err != nil {
		return err
	}
	if p.Markdown || p.Table || p.Raw {
		return fmt.Errorf("only --json is valid for tanso sources")
	}
	if len(p.Positionals) > 0 {
		return fmt.Errorf("unexpected argument for tanso sources: %s", p.Positionals[0])
	}
	return nil
}

func validateSkills(p parsed) error {
	if err := rejectUnknownFlags(p); err != nil {
		return err
	}
	if len(p.Positionals) == 0 {
		return fmt.Errorf("usage: tanso skills <list|read>")
	}
	switch p.Positionals[0] {
	case "list":
		if p.Markdown || p.Table || p.Raw {
			return fmt.Errorf("only --json is valid for tanso skills list")
		}
		if len(p.Positionals) != 1 {
			return fmt.Errorf("tanso skills list takes no arguments")
		}
	case "read":
		if p.Markdown || p.Table || p.Raw {
			return fmt.Errorf("only --json is valid for tanso skills read")
		}
		if len(p.Positionals) < 2 || len(p.Positionals) > 3 {
			return fmt.Errorf("usage: tanso skills read <name>[/<path>] [path]")
		}
	default:
		return nil
	}
	return nil
}

func validateConfig(p parsed) error {
	if err := rejectUnknownFlags(p); err != nil {
		return err
	}
	if len(p.Positionals) == 0 {
		return fmt.Errorf("usage: tanso config <init|path|show>")
	}
	if len(p.Positionals) > 1 {
		return fmt.Errorf("unexpected argument for tanso config %s: %s", p.Positionals[0], p.Positionals[1])
	}
	switch p.Positionals[0] {
	case "init":
		if p.JSON || p.Markdown || p.Table || p.Raw {
			return fmt.Errorf("output flags are not valid for tanso config init")
		}
		if p.ConfigPath != "" {
			return fmt.Errorf("--config is not valid for tanso config init; use --path")
		}
	case "path":
		if p.JSON || p.Markdown || p.Table || p.Raw {
			return fmt.Errorf("output flags are not valid for tanso config path")
		}
		if p.Path != "" || p.Force || p.ConfigPath != "" {
			return fmt.Errorf("flags are not valid for tanso config path")
		}
	case "show":
		if !p.JSON {
			return fmt.Errorf("only --json is valid for tanso config show")
		}
		if p.Markdown || p.Table || p.Raw {
			return fmt.Errorf("only --json is valid for tanso config show")
		}
		if p.Path != "" || p.Force {
			return fmt.Errorf("--path and --force are not valid for tanso config show")
		}
	default:
		return nil
	}
	return nil
}

func rejectUnknownFlags(p parsed) error {
	if len(p.UnknownFlags) > 0 {
		return fmt.Errorf("unknown flag: %s", p.UnknownFlags[0])
	}
	return nil
}

func validateOutputModes(p parsed) error {
	count := 0
	for _, enabled := range []bool{p.JSON, p.Markdown, p.Table, p.Raw} {
		if enabled {
			count++
		}
	}
	if count > 1 {
		return fmt.Errorf("output flags are mutually exclusive")
	}
	return nil
}

func validateSourceFlag(p parsed) error {
	if len(p.SourceIDs) == 0 {
		return nil
	}
	if isGenericCommand(p) {
		return nil
	}
	return fmt.Errorf("--source is only valid for generic tanso <query> or tanso all <query>")
}

func isRetrievalCommand(p parsed) bool {
	switch p.Command {
	case "bocha", "volc", "zhihu", "hot":
		return true
	default:
		return false
	}
}

func isConfigInitCommand(p parsed) bool {
	if p.Command == "init" {
		return true
	}
	return p.Command == "config" && len(p.Positionals) > 0 && p.Positionals[0] == "init"
}

func isZhihuWebCommand(p parsed) bool {
	return p.Command == "zhihu" && len(p.Positionals) > 0 && p.Positionals[0] == "web"
}

func isGenericCommand(p parsed) bool {
	if p.Command == "" {
		return false
	}
	if p.Command == "all" {
		return true
	}
	switch p.Command {
	case "help", "version", "sources", "skills", "config", "init", "bocha", "volc", "zhihu", "hot":
		return false
	default:
		return true
	}
}

func writeSourcesText(stdout io.Writer) {
	_, _ = fmt.Fprintln(stdout, "bocha_web")
	_, _ = fmt.Fprintln(stdout, "volcengine_answer")
	_, _ = fmt.Fprintln(stdout, "zhihu_search")
	_, _ = fmt.Fprintln(stdout, "zhihu_web")
	_, _ = fmt.Fprintln(stdout, "zhihu_hot")
}

func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
