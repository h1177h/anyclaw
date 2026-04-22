package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const defaultBootstrapMaxChars = 20_000

type bootstrapSeed struct {
	AgentDescription string
	AgentName        string
	AssistantStyle   string
	Constraints      string
	UserProfile      string
	WorkspaceFocus   string
}

type bootstrapFile struct {
	Name      string
	Content   string
	Missing   bool
	Truncated bool
}

var bootstrapFileOrder = []string{
	"AGENTS.md",
	"SOUL.md",
	"TOOLS.md",
	"IDENTITY.md",
	"USER.md",
	"HEARTBEAT.md",
	"BOOTSTRAP.md",
	"MEMORY.md",
}

func ensureBootstrapFiles(dir string, seed bootstrapSeed) error {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(dir, "memory"), 0o755); err != nil {
		return err
	}

	templates := defaultBootstrapTemplates(seed)
	for _, name := range bootstrapFileOrder {
		path := filepath.Join(dir, name)
		if fileExists(path) {
			continue
		}
		content, ok := templates[name]
		if !ok {
			continue
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func loadBootstrapFiles(dir string, maxChars int) ([]bootstrapFile, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, nil
	}
	if maxChars <= 0 {
		maxChars = defaultBootstrapMaxChars
	}

	files := make([]bootstrapFile, 0, len(bootstrapFileOrder))
	for _, name := range bootstrapFileOrder {
		actualName := name
		path := filepath.Join(dir, name)
		if name == "MEMORY.md" && !fileExists(path) {
			fallback := filepath.Join(dir, "memory.md")
			if fileExists(fallback) {
				path = fallback
				actualName = "memory.md"
			}
		}

		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				files = append(files, bootstrapFile{
					Name:    name,
					Missing: true,
					Content: fmt.Sprintf("(missing workspace file: %s)", name),
				})
				continue
			}
			return nil, err
		}

		content := strings.TrimSpace(normalizeBootstrapNewlines(string(data)))
		truncated := false
		if utf8.RuneCountInString(content) > maxChars {
			content = truncateBootstrapRunes(content, maxChars)
			content = strings.TrimSpace(content) + "\n\n[truncated]"
			truncated = true
		}
		if content == "" {
			content = "(empty)"
		}
		files = append(files, bootstrapFile{
			Name:      actualName,
			Content:   content,
			Truncated: truncated,
		})
	}

	return files, nil
}

func defaultBootstrapTemplates(seed bootstrapSeed) map[string]string {
	name := strings.TrimSpace(seed.AgentName)
	if name == "" {
		name = "AnyClaw"
	}
	description := strings.TrimSpace(seed.AgentDescription)
	if description == "" {
		description = "Execution-oriented local AI assistant."
	}

	userNotes := firstNonEmpty(strings.TrimSpace(seed.UserProfile), "Add durable user preferences here.")
	workspaceFocus := firstNonEmpty(strings.TrimSpace(seed.WorkspaceFocus), "General")
	assistantStyle := firstNonEmpty(strings.TrimSpace(seed.AssistantStyle), "Direct and collaborative")
	constraints := firstNonEmpty(strings.TrimSpace(seed.Constraints), "List any execution constraints here.")

	return map[string]string{
		"AGENTS.md": strings.TrimSpace(fmt.Sprintf(`# AGENTS

## Primary Agent
- Name: %s
- Goal: Complete the user's task safely, end to end, and verify the real outcome.
`, name)),
		"SOUL.md": strings.TrimSpace(fmt.Sprintf(`# SOUL

- Identity: %s
- Description: %s
- Style: Calm, direct, action-oriented, and collaborative.
`, name, description)),
		"TOOLS.md": `# TOOLS

- Prefer current file state and command output over assumptions.
- Verify important side effects instead of assuming success.
- Use destructive actions only with explicit approval or clear policy coverage.
`,
		"IDENTITY.md": strings.TrimSpace(fmt.Sprintf(`# IDENTITY

- Agent: %s
- Description: %s
- Workspace Focus: %s
- Assistant Style: %s
`, name, description, workspaceFocus, assistantStyle)),
		"USER.md": strings.TrimSpace(fmt.Sprintf(`# USER

%s
`, userNotes)),
		"HEARTBEAT.md": strings.TrimSpace(fmt.Sprintf(`# HEARTBEAT

- During longer work, send brief progress updates.
- Respect these constraints: %s
`, constraints)),
		"BOOTSTRAP.md": `# BOOTSTRAP

Review and personalize these files:
- AGENTS.md
- SOUL.md
- IDENTITY.md
- USER.md
- TOOLS.md
- MEMORY.md
`,
		"MEMORY.md": `# MEMORY

No durable project memory has been captured yet.
`,
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func normalizeBootstrapNewlines(input string) string {
	input = strings.ReplaceAll(input, "\r\n", "\n")
	return strings.ReplaceAll(input, "\r", "\n")
}

func truncateBootstrapRunes(input string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(input)
	if len(runes) <= limit {
		return input
	}
	return string(runes[:limit])
}
