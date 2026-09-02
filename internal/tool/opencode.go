package tool

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
)

// OpencodeTool implements Tool for opencode (https://opencode.ai)
type OpencodeTool struct {
	permissionMode  string // "bypass" (default) or "interactive"
	contextFilePath string // absolute path to sandbox context file inside container (set by SetAutoContextPath)
}

// NewOpencode creates a new opencode tool instance
func NewOpencode() Tool { return &OpencodeTool{} }

func (c *OpencodeTool) Name() string { return "opencode" }

func (c *OpencodeTool) Binary() string { return "opencode" }

// ConfigDirName returns the XDG-standard config directory for opencode.
func (c *OpencodeTool) ConfigDirName() string { return mustBundle("opencode").ConfigDir }

func (c *OpencodeTool) SessionsDirName() string { return "sessions-opencode" }

// BuildCommand builds the opencode launch command.
// When resume is true, passes --continue to auto-resume the last session,
// or --session <id> if a specific session ID is provided.
func (c *OpencodeTool) BuildCommand(sessionID string, resume bool, resumeSessionID string) []string {
	cmd := []string{"opencode"}
	if resume {
		if resumeSessionID != "" {
			cmd = append(cmd, "--session", resumeSessionID)
		} else {
			cmd = append(cmd, "--continue")
		}
	}
	return cmd
}

// DiscoverSessionID returns "" because opencode uses SQLite (not JSONL files).
func (c *OpencodeTool) DiscoverSessionID(stateDir string) string { return "" }

// GetSandboxSettings returns the opencode permission config.
// In bypass mode (default): injects {"permission": {"*": "allow"}} so opencode runs
// without interactive prompts.
// In interactive mode: injects {"permission": {"*": "ask"}} so opencode prompts the
// user before each action (human-in-the-loop).
func (c *OpencodeTool) GetSandboxSettings() map[string]interface{} {
	var settings map[string]interface{}
	if c.permissionMode == "interactive" {
		settings = map[string]interface{}{
			"permission": map[string]interface{}{
				"*": "ask",
			},
		}
	} else {
		settings = map[string]interface{}{
			"permission": map[string]interface{}{
				"*": "allow",
			},
		}
	}

	// Include instructions field referencing the sandbox context file
	if c.contextFilePath != "" {
		settings["instructions"] = []string{c.contextFilePath}
	}

	return settings
}

// SetPermissionMode sets the permission mode for opencode.
// Valid values: "bypass" (default) or "interactive" (human-in-the-loop).
func (c *OpencodeTool) SetPermissionMode(mode string) {
	c.permissionMode = mode
}

// SetAutoContextPath implements ToolWithAutoContextPath.
// Stores the absolute path to the sandbox context file so it can be
// referenced in the opencode.json instructions field.
func (c *OpencodeTool) SetAutoContextPath(path string) {
	c.contextFilePath = path
}

// EssentialConfigFiles implements ToolWithConfigDirFiles.
func (c *OpencodeTool) EssentialConfigFiles() []string {
	return mustBundle("opencode").Files
}

// SandboxSettingsFileName implements ToolWithConfigDirFiles.
func (c *OpencodeTool) SandboxSettingsFileName() string {
	return mustBundle("opencode").SandboxSettingsFile
}

// StateConfigFileName implements ToolWithConfigDirFiles.
// Opencode has no sibling state file.
func (c *OpencodeTool) StateConfigFileName() string { return mustBundle("opencode").StateFile }

// AlwaysSetupConfig implements ToolWithConfigDirFiles.
// Opencode needs sandbox permission bypass even without host config dir.
func (c *OpencodeTool) AlwaysSetupConfig() bool { return mustBundle("opencode").AlwaysSetup }

// GetContainerEnv implements ToolWithContainerEnv.
// Redirects XDG data and state directories to the workspace mount so opencode's
// SQLite database persists across ephemeral container recreations.
// Without this, data lives in ~/.local/share/opencode/ (inside the container)
// and is destroyed when the ephemeral container is deleted.
//
// Also exports OPENCODE_AUTH_CONTENT with the host's auth.json, compacted to a
// single line. opencode prefers this env var over reading the auth file, so
// credentials are never copied into the container. A missing or invalid host
// auth.json degrades to the tool's default (no credential).
func (c *OpencodeTool) GetContainerEnv(workspacePath string) map[string]string {
	env := map[string]string{
		"XDG_DATA_HOME":  filepath.Join(workspacePath, ".local", "share"),
		"XDG_STATE_HOME": filepath.Join(workspacePath, ".local", "state"),
	}
	if auth := readHostAuthJSON(); auth != "" {
		env["OPENCODE_AUTH_CONTENT"] = auth
	}
	return env
}

// readHostAuthJSON returns the host's opencode auth.json (from opencode's data
// dir: $XDG_DATA_HOME/opencode/auth.json, defaulting to ~/.local/share) as a
// compact, single-line JSON string, or "" when the file is missing or invalid.
// The container env sanitizer (session.planToolContainerEnv) rejects values
// containing newlines, so pretty-printed JSON has to be flattened here.
func readHostAuthJSON() string {
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			dataDir = filepath.Join(home, ".local", "share")
		}
	}
	content, err := os.ReadFile(filepath.Join(dataDir, "opencode", "auth.json"))
	if err != nil {
		return ""
	}
	var buf bytes.Buffer
	if err := json.Compact(&buf, content); err != nil {
		return ""
	}
	return buf.String()
}
