package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionText(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"version"}, "1.0.0", &stdout, &stderr)

	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d", code, ExitOK)
	}
	if got, want := stdout.String(), "tanso 1.0.0\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestVersionJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"version", "--json"}, "1.0.0", &stdout, &stderr)

	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d", code, ExitOK)
	}
	if got, want := stdout.String(), "{\"version\":\"1.0.0\"}\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestVersionLongFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"--version"}, "1.0.0", &stdout, &stderr)

	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d", code, ExitOK)
	}
	if got, want := stdout.String(), "tanso 1.0.0\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestHelpFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "root long", args: []string{"--help"}, want: "Usage:"},
		{name: "root short", args: []string{"-h"}, want: "Global flags:"},
		{name: "help topic", args: []string{"help", "bocha"}, want: "tanso bocha <query>"},
		{name: "subcommand", args: []string{"bocha", "--help"}, want: "tanso bocha <query>"},
		{name: "nested subcommand", args: []string{"config", "init", "--help"}, want: "tanso config init"},
		{name: "zhihu web", args: []string{"zhihu", "web", "--help"}, want: "--search-db"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := Run(tt.args, "1.0.0", &stdout, &stderr)

			if code != ExitOK {
				t.Fatalf("exit code = %d, want %d; stderr=%q", code, ExitOK, stderr.String())
			}
			if !strings.Contains(stdout.String(), tt.want) {
				t.Fatalf("stdout missing %q:\n%s", tt.want, stdout.String())
			}
			if got := stderr.String(); got != "" {
				t.Fatalf("stderr = %q, want empty", got)
			}
		})
	}
}

func TestInvalidSourceSpecificFlagOnWrongCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"bocha", "query", "--filter", `host=="example.com"`}, "1.0.0", &stdout, &stderr)

	if code != ExitInvalidArgument {
		t.Fatalf("exit code = %d, want %d", code, ExitInvalidArgument)
	}
	if !strings.Contains(stderr.String(), "--filter is only valid for tanso zhihu web") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestInvalidSearchDBOnWrongCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"bocha", "query", "--search-db", "static"}, "1.0.0", &stdout, &stderr)

	if code != ExitInvalidArgument {
		t.Fatalf("exit code = %d, want %d", code, ExitInvalidArgument)
	}
	if !strings.Contains(stderr.String(), "--search-db is only valid for tanso zhihu web") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestZhihuWebSourceSpecificFlagsAllowLeadingGlobalFlags(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ZHIHU_ACCESS_SECRET", "")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"--json", "zhihu", "web", "ChatGPT", "--filter", `host=="example.com"`}, "1.0.0", &stdout, &stderr)

	if code != ExitCredential {
		t.Fatalf("exit code = %d, want %d; stdout=%q stderr=%q", code, ExitCredential, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"source":"zhihu_web"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if strings.Contains(stderr.String(), "--filter is only valid") {
		t.Fatalf("stderr should not reject valid zhihu web flags: %q", stderr.String())
	}
}

func TestSourceSpecificFlagsInvalidOnInspectionCommands(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "version filter",
			args: []string{"version", "--filter", "x"},
			want: "--filter is only valid for tanso zhihu web",
		},
		{
			name: "help search db",
			args: []string{"help", "--search-db", "x"},
			want: "--search-db is only valid for tanso zhihu web",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := Run(tt.args, "1.0.0", &stdout, &stderr)

			if code != ExitInvalidArgument {
				t.Fatalf("exit code = %d, want %d", code, ExitInvalidArgument)
			}
			if !strings.Contains(stderr.String(), tt.want) {
				t.Fatalf("stderr = %q", stderr.String())
			}
		})
	}
}

func TestInvalidFlagsAndPositionalsOnImplementedCommands(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "sources bogus flag", args: []string{"sources", "--bogus"}},
		{name: "sources extra positional", args: []string{"sources", "extra"}},
		{name: "version bogus flag", args: []string{"version", "--bogus"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := Run(tt.args, "1.0.0", &stdout, &stderr)

			if code != ExitInvalidArgument {
				t.Fatalf("exit code = %d, want %d", code, ExitInvalidArgument)
			}
			if stderr.Len() == 0 {
				t.Fatalf("stderr empty, want diagnostic")
			}
		})
	}
}

