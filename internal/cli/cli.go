package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/geekjourneyx/findo/internal/config"
	"github.com/geekjourneyx/findo/internal/findoerr"
	"github.com/geekjourneyx/findo/internal/output"
	"github.com/geekjourneyx/findo/internal/search"
	sourcepkg "github.com/geekjourneyx/findo/internal/source"
	"github.com/geekjourneyx/findo/internal/source/bocha"
	"github.com/geekjourneyx/findo/internal/source/volcengine"
	"github.com/geekjourneyx/findo/internal/source/zhihu"
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
	Limit        int
	LimitSet     bool
	ConfigPath   string
	UnknownFlags []string
}

func Run(args []string, version string, stdout, stderr io.Writer) int {
	p, err := parse(args)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return ExitInvalidArgument
	}
	if p.Filter != "" && !isZhihuWebCommand(args) {
		_, _ = fmt.Fprintln(stderr, "--filter is only valid for findo zhihu web")
		return ExitInvalidArgument
	}
	if p.SearchDB != "" && !isZhihuWebCommand(args) {
		_, _ = fmt.Fprintln(stderr, "--search-db is only valid for findo zhihu web")
		return ExitInvalidArgument
	}
	if err := validateOutputModes(p); err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return ExitInvalidArgument
	}
	if p.LimitSet && (p.Limit <= 0 || p.Limit > 50) {
		_, _ = fmt.Fprintln(stderr, "--limit must be 1..50")
		return ExitInvalidArgument
	}
	if len(args) == 0 || p.Command == "help" {
		if err := validateHelp(p); err != nil {
			_, _ = fmt.Fprintln(stderr, err.Error())
			return ExitInvalidArgument
		}
		_, _ = fmt.Fprintln(stdout, "findo <query>")
		return ExitOK
	}
	if p.Command == "version" {
		if err := validateVersion(p); err != nil {
			_, _ = fmt.Fprintln(stderr, err.Error())
			return ExitInvalidArgument
		}
		if p.JSON {
			_, _ = fmt.Fprintf(stdout, `{"version":%q}`+"\n", version)
			return ExitOK
		}
		_, _ = fmt.Fprintf(stdout, "findo %s\n", version)
		return ExitOK
	}
	if p.Command == "sources" {
		if err := validateSources(p); err != nil {
			_, _ = fmt.Fprintln(stderr, err.Error())
			return ExitInvalidArgument
		}
		if p.JSON {
			_ = output.WriteJSON(stdout, map[string]any{"version": version, "sources": sourcepkg.StaticInfos()})
			return ExitOK
		}
		writeSourcesText(stdout)
		return ExitOK
	}
	if isRetrievalCommand(p) {
		return runRetrieval(p, version, stdout, stderr)
	}

	_, _ = fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
	return ExitInvalidArgument
}

func parse(args []string) (parsed, error) {
	p := parsed{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
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
		case "--config":
			if i+1 >= len(args) {
				return p, fmt.Errorf("--config requires a value")
			}
			p.ConfigPath = args[i+1]
			i++
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

func runRetrieval(p parsed, version string, stdout, stderr io.Writer) int {
	if err := rejectUnknownFlags(p); err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return ExitInvalidArgument
	}
	cfg, err := config.Load(config.Options{Path: p.ConfigPath})
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return ExitConfig
	}
	plan, err := retrievalPlan(p, cfg)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return ExitInvalidArgument
	}

	ctx, cancel := context.WithTimeout(context.Background(), parseTimeout(cfg.Search.Timeout))
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
	if err := writeEnvelope(stdout, env, p); err != nil {
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

func retrievalPlan(p parsed, cfg config.Config) (retrieval, error) {
	limit := cfg.Search.Limit
	if p.LimitSet {
		limit = p.Limit
	}

	switch p.Command {
	case "bocha":
		text, err := singleQuery(p.Positionals, "findo bocha")
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
		text, err := singleQuery(args, "findo volc")
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
		global := len(args) > 0 && args[0] == "web"
		if global {
			args = args[1:]
		}
		text, err := singleQuery(args, "findo zhihu")
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
			return retrieval{}, fmt.Errorf("usage: findo hot zhihu")
		}
		client := zhihu.Client{EndpointBase: cfg.Zhihu.EndpointBase, AccessSecret: cfg.Zhihu.AccessSecret}
		return retrieval{
			source:         search.SourceZhihuHot,
			mode:           search.QueryModeHotlist,
			requestedLimit: limit,
			effectiveLimit: clamp(limit, 1, 30),
			run: func(ctx context.Context) ([]search.Result, error) {
				return client.Hotlist(ctx, search.HotlistQuery{Limit: limit, Language: cfg.Search.Language})
			},
		}, nil
	default:
		return retrieval{}, fmt.Errorf("unknown command: %s", p.Command)
	}
}

