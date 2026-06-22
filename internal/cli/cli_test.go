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

func TestSourcesJSON(t *testing.T) {
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
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
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
