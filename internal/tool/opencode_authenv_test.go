package tool

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func authEnvTool(t *testing.T) ToolWithContainerEnv {
	t.Helper()
	oc := NewOpencode()
	twce, ok := oc.(ToolWithContainerEnv)
	if !ok {
		t.Fatal("OpencodeTool should implement ToolWithContainerEnv")
	}
	return twce
}

func writeAuth(t *testing.T, dataDir, contents string) string {
	p := filepath.Join(dataDir, "opencode")
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(p, "auth.json"), []byte(contents), 0o600); err != nil {
		t.Fatalf("writeFile: %v", err)
	}
	return dataDir
}

// TestOpencodeAuthContent_Set verifies a readable, valid auth.json becomes a
// compact single-line OPENCODE_AUTH_CONTENT.
func TestOpencodeAuthContent_Set(t *testing.T) {
	dir := t.TempDir()
	auth := `{
  "openai": { "type": "api", "key": "sk-x" },
  "github-copilot": { "type": "oauth", "refresh": "r", "access": "a", "expires": 123 }
}`
	t.Setenv("XDG_DATA_HOME", writeAuth(t, dir, auth))

	oc := NewOpencode().(ToolWithContainerEnv)
	env := oc.GetContainerEnv("/workspace")

	got, ok := env["OPENCODE_AUTH_CONTENT"]
	if !ok {
		t.Fatal("OPENCODE_AUTH_CONTENT missing")
	}
	if got == "" || got == auth {
		t.Errorf("OPENCODE_AUTH_CONTENT not compacted: %q", got)
	}
	for _, s := range []string{"\n", "\r", "\x00"} {
		if strings.Contains(got, s) {
			t.Errorf("OPENCODE_AUTH_CONTENT contains %q: %q", s, got)
		}
	}
	want := `{"openai":{"type":"api","key":"sk-x"},"github-copilot":{"type":"oauth","refresh":"r","access":"a","expires":123}}`
	if got != want {
		t.Errorf("OPENCODE_AUTH_CONTENT = %q, want %q", got, want)
	}
}

// TestOpencodeAuthContent_DefaultXDGPath falls back to ~/.local/share when
// XDG_DATA_HOME is unset.
func TestOpencodeAuthContent_DefaultXDGPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")
	writeAuth(t, filepath.Join(home, ".local", "share"), `{"openai":{"type":"api","key":"sk-x"}}`)

	env := authEnvTool(t).GetContainerEnv("/workspace")
	if _, ok := env["OPENCODE_AUTH_CONTENT"]; !ok {
		t.Error("OPENCODE_AUTH_CONTENT missing with default data dir")
	}
}

// TestOpencodeAuthContent_Missing degrades quietly when no file exists.
func TestOpencodeAuthContent_Missing(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	env := authEnvTool(t).GetContainerEnv("/workspace")
	if _, ok := env["OPENCODE_AUTH_CONTENT"]; ok {
		t.Errorf("OPENCODE_AUTH_CONTENT present on missing file: %q", env["OPENCODE_AUTH_CONTENT"])
	}
}

// TestOpencodeAuthContent_Invalid skips a malformed auth.json instead of
// shipping empty/broken content into the container.
func TestOpencodeAuthContent_Invalid(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", writeAuth(t, dir, `not json {`))
	env := authEnvTool(t).GetContainerEnv("/workspace")
	if _, ok := env["OPENCODE_AUTH_CONTENT"]; ok {
		t.Errorf("OPENCODE_AUTH_CONTENT present on invalid JSON: %q", env["OPENCODE_AUTH_CONTENT"])
	}
}

// TestOpencodeAuthContent_XDGStillOverridden keeps the workspace DB persistence
// untouched.
func TestOpencodeAuthContent_XDGStillOverridden(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	env := authEnvTool(t).GetContainerEnv("/sandbox/path")
	if env["XDG_DATA_HOME"] != "/sandbox/path/.local/share" {
		t.Errorf("XDG_DATA_HOME = %q", env["XDG_DATA_HOME"])
	}
	if env["XDG_STATE_HOME"] != "/sandbox/path/.local/state" {
		t.Errorf("XDG_STATE_HOME = %q", env["XDG_STATE_HOME"])
	}
}