func sourceStatus(source search.SourceID, effectiveLimit int, durationMS int64, results []search.Result, err error) search.SourceStatus {
	status := search.SourceStatusOK
	var ferr *findoerr.Error
	if err != nil {
		converted := toFindoError(source, err)
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

func errorsFor(err error) []findoerr.Error {
	if err == nil {
		return []findoerr.Error{}
	}
	var ferr findoerr.Error
	if errors.As(err, &ferr) {
		return []findoerr.Error{ferr}
	}
	return []findoerr.Error{{Code: findoerr.InternalError, Message: err.Error(), Retryable: false}}
}

func toFindoError(source search.SourceID, err error) findoerr.Error {
	var ferr findoerr.Error
	if errors.As(err, &ferr) {
		return ferr
	}
	return findoerr.Error{Code: findoerr.InternalError, Message: err.Error(), Source: string(source)}
}

func statusForError(err findoerr.Error) search.SourceStatusValue {
	switch err.Code {
	case findoerr.CredentialMissing:
		return search.SourceStatusSkipped
	case findoerr.SourceTimeout:
		return search.SourceStatusTimeout
	case findoerr.SourceUnauthorized:
		return search.SourceStatusUnauthorized
	case findoerr.SourceRateLimited:
		return search.SourceStatusRateLimited
	default:
		return search.SourceStatusError
	}
}

func writeEnvelope(stdout io.Writer, env search.Envelope, p parsed) error {
	if p.JSON || p.Raw {
		return output.WriteJSON(stdout, env)
	}
	if p.Markdown {
		return output.WriteMarkdown(stdout, env)
	}
	return output.WriteTable(stdout, env)
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

func validateHelp(p parsed) error {
	if err := rejectUnknownFlags(p); err != nil {
		return err
	}
	if p.JSON || p.Markdown || p.Table || p.Raw {
		return fmt.Errorf("output flags are not valid for findo help")
	}
	if len(p.Positionals) > 0 {
		return fmt.Errorf("unexpected argument for findo help: %s", p.Positionals[0])
	}
	return nil
}

func validateVersion(p parsed) error {
	if err := rejectUnknownFlags(p); err != nil {
		return err
	}
	if p.Markdown || p.Table || p.Raw {
		return fmt.Errorf("only --json is valid for findo version")
	}
	if len(p.Positionals) > 0 {
		return fmt.Errorf("unexpected argument for findo version: %s", p.Positionals[0])
	}
	return nil
}

func validateSources(p parsed) error {
	if err := rejectUnknownFlags(p); err != nil {
		return err
	}
	if p.Markdown || p.Table || p.Raw {
		return fmt.Errorf("only --json is valid for findo sources")
	}
	if len(p.Positionals) > 0 {
		return fmt.Errorf("unexpected argument for findo sources: %s", p.Positionals[0])
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

func isRetrievalCommand(p parsed) bool {
	switch p.Command {
	case "bocha", "volc", "zhihu", "hot":
		return true
	default:
		return false
	}
}

func isZhihuWebCommand(args []string) bool {
	return len(args) >= 2 && args[0] == "zhihu" && args[1] == "web"
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