func TestDefaultQueryRunsConfiguredDefaultSources(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("TANSO_CONFIG", "")
	t.Setenv("BOCHA_API_KEY", "")
	t.Setenv("ARK_API_KEY", "")
	t.Setenv("VOLCENGINE_API_KEY", "")
	t.Setenv("ZHIHU_ACCESS_SECRET", "")
	t.Setenv("ZHIHU_API_KEY", "")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"AI Agent 商业化", "--json"}, "1.0.0", &stdout, &stderr)

	if code != ExitCredential {
		t.Fatalf("exit code = %d, want %d; stdout=%q stderr=%q", code, ExitCredential, stdout.String(), stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{`"text":"AI Agent 商业化"`, `"mode":"mixed"`, `"source":"bocha_web"`, `"source":"volcengine_answer"`, `"source":"zhihu_search"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %s: %s", want, out)
		}
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestAllQueryRunsAllSearchAndAnswerSources(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("TANSO_CONFIG", "")
	t.Setenv("BOCHA_API_KEY", "")
	t.Setenv("ARK_API_KEY", "")
	t.Setenv("VOLCENGINE_API_KEY", "")
	t.Setenv("ZHIHU_ACCESS_SECRET", "")
	t.Setenv("ZHIHU_API_KEY", "")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"all", "AI Agent 商业化", "--json"}, "1.0.0", &stdout, &stderr)

	if code != ExitCredential {
		t.Fatalf("exit code = %d, want %d; stdout=%q stderr=%q", code, ExitCredential, stdout.String(), stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{`"source":"bocha_web"`, `"source":"volcengine_answer"`, `"source":"zhihu_search"`, `"source":"zhihu_web"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %s: %s", want, out)
		}
	}
	if strings.Contains(out, `"source":"zhihu_hot"`) {
		t.Fatalf("generic all should not include hotlist: %s", out)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestGenericSourceFlagLimitsSources(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("TANSO_CONFIG", "")
	t.Setenv("BOCHA_API_KEY", "")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"AI Agent 商业化", "--source", "bocha_web", "--json"}, "1.0.0", &stdout, &stderr)

	if code != ExitCredential {
		t.Fatalf("exit code = %d, want %d; stdout=%q stderr=%q", code, ExitCredential, stdout.String(), stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, `"sources":["bocha_web"]`) || strings.Contains(out, `"source":"zhihu_search"`) {
		t.Fatalf("unexpected sources: %s", out)
	}
}

func TestSourceFlagRejectedOnExplicitSourceCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"bocha", "query", "--source", "bocha_web"}, "1.0.0", &stdout, &stderr)

	if code != ExitInvalidArgument {
		t.Fatalf("exit code = %d, want %d", code, ExitInvalidArgument)
	}
	if !strings.Contains(stderr.String(), "--source is only valid") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestHumanFailureWritesDiagnosticsToStderr(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("BOCHA_API_KEY", "")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"bocha", "AI Agent 商业化"}, "1.0.0", &stdout, &stderr)

	if code != ExitCredential {
		t.Fatalf("exit code = %d, want %d", code, ExitCredential)
	}
	if got := stdout.String(); got != "" {
		t.Fatalf("stdout = %q, want empty", got)
	}
	if !strings.Contains(stderr.String(), "CREDENTIAL_MISSING") {
		t.Fatalf("stderr missing diagnostic: %q", stderr.String())
	}
}

