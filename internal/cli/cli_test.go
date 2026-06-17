package cli

import (
	"bytes"
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
	if got, want := stdout.String(), "findo 1.0.0\n"; got != want {
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
	if !strings.Contains(stderr.String(), "--filter is only valid for findo zhihu web") {
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
	if !strings.Contains(stderr.String(), "--search-db is only valid for findo zhihu web") {
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
			want: "--filter is only valid for findo zhihu web",
		},
		{
			name: "help search db",
			args: []string{"help", "--search-db", "x"},
			want: "--search-db is only valid for findo zhihu web",
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
