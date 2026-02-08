package store

import "time"

// GoalStatus represents the completion state of a goal.
type GoalStatus string

const (
	StatusIncomplete GoalStatus = "incomplete"
	StatusInProgress GoalStatus = "in-progress"
	StatusComplete   GoalStatus = "complete"
)

// Horizon represents the temporal priority of a goal.
type Horizon string

const (
	HorizonToday    Horizon = "today"
	HorizonTomorrow Horizon = "tomorrow"
	HorizonFuture   Horizon = "future"
)

// Goal represents a goal or sub-goal loaded from a goal.md file.
type Goal struct {
	// Frontmatter fields
	Title         string            `yaml:"title"`
	Status        GoalStatus        `yaml:"status"`
	Horizon       Horizon           `yaml:"horizon,omitempty"`
	Created       time.Time         `yaml:"created"`
	Updated       time.Time         `yaml:"updated"`
	Tags          []string          `yaml:"tags,omitempty"`
	Links         map[string]string `yaml:"links,omitempty"`
	ChildrenOrder []string          `yaml:"children_order,omitempty"`

	// Parsed from markdown body
	Body string `yaml:"-"`

	// Filesystem metadata (not serialized to YAML)
	Slug     string  `yaml:"-"` // directory name
	Path     string  `yaml:"-"` // relative path from goals/ (e.g., "otr/ios")
	FilePath string  `yaml:"-"` // absolute path to goal.md
	Children []*Goal `yaml:"-"`
	Parent   *Goal   `yaml:"-"`
}

// IsComplete returns true if the goal is marked complete.
func (g *Goal) IsComplete() bool {
	return g.Status == StatusComplete
}

// IsInProgress returns true if the goal is in progress.
func (g *Goal) IsInProgress() bool {
	return g.Status == StatusInProgress
}

// FullPath returns the slash-separated path suitable for CLI commands.
func (g *Goal) FullPath() string {
	return g.Path
}

// Queue represents the ordered list of active work items.
type Queue struct {
	Updated time.Time `yaml:"updated"`
	Items   []string  // directory names under goals/
}
