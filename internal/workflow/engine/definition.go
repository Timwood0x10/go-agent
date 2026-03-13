package engine

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// Definition errors.
var (
	ErrFieldNotFound            = errors.New("field not found")
	ErrDuplicateAgentDefinition = errors.New("duplicate agent definition")
)

// AgentDefinition represents an agent definition from markdown.
type AgentDefinition struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Description string            `json:"description"`
	Prompts     map[string]string `json:"prompts"`
	Tools       []string          `json:"tools"`
	Metadata    map[string]string `json:"metadata"`
}

// DefinitionParser parses agent definitions from markdown files.
type DefinitionParser struct {
}

// NewDefinitionParser creates a new DefinitionParser.
func NewDefinitionParser() *DefinitionParser {
	return &DefinitionParser{}
}

// Parse parses an agent definition from a reader.
func (p *DefinitionParser) Parse(ctx context.Context, r io.Reader) (*AgentDefinition, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read content: %w", err)
	}

	return p.ParseBytes(ctx, content)
}

// ParseBytes parses an agent definition from bytes.
func (p *DefinitionParser) ParseBytes(ctx context.Context, content []byte) (*AgentDefinition, error) {
	text := string(content)

	def := &AgentDefinition{
		Prompts:  make(map[string]string),
		Tools:    make([]string, 0),
		Metadata: make(map[string]string),
	}

	name, err := p.extractField(text, "name")
	if err != nil {
		return nil, fmt.Errorf("extract name: %w", err)
	}
	def.Name = name

	agentType, err := p.extractField(text, "type")
	if err != nil {
		return nil, fmt.Errorf("extract type: %w", err)
	}
	def.Type = agentType

	description, err := p.extractField(text, "description")
	if err == nil {
		def.Description = description
	}

	def.Prompts = p.extractPrompts(text)

	def.Tools = p.extractTools(text)

	def.Metadata = p.extractMetadata(text)

	return def, nil
}

// ParseFile parses an agent definition from a file.
func (p *DefinitionParser) ParseFile(ctx context.Context, path string) (*AgentDefinition, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file %s: %w", path, err)
	}
	defer file.Close()

	return p.Parse(ctx, bufio.NewReader(file))
}

// extractField extracts a field value from markdown.
func (p *DefinitionParser) extractField(content, field string) (string, error) {
	patterns := []string{
		fmt.Sprintf(`(?i)%s\s*::\s*(.+?)(?:\n|$)`, field),
		fmt.Sprintf(`(?i)%s\s*:\s*(.+?)(?:\n|$)`, field),
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(content)
		if len(matches) > 1 {
			return strings.TrimSpace(matches[1]), nil
		}
	}

	return "", ErrFieldNotFound
}

// extractPrompts extracts prompts from markdown.
func (p *DefinitionParser) extractPrompts(content string) map[string]string {
	prompts := make(map[string]string)

	promptPattern := regexp.MustCompile(`(?si)##\s*Prompt\s*:(\w+)\s*\n(.*?)(?=##|\z)`)
	matches := promptPattern.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 2 {
			promptName := strings.TrimSpace(match[1])
			promptContent := strings.TrimSpace(match[2])
			prompts[promptName] = promptContent
		}
	}

	return prompts
}

// extractTools extracts tool names from markdown.
func (p *DefinitionParser) extractTools(content string) []string {
	tools := make([]string, 0)

	toolPattern := regexp.MustCompile(`(?i)##\s*Tools\s*\n(.*?)(?=##|\z)`)
	matches := toolPattern.FindStringSubmatch(content)

	if len(matches) > 1 {
		toolList := matches[1]
		toolItemPattern := regexp.MustCompile(`(?m)^\s*-\s*(.+?)$`)
		toolMatches := toolItemPattern.FindAllStringSubmatch(toolList, -1)

		for _, toolMatch := range toolMatches {
			if len(toolMatch) > 1 {
				toolName := strings.TrimSpace(toolMatch[1])
				tools = append(tools, toolName)
			}
		}
	}

	return tools
}

// extractMetadata extracts metadata from markdown.
func (p *DefinitionParser) extractMetadata(content string) map[string]string {
	metadata := make(map[string]string)

	metaPattern := regexp.MustCompile(`(?i)##\s*Metadata\s*\n(.*?)(?=##|\z)`)
	matches := metaPattern.FindStringSubmatch(content)

	if len(matches) > 1 {
		metaContent := matches[1]
		metaItemPattern := regexp.MustCompile(`(?m)^\s*-\s*(\w+)\s*:\s*(.+?)$`)
		metaMatches := metaItemPattern.FindAllStringSubmatch(metaContent, -1)

		for _, metaMatch := range metaMatches {
			if len(metaMatch) > 2 {
				key := strings.TrimSpace(metaMatch[1])
				value := strings.TrimSpace(metaMatch[2])
				metadata[key] = value
			}
		}
	}

	return metadata
}

// DirectoryParser parses agent definitions from a directory.
type DirectoryParser struct {
	parser *DefinitionParser
}

// NewDirectoryParser creates a new DirectoryParser.
func NewDirectoryParser(parser *DefinitionParser) *DirectoryParser {
	return &DirectoryParser{
		parser: parser,
	}
}

// ParseAll parses all agent definitions from a directory.
func (p *DirectoryParser) ParseAll(ctx context.Context, dir string) (map[string]*AgentDefinition, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read directory %s: %w", dir, err)
	}

	definitions := make(map[string]*AgentDefinition)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		path := dir + "/" + entry.Name()
		def, err := p.parser.ParseFile(ctx, path)
		if err != nil {
			return nil, fmt.Errorf("parse file %s: %w", path, err)
		}

		if _, exists := definitions[def.Name]; exists {
			return nil, ErrDuplicateAgentDefinition
		}

		definitions[def.Name] = def
	}

	return definitions, nil
}
