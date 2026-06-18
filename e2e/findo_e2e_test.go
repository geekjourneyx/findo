package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindoBinaryVersionAndSources(t *testing.T) {
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "findo")

	build := exec.Command("go", "build", "-buildvcs=false", "-trimpath", "-o", bin, "./cmd/findo")
	build.Dir = ".."
	build.Env = append(os.Environ(), "GOCACHE=/tmp/findo-go-cache")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}

	version := exec.Command(bin, "version")
	out, err := version.CombinedOutput()
	if err != nil {
		t.Fatalf("version failed: %v\n%s", err, out)
	}
	wantVersion := "findo " + makefileVersion(t)
	if strings.TrimSpace(string(out)) != wantVersion {
		t.Fatalf("version output = %q", out)
	}

	sources := exec.Command(bin, "sources", "--json")
	sources.Env = append(os.Environ(), "BOCHA_API_KEY=", "VOLCENGINE_API_KEY=", "ARK_API_KEY=", "ZHIHU_ACCESS_SECRET=", "ZHIHU_API_KEY=")
	out, err = sources.CombinedOutput()
	if err != nil {
		t.Fatalf("sources failed: %v\n%s", err, out)
	}
	for _, source := range []string{"bocha_web", "volcengine_answer", "zhihu_search", "zhihu_web", "zhihu_hot"} {
		if !strings.Contains(string(out), `"source":"`+source+`"`) {
			t.Fatalf("sources output missing %s: %s", source, out)
		}
	}
}

func makefileVersion(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../Makefile")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "VERSION ?= ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "VERSION ?= "))
		}
	}
	t.Fatal("Makefile VERSION is missing")
	return ""
}
