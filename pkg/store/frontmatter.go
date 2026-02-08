package store

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

const frontmatterDelimiter = "---"

// ParseFrontmatter splits a markdown file into YAML frontmatter and body.
// Returns the parsed Goal and any error.
func ParseFrontmatter(content string) (*Goal, error) {
	content = strings.TrimSpace(content)

	if !strings.HasPrefix(content, frontmatterDelimiter) {
		// No frontmatter — treat entire content as body
		return &Goal{Body: content}, nil
	}

	// Find the closing delimiter
	rest := content[len(frontmatterDelimiter):]
	idx := strings.Index(rest, "\n"+frontmatterDelimiter)
	if idx == -1 {
		return nil, fmt.Errorf("unclosed frontmatter delimiter")
	}

	yamlContent := rest[:idx]
	body := rest[idx+len("\n"+frontmatterDelimiter):]
	body = strings.TrimLeft(body, "\n")

	var goal Goal
	if err := yaml.Unmarshal([]byte(yamlContent), &goal); err != nil {
		return nil, fmt.Errorf("parsing frontmatter YAML: %w", err)
	}

	goal.Body = body
	return &goal, nil
}

// SerializeFrontmatter renders a Goal back to markdown with YAML frontmatter.
func SerializeFrontmatter(g *Goal) (string, error) {
	yamlBytes, err := yaml.Marshal(g)
	if err != nil {
		return "", fmt.Errorf("serializing frontmatter YAML: %w", err)
	}

	var b strings.Builder
	b.WriteString(frontmatterDelimiter)
	b.WriteString("\n")
	b.WriteString(strings.TrimRight(string(yamlBytes), "\n"))
	b.WriteString("\n")
	b.WriteString(frontmatterDelimiter)
	b.WriteString("\n")
	if g.Body != "" {
		b.WriteString("\n")
		b.WriteString(g.Body)
		if !strings.HasSuffix(g.Body, "\n") {
			b.WriteString("\n")
		}
	}

	return b.String(), nil
}

// ParseQueue parses a queue.md file into a Queue struct.
func ParseQueue(content string) (*Queue, error) {
	content = strings.TrimSpace(content)

	var q Queue

	if strings.HasPrefix(content, frontmatterDelimiter) {
		rest := content[len(frontmatterDelimiter):]
		idx := strings.Index(rest, "\n"+frontmatterDelimiter)
		if idx == -1 {
			return nil, fmt.Errorf("unclosed frontmatter delimiter in queue.md")
		}

		yamlContent := rest[:idx]
		body := rest[idx+len("\n"+frontmatterDelimiter):]
		body = strings.TrimSpace(body)

		if err := yaml.Unmarshal([]byte(yamlContent), &q); err != nil {
			return nil, fmt.Errorf("parsing queue frontmatter: %w", err)
		}

		content = body
	}

	// Parse numbered list
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Strip leading "1. ", "2. ", etc.
		for i, c := range line {
			if c == '.' {
				item := strings.TrimSpace(line[i+1:])
				if item != "" {
					q.Items = append(q.Items, item)
				}
				break
			}
			if c < '0' || c > '9' {
				// Not a numbered list item, try as plain text
				q.Items = append(q.Items, line)
				break
			}
			if i == len([]rune(line))-1 {
				// All digits, no dot — treat as plain text
				q.Items = append(q.Items, line)
			}
		}
	}

	return &q, nil
}

// SerializeQueue renders a Queue back to markdown.
func SerializeQueue(q *Queue) string {
	var b strings.Builder
	b.WriteString(frontmatterDelimiter)
	b.WriteString("\n")
	yamlBytes, _ := yaml.Marshal(struct {
		Updated string `yaml:"updated"`
	}{
		Updated: q.Updated.Format("2006-01-02T15:04:05Z"),
	})
	b.WriteString(strings.TrimRight(string(yamlBytes), "\n"))
	b.WriteString("\n")
	b.WriteString(frontmatterDelimiter)
	b.WriteString("\n\n")

	for i, item := range q.Items {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, item))
	}

	return b.String()
}
