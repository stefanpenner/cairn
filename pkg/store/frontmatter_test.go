package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		check    func(t *testing.T, g *Goal)
	}{
		{
			name: "full frontmatter with body",
			input: `---
title: "iOS"
status: incomplete
created: 2026-02-08T10:00:00Z
updated: 2026-02-08T14:30:00Z
tags: [mobile, otr]
links:
  pr: "https://github.com/org/repo/pull/42"
---

# iOS

Notes about the iOS sub-goal.
`,
			check: func(t *testing.T, g *Goal) {
				assert.Equal(t, "iOS", g.Title)
				assert.Equal(t, StatusIncomplete, g.Status)
				assert.Equal(t, []string{"mobile", "otr"}, g.Tags)
				assert.Equal(t, "https://github.com/org/repo/pull/42", g.Links["pr"])
				assert.Contains(t, g.Body, "# iOS")
				assert.Contains(t, g.Body, "Notes about the iOS sub-goal.")
			},
		},
		{
			name: "with horizon field",
			input: `---
title: "Fix bug"
status: incomplete
horizon: today
created: 2026-02-08T10:00:00Z
updated: 2026-02-08T14:30:00Z
---

Quick fix needed.
`,
			check: func(t *testing.T, g *Goal) {
				assert.Equal(t, HorizonToday, g.Horizon)
				assert.Contains(t, g.Body, "Quick fix needed.")
			},
		},
		{
			name:  "no frontmatter",
			input: "Just some notes without frontmatter.",
			check: func(t *testing.T, g *Goal) {
				assert.Equal(t, "", g.Title)
				assert.Equal(t, "Just some notes without frontmatter.", g.Body)
			},
		},
		{
			name:    "unclosed frontmatter",
			input:   "---\ntitle: broken\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, err := ParseFrontmatter(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			tt.check(t, g)
		})
	}
}

func TestSerializeFrontmatter(t *testing.T) {
	g := &Goal{
		Title:   "iOS",
		Status:  StatusIncomplete,
		Horizon: HorizonToday,
		Created: time.Date(2026, 2, 8, 10, 0, 0, 0, time.UTC),
		Updated: time.Date(2026, 2, 8, 14, 30, 0, 0, time.UTC),
		Tags:    []string{"mobile", "otr"},
		Links:   map[string]string{"pr": "https://github.com/org/repo/pull/42"},
		Body:    "# iOS\n\nSome notes.\n",
	}

	content, err := SerializeFrontmatter(g)
	require.NoError(t, err)

	// Round-trip: parse back
	parsed, err := ParseFrontmatter(content)
	require.NoError(t, err)
	assert.Equal(t, g.Title, parsed.Title)
	assert.Equal(t, g.Status, parsed.Status)
	assert.Equal(t, g.Horizon, parsed.Horizon)
	assert.Equal(t, g.Tags, parsed.Tags)
	assert.Equal(t, g.Links["pr"], parsed.Links["pr"])
	assert.Contains(t, parsed.Body, "# iOS")
}

func TestParseQueue(t *testing.T) {
	input := `---
updated: 2026-02-08T14:30:00Z
---

1. otr
2. infra-migration
3. learn-rust
`
	q, err := ParseQueue(input)
	require.NoError(t, err)
	assert.Equal(t, []string{"otr", "infra-migration", "learn-rust"}, q.Items)
}

func TestSerializeQueue(t *testing.T) {
	q := &Queue{
		Updated: time.Date(2026, 2, 8, 14, 30, 0, 0, time.UTC),
		Items:   []string{"otr", "infra-migration"},
	}

	content := SerializeQueue(q)
	assert.Contains(t, content, "1. otr")
	assert.Contains(t, content, "2. infra-migration")

	// Round-trip
	parsed, err := ParseQueue(content)
	require.NoError(t, err)
	assert.Equal(t, q.Items, parsed.Items)
}