func TestSourcesJSON(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("TANSO_CONFIG", "")
	t.Setenv("BOCHA_API_KEY", "")
	t.Setenv("ARK_API_KEY", "")
	t.Setenv("VOLCENGINE_API_KEY", "")
	t.Setenv("ZHIHU_ACCESS_SECRET", "")
	t.Setenv("ZHIHU_API_KEY", "")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"sources", "--json"}, "1.0.0", &stdout, &stderr)

	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d; stderr=%q", code, ExitOK, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"version":"1.0.0"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
	for _, source := range []string{"bocha_web", "volcengine_answer", "zhihu_search", "zhihu_web", "zhihu_hot"} {
		if !strings.Contains(stdout.String(), `"source":"`+source+`"`) {
			t.Fatalf("stdout missing source %q: %q", source, stdout.String())
		}
	}
	got := decodeSourcesJSON(t, stdout.Bytes())
	for _, source := range got.Sources {
		if source.Configured {
			t.Fatalf("source %s configured = true, want false", source.Source)
		}
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestSourcesJSONMarksConfiguredFromEnv(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("TANSO_CONFIG", "")
	t.Setenv("BOCHA_API_KEY", "bocha-env-secret")
	t.Setenv("ARK_API_KEY", "ark-env-secret")
	t.Setenv("VOLCENGINE_API_KEY", "")
	t.Setenv("ZHIHU_ACCESS_SECRET", "zhihu-env-secret")
	t.Setenv("ZHIHU_API_KEY", "")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"sources", "--json"}, "1.0.0", &stdout, &stderr)

	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d; stderr=%q", code, ExitOK, stderr.String())
	}
	got := decodeSourcesJSON(t, stdout.Bytes())
	for _, source := range got.Sources {
		if !source.Configured {
			t.Fatalf("source %s configured = false, want true", source.Source)
		}
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestSourcesJSONMarksConfiguredFromConfigPath(t *testing.T) {
	t.Setenv("BOCHA_API_KEY", "")
	t.Setenv("ARK_API_KEY", "")
	t.Setenv("VOLCENGINE_API_KEY", "")
	t.Setenv("ZHIHU_ACCESS_SECRET", "")
	t.Setenv("ZHIHU_API_KEY", "")
	path := filepath.Join(t.TempDir(), "tanso.yaml")
	err := os.WriteFile(path, []byte(`
bocha:
  api_key: bocha-file-secret
`), 0600)
	if err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"sources", "--json", "--config", path}, "1.0.0", &stdout, &stderr)

	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d; stderr=%q", code, ExitOK, stderr.String())
	}
	got := decodeSourcesJSON(t, stdout.Bytes())
	if !configuredFor(got, "bocha_web") {
		t.Fatalf("bocha_web configured = false, want true: %#v", got.Sources)
	}
	for _, source := range []string{"volcengine_answer", "zhihu_search", "zhihu_web", "zhihu_hot"} {
		if configuredFor(got, source) {
			t.Fatalf("%s configured = true, want false: %#v", source, got.Sources)
		}
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

type sourcesResponse struct {
	Sources []struct {
		Source     string `json:"source"`
		Configured bool   `json:"configured"`
	} `json:"sources"`
}

func decodeSourcesJSON(t *testing.T, b []byte) sourcesResponse {
	t.Helper()
	var got sourcesResponse
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal stdout: %v\n%s", err, string(b))
	}
	return got
}

func configuredFor(got sourcesResponse, source string) bool {
	for _, item := range got.Sources {
		if item.Source == source {
			return item.Configured
		}
	}
	return false
}

func TestSkillsListJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"skills", "list", "--json"}, "1.2.0", &stdout, &stderr)

	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d; stderr=%q", code, ExitOK, stderr.String())
	}
	var got struct {
		Version string `json:"version"`
		Count   int    `json:"count"`
		Skills  []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"skills"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal stdout: %v\n%s", err, stdout.String())
	}
	if got.Version != "1.2.0" || got.Count != 1 {
		t.Fatalf("unexpected response: %#v", got)
	}
	if got.Skills[0].Name != "tanso" || !strings.Contains(got.Skills[0].Description, "exploring Chinese internet signals") {
		t.Fatalf("unexpected skill: %#v", got.Skills[0])
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestSkillsReadRaw(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"skills", "read", "tanso"}, "1.2.0", &stdout, &stderr)

	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d; stderr=%q", code, ExitOK, stderr.String())
	}
	if !strings.HasPrefix(stdout.String(), "---\nname: tanso") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if strings.Contains(stdout.String(), `"content"`) {
		t.Fatalf("raw output must not be JSON wrapped: %s", stdout.String())
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestSkillsReadJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"skills", "read", "tanso", "--json"}, "1.2.0", &stdout, &stderr)

	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d; stderr=%q", code, ExitOK, stderr.String())
	}
	var got struct {
		Version  string `json:"version"`
		Skill    string `json:"skill"`
		Path     string `json:"path"`
		Content  string `json:"content"`
		Guidance string `json:"guidance"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal stdout: %v\n%s", err, stdout.String())
	}
	if got.Version != "1.2.0" || got.Skill != "tanso" || got.Path != "SKILL.md" {
		t.Fatalf("unexpected response: %#v", got)
	}
	if !strings.Contains(got.Content, "AI Search CLI") {
		t.Fatalf("content missing bundled skill: %.120q", got.Content)
	}
	if !strings.Contains(got.Guidance, "tanso skills read tanso --json") {
		t.Fatalf("guidance = %q", got.Guidance)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestSkillsReadRejectsTraversal(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"skills", "read", "tanso", "../../etc/passwd"}, "1.2.0", &stdout, &stderr)

	if code != ExitInvalidArgument {
		t.Fatalf("exit code = %d, want %d", code, ExitInvalidArgument)
	}
	if !strings.Contains(stderr.String(), "invalid path") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if got := stdout.String(); got != "" {
		t.Fatalf("stdout = %q, want empty", got)
	}
}

func TestConfigPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"config", "path"}, "1.0.0", &stdout, &stderr)

	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d; stderr=%q", code, ExitOK, stderr.String())
	}
	want := filepath.Join(dir, "tanso", "config.yaml") + "\n"
	if got := stdout.String(); got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestConfigPathUsesTansoConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "custom.yaml")
	t.Setenv("TANSO_CONFIG", path)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"config", "path"}, "1.0.0", &stdout, &stderr)

	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d; stderr=%q", code, ExitOK, stderr.String())
	}
	if got, want := stdout.String(), path+"\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestConfigInitCreatesDefaultConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tanso.yaml")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"config", "init", "--path", path}, "1.0.0", &stdout, &stderr)

	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d; stderr=%q", code, ExitOK, stderr.String())
	}
	if !strings.Contains(stdout.String(), "created config: "+path) {
		t.Fatalf("stdout = %q", stdout.String())
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("mode = %v, want 0600", got)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `api_key: ""`) {
		t.Fatalf("config should contain empty API key fields:\n%s", string(b))
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestConfigInitDoesNotOverwriteWithoutForce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tanso.yaml")
	if err := os.WriteFile(path, []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"config", "init", "--path", path}, "1.0.0", &stdout, &stderr)

	if code != ExitConfig {
		t.Fatalf("exit code = %d, want %d", code, ExitConfig)
	}
	if !strings.Contains(stderr.String(), "config already exists") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if got := stdout.String(); got != "" {
		t.Fatalf("stdout = %q, want empty", got)
	}
}

func TestConfigInitForceOverwrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tanso.yaml")
	if err := os.WriteFile(path, []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"config", "init", "--path", path, "--force"}, "1.0.0", &stdout, &stderr)

	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d; stderr=%q", code, ExitOK, stderr.String())
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) == "existing" {
		t.Fatalf("config was not overwritten")
	}
}

func TestInitAliasAcceptsPathAndForce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tanso.yaml")
	if err := os.WriteFile(path, []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"init", "--path", path, "--force"}, "1.0.0", &stdout, &stderr)

	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d; stderr=%q", code, ExitOK, stderr.String())
	}
	if !strings.Contains(stdout.String(), "created config: "+path) {
		t.Fatalf("stdout = %q", stdout.String())
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) == "existing" {
		t.Fatalf("config was not overwritten")
	}
}

func TestConfigShowJSONRedactsSecrets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tanso.yaml")
	err := os.WriteFile(path, []byte(`
bocha:
  api_key: bocha-secret
volcengine:
  api_key: ark-secret
zhihu:
  access_secret: zhihu-secret
`), 0600)
	if err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"config", "show", "--config", path, "--json"}, "1.0.0", &stdout, &stderr)

	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d; stderr=%q", code, ExitOK, stderr.String())
	}
	out := stdout.String()
	for _, secret := range []string{"bocha-secret", "ark-secret", "zhihu-secret"} {
		if strings.Contains(out, secret) {
			t.Fatalf("stdout leaked secret %q: %s", secret, out)
		}
	}
	if got := strings.Count(out, `"***"`); got != 3 {
		t.Fatalf("redaction count = %d, want 3 in %s", got, out)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestConfigShowJSONRedactsEnvSecrets(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("BOCHA_API_KEY", "bocha-env-secret")
	t.Setenv("ARK_API_KEY", "ark-env-secret")
	t.Setenv("ZHIHU_ACCESS_SECRET", "zhihu-env-secret")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"config", "show", "--json"}, "1.0.0", &stdout, &stderr)

	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d; stderr=%q", code, ExitOK, stderr.String())
	}
	out := stdout.String()
	for _, secret := range []string{"bocha-env-secret", "ark-env-secret", "zhihu-env-secret"} {
		if strings.Contains(out, secret) {
			t.Fatalf("stdout leaked secret %q: %s", secret, out)
		}
	}
	if got := strings.Count(out, `"***"`); got != 3 {
		t.Fatalf("redaction count = %d, want 3 in %s", got, out)
	}
}

func TestConfigShowRequiresJSON(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"config", "show"}, "1.0.0", &stdout, &stderr)

	if code != ExitInvalidArgument {
		t.Fatalf("exit code = %d, want %d", code, ExitInvalidArgument)
	}
	if !strings.Contains(stderr.String(), "only --json is valid") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRetrievalReadsDefaultConfigPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path := filepath.Join(dir, "tanso", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("search:\n  limit: 99\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"bocha", "query"}, "1.0.0", &stdout, &stderr)

	if code != ExitConfig {
		t.Fatalf("exit code = %d, want %d", code, ExitConfig)
	}
	if !strings.Contains(stderr.String(), "search.limit must be 1..50") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRetrievalConfigErrorUsesJSONWhenRequested(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"bocha", "query", "--json", "--config", filepath.Join(t.TempDir(), "missing.yaml")}, "1.0.0", &stdout, &stderr)

	if code != ExitConfig {
		t.Fatalf("exit code = %d, want %d; stdout=%q stderr=%q", code, ExitConfig, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"status":"error"`) || !strings.Contains(stdout.String(), `"code":"CONFIG_INVALID"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestPathAndForceRejectedOutsideConfigInit(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"bocha", "query", "--path", "tanso.yaml"}, "1.0.0", &stdout, &stderr)

	if code != ExitInvalidArgument {
		t.Fatalf("exit code = %d, want %d", code, ExitInvalidArgument)
	}
	if !strings.Contains(stderr.String(), "only valid for tanso config init") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestNoStdinQuerySupport(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{}, "1.0.0", &stdout, &stderr)

	if code != ExitOK {
		t.Fatalf("help without stdin should exit 0, got %d", code)
	}
	if strings.Contains(stdout.String(), "stdin") {
		t.Fatalf("help should not advertise stdin query support")
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestExplicitSourceMissingCredentialExitsCredential(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("BOCHA_API_KEY", "")

	code := Run([]string{"bocha", "query", "--json"}, "1.0.0", &stdout, &stderr)

	if code != ExitCredential {
		t.Fatalf("exit = %d, want %d; stdout=%q stderr=%q", code, ExitCredential, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"code":"CREDENTIAL_MISSING"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"source":"bocha_web"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestZhihuHotAlias(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ZHIHU_ACCESS_SECRET", "")

	code := Run([]string{"zhihu", "hot", "--json"}, "1.0.0", &stdout, &stderr)

	if code != ExitCredential {
		t.Fatalf("exit = %d, want %d; stdout=%q stderr=%q", code, ExitCredential, stdout.String(), stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{`"mode":"hotlist"`, `"source":"zhihu_hot"`, `"code":"CREDENTIAL_MISSING"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %s: %s", want, out)
		}
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}
