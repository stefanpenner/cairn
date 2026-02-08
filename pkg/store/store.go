package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Store manages the filesystem-backed goal data.
type Store struct {
	Root string // e.g., ~/.cairn
}

// NewStore creates a Store rooted at the given directory.
// It creates the directory structure if it doesn't exist.
func NewStore(root string) (*Store, error) {
	goalsDir := filepath.Join(root, "goals")
	if err := os.MkdirAll(goalsDir, 0755); err != nil {
		return nil, fmt.Errorf("creating goals directory: %w", err)
	}
	return &Store{Root: root}, nil
}

// GoalsDir returns the path to the goals directory.
func (s *Store) GoalsDir() string {
	return filepath.Join(s.Root, "goals")
}

// QueuePath returns the path to queue.md.
func (s *Store) QueuePath() string {
	return filepath.Join(s.Root, "queue.md")
}

// LoadQueue reads and parses queue.md.
func (s *Store) LoadQueue() (*Queue, error) {
	data, err := os.ReadFile(s.QueuePath())
	if os.IsNotExist(err) {
		return &Queue{Updated: time.Now()}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading queue.md: %w", err)
	}
	return ParseQueue(string(data))
}

// SaveQueue writes queue.md to disk.
func (s *Store) SaveQueue(q *Queue) error {
	q.Updated = time.Now()
	content := SerializeQueue(q)
	return os.WriteFile(s.QueuePath(), []byte(content), 0644)
}

// LoadGoal reads a single goal from its directory path (relative to goals/).
func (s *Store) LoadGoal(goalPath string) (*Goal, error) {
	filePath := filepath.Join(s.GoalsDir(), goalPath, "goal.md")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading goal %s: %w", goalPath, err)
	}

	goal, err := ParseFrontmatter(string(data))
	if err != nil {
		return nil, fmt.Errorf("parsing goal %s: %w", goalPath, err)
	}

	goal.Slug = filepath.Base(goalPath)
	goal.Path = goalPath
	goal.FilePath = filePath
	return goal, nil
}

// LoadGoalTree loads the entire goal hierarchy from disk.
func (s *Store) LoadGoalTree() ([]*Goal, error) {
	goalsDir := s.GoalsDir()
	entries, err := os.ReadDir(goalsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading goals directory: %w", err)
	}

	// Load all top-level goals into a map
	goalMap := make(map[string]*Goal)
	var defaultOrder []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		goal, err := s.loadGoalRecursive(entry.Name(), nil)
		if err != nil {
			continue // skip broken goals
		}
		goalMap[entry.Name()] = goal
		defaultOrder = append(defaultOrder, entry.Name())
	}

	// Check for a top-level ordering file (goal.md in goals/ directory)
	var topOrder []string
	topGoalPath := filepath.Join(goalsDir, "goal.md")
	if data, err := os.ReadFile(topGoalPath); err == nil {
		if topGoal, err := ParseFrontmatter(string(data)); err == nil {
			topOrder = topGoal.ChildrenOrder
		}
	}

	var goals []*Goal
	if len(topOrder) > 0 {
		seen := make(map[string]bool)
		for _, name := range topOrder {
			if g, ok := goalMap[name]; ok {
				goals = append(goals, g)
				seen[name] = true
			}
		}
		// Append any not in the ordering
		for _, name := range defaultOrder {
			if !seen[name] {
				goals = append(goals, goalMap[name])
			}
		}
	} else {
		for _, name := range defaultOrder {
			goals = append(goals, goalMap[name])
		}
	}

	return goals, nil
}

func (s *Store) loadGoalRecursive(goalPath string, parent *Goal) (*Goal, error) {
	goal, err := s.LoadGoal(goalPath)
	if err != nil {
		// If no goal.md exists, create a minimal goal from the directory name
		goal = &Goal{
			Title:  filepath.Base(goalPath),
			Status: StatusIncomplete,
			Slug:   filepath.Base(goalPath),
			Path:   goalPath,
		}
	}
	goal.Parent = parent

	// Look for child directories
	dir := filepath.Join(s.GoalsDir(), goalPath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return goal, nil
	}

	// Build a map of child name -> loaded child
	childMap := make(map[string]*Goal)
	var defaultOrder []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		childPath := filepath.Join(goalPath, entry.Name())
		child, err := s.loadGoalRecursive(childPath, goal)
		if err != nil {
			continue
		}
		childMap[entry.Name()] = child
		defaultOrder = append(defaultOrder, entry.Name())
	}

	// Use children_order if present, falling back to alphabetical (os.ReadDir order)
	if len(goal.ChildrenOrder) > 0 {
		seen := make(map[string]bool)
		for _, name := range goal.ChildrenOrder {
			if child, ok := childMap[name]; ok {
				goal.Children = append(goal.Children, child)
				seen[name] = true
			}
		}
		// Append any children not listed in children_order
		for _, name := range defaultOrder {
			if !seen[name] {
				goal.Children = append(goal.Children, childMap[name])
			}
		}
	} else {
		for _, name := range defaultOrder {
			goal.Children = append(goal.Children, childMap[name])
		}
	}

	return goal, nil
}

// SaveGoal writes a goal to disk.
func (s *Store) SaveGoal(g *Goal) error {
	g.Updated = time.Now()

	dir := filepath.Join(s.GoalsDir(), g.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating goal directory: %w", err)
	}

	content, err := SerializeFrontmatter(g)
	if err != nil {
		return fmt.Errorf("serializing goal: %w", err)
	}

	filePath := filepath.Join(dir, "goal.md")
	g.FilePath = filePath
	return os.WriteFile(filePath, []byte(content), 0644)
}

// CreateGoal creates a new goal under the given parent path.
// If parentPath is empty, creates a top-level goal.
func (s *Store) CreateGoal(parentPath, slug string) (*Goal, error) {
	slug = strings.ToLower(strings.ReplaceAll(slug, " ", "-"))

	var goalPath string
	if parentPath == "" {
		goalPath = slug
	} else {
		goalPath = filepath.Join(parentPath, slug)
	}

	dir := filepath.Join(s.GoalsDir(), goalPath)
	if _, err := os.Stat(dir); err == nil {
		return nil, fmt.Errorf("goal %s already exists", goalPath)
	}

	now := time.Now()
	goal := &Goal{
		Title:   slug,
		Status:  StatusIncomplete,
		Horizon: HorizonFuture,
		Created: now,
		Updated: now,
		Slug:    slug,
		Path:    goalPath,
	}

	if err := s.SaveGoal(goal); err != nil {
		return nil, err
	}

	return goal, nil
}

// DeleteGoal removes a goal directory and all its children.
func (s *Store) DeleteGoal(goalPath string) error {
	dir := filepath.Join(s.GoalsDir(), goalPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("goal %s not found", goalPath)
	}
	return os.RemoveAll(dir)
}

// ToggleStatus cycles a goal through incomplete → in-progress → complete → incomplete.
func (s *Store) ToggleStatus(goalPath string) (*Goal, error) {
	goal, err := s.LoadGoal(goalPath)
	if err != nil {
		return nil, err
	}

	switch goal.Status {
	case StatusIncomplete:
		goal.Status = StatusInProgress
	case StatusInProgress:
		goal.Status = StatusComplete
	default:
		goal.Status = StatusIncomplete
	}

	if err := s.SaveGoal(goal); err != nil {
		return nil, err
	}
	return goal, nil
}

// SetHorizon sets the temporal horizon of a goal.
func (s *Store) SetHorizon(goalPath string, horizon Horizon) (*Goal, error) {
	goal, err := s.LoadGoal(goalPath)
	if err != nil {
		return nil, err
	}

	goal.Horizon = horizon
	if err := s.SaveGoal(goal); err != nil {
		return nil, err
	}
	return goal, nil
}

// AddNote appends a note entry to a goal's body.
func (s *Store) AddNote(goalPath, text string) (*Goal, error) {
	goal, err := s.LoadGoal(goalPath)
	if err != nil {
		return nil, err
	}

	today := time.Now().Format("2006-01-02")
	dateHeader := fmt.Sprintf("## %s", today)

	if strings.Contains(goal.Body, dateHeader) {
		// Append under existing date header
		idx := strings.Index(goal.Body, dateHeader)
		afterHeader := idx + len(dateHeader)
		// Find end of line
		nlIdx := strings.Index(goal.Body[afterHeader:], "\n")
		if nlIdx == -1 {
			goal.Body += "\n- " + text + "\n"
		} else {
			insertAt := afterHeader + nlIdx + 1
			goal.Body = goal.Body[:insertAt] + "- " + text + "\n" + goal.Body[insertAt:]
		}
	} else {
		// Add new date header
		if goal.Body != "" && !strings.HasSuffix(goal.Body, "\n") {
			goal.Body += "\n"
		}
		if goal.Body != "" {
			goal.Body += "\n"
		}
		goal.Body += dateHeader + "\n- " + text + "\n"
	}

	if err := s.SaveGoal(goal); err != nil {
		return nil, err
	}
	return goal, nil
}

// SearchNotes searches across all goals for matching text.
func (s *Store) SearchNotes(query string) ([]*Goal, error) {
	allGoals, err := s.LoadGoalTree()
	if err != nil {
		return nil, err
	}

	query = strings.ToLower(query)
	var matches []*Goal

	var search func(goals []*Goal)
	search = func(goals []*Goal) {
		for _, g := range goals {
			if strings.Contains(strings.ToLower(g.Title), query) ||
				strings.Contains(strings.ToLower(g.Body), query) {
				matches = append(matches, g)
			}
			search(g.Children)
		}
	}
	search(allGoals)

	return matches, nil
}

// ReorderGoal swaps a goal with a sibling in the given direction (delta: -1 for up, +1 for down).
// It updates the parent's children_order field in frontmatter. For top-level goals, it updates
// goals/goal.md.
func (s *Store) ReorderGoal(goalPath string, delta int) error {
	slug := filepath.Base(goalPath)
	parentPath := filepath.Dir(goalPath)
	if parentPath == "." {
		parentPath = ""
	}

	// Get the current sibling order
	siblings, err := s.getSiblingOrder(parentPath)
	if err != nil {
		return err
	}

	// Find the goal's index
	idx := -1
	for i, name := range siblings {
		if name == slug {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("goal %s not found among siblings", slug)
	}

	newIdx := idx + delta
	if newIdx < 0 || newIdx >= len(siblings) {
		return nil // at boundary, nothing to do
	}

	// Swap
	siblings[idx], siblings[newIdx] = siblings[newIdx], siblings[idx]

	// Save the updated order
	return s.saveChildrenOrder(parentPath, siblings)
}

// MoveGoal moves a goal directory to a new parent.
// If newParentPath is empty, it becomes a top-level goal.
func (s *Store) MoveGoal(goalPath, newParentPath string) error {
	slug := filepath.Base(goalPath)
	oldParentPath := filepath.Dir(goalPath)
	if oldParentPath == "." {
		oldParentPath = ""
	}

	// Prevent moving into self or a descendant
	if newParentPath == goalPath || strings.HasPrefix(newParentPath, goalPath+string(filepath.Separator)) {
		return fmt.Errorf("cannot move a goal into itself or a descendant")
	}

	// Build new path
	var newGoalPath string
	if newParentPath == "" {
		newGoalPath = slug
	} else {
		newGoalPath = filepath.Join(newParentPath, slug)
	}

	// Check for conflict at destination
	dstDir := filepath.Join(s.GoalsDir(), newGoalPath)
	if _, err := os.Stat(dstDir); err == nil {
		return fmt.Errorf("goal %s already exists at destination", newGoalPath)
	}

	// Ensure destination parent directory exists
	dstParentDir := filepath.Join(s.GoalsDir(), newParentPath)
	if newParentPath != "" {
		if _, err := os.Stat(dstParentDir); os.IsNotExist(err) {
			return fmt.Errorf("destination parent %s does not exist", newParentPath)
		}
	}

	// Move the directory
	srcDir := filepath.Join(s.GoalsDir(), goalPath)
	if err := os.Rename(srcDir, dstDir); err != nil {
		return fmt.Errorf("moving goal directory: %w", err)
	}

	// Update the moved goal's internal path references in goal.md
	s.updateGoalPaths(newGoalPath)

	// Remove the goal from old parent's children_order
	s.removeFromChildrenOrder(oldParentPath, slug)

	// Add the goal to new parent's children_order
	s.addToChildrenOrder(newParentPath, slug)

	return nil
}

// getSiblingOrder returns the ordered list of child directory names for a parent path.
// If children_order is set, it uses that; otherwise falls back to directory listing order.
func (s *Store) getSiblingOrder(parentPath string) ([]string, error) {
	var dir string
	if parentPath == "" {
		dir = s.GoalsDir()
	} else {
		dir = filepath.Join(s.GoalsDir(), parentPath)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", dir, err)
	}

	var dirNames []string
	for _, e := range entries {
		if e.IsDir() {
			dirNames = append(dirNames, e.Name())
		}
	}

	// Check for existing children_order
	var order []string
	if parentPath == "" {
		// Top-level: check goals/goal.md
		topGoalPath := filepath.Join(s.GoalsDir(), "goal.md")
		if data, err := os.ReadFile(topGoalPath); err == nil {
			if topGoal, err := ParseFrontmatter(string(data)); err == nil {
				order = topGoal.ChildrenOrder
			}
		}
	} else {
		goal, err := s.LoadGoal(parentPath)
		if err == nil {
			order = goal.ChildrenOrder
		}
	}

	if len(order) > 0 {
		// Merge: use order first, then append any not in order
		seen := make(map[string]bool)
		dirSet := make(map[string]bool)
		for _, d := range dirNames {
			dirSet[d] = true
		}
		var result []string
		for _, name := range order {
			if dirSet[name] {
				result = append(result, name)
				seen[name] = true
			}
		}
		for _, name := range dirNames {
			if !seen[name] {
				result = append(result, name)
			}
		}
		return result, nil
	}

	return dirNames, nil
}

// saveChildrenOrder persists the children_order to the appropriate goal.md.
func (s *Store) saveChildrenOrder(parentPath string, order []string) error {
	if parentPath == "" {
		// Top-level: save to goals/goal.md
		topGoalPath := filepath.Join(s.GoalsDir(), "goal.md")
		var goal *Goal
		if data, err := os.ReadFile(topGoalPath); err == nil {
			goal, _ = ParseFrontmatter(string(data))
		}
		if goal == nil {
			goal = &Goal{}
		}
		goal.ChildrenOrder = order
		content, err := SerializeFrontmatter(goal)
		if err != nil {
			return err
		}
		return os.WriteFile(topGoalPath, []byte(content), 0644)
	}

	goal, err := s.LoadGoal(parentPath)
	if err != nil {
		// Create minimal goal if it doesn't exist
		goal = &Goal{
			Title:  filepath.Base(parentPath),
			Status: StatusIncomplete,
			Slug:   filepath.Base(parentPath),
			Path:   parentPath,
		}
	}
	goal.ChildrenOrder = order
	return s.SaveGoal(goal)
}

// removeFromChildrenOrder removes a slug from a parent's children_order.
func (s *Store) removeFromChildrenOrder(parentPath, slug string) {
	order, err := s.getSiblingOrder(parentPath)
	if err != nil {
		return
	}
	var newOrder []string
	for _, name := range order {
		if name != slug {
			newOrder = append(newOrder, name)
		}
	}
	s.saveChildrenOrder(parentPath, newOrder)
}

// addToChildrenOrder appends a slug to a parent's children_order.
func (s *Store) addToChildrenOrder(parentPath, slug string) {
	order, err := s.getSiblingOrder(parentPath)
	if err != nil {
		return
	}
	// Only add if not already present
	for _, name := range order {
		if name == slug {
			return
		}
	}
	order = append(order, slug)
	s.saveChildrenOrder(parentPath, order)
}

// updateGoalPaths recursively updates path references in goal.md files after a move.
func (s *Store) updateGoalPaths(goalPath string) {
	// We don't need to update the file contents since Path/Slug are not stored in frontmatter.
	// They are derived from the filesystem at load time. This is a no-op.
}

// GoalsByHorizon returns goals grouped by their temporal horizon.
func (s *Store) GoalsByHorizon() (today, tomorrow, future []*Goal, err error) {
	allGoals, err := s.LoadGoalTree()
	if err != nil {
		return nil, nil, nil, err
	}

	var categorize func(goals []*Goal)
	categorize = func(goals []*Goal) {
		for _, g := range goals {
			switch g.Horizon {
			case HorizonToday:
				today = append(today, g)
			case HorizonTomorrow:
				tomorrow = append(tomorrow, g)
			default:
				future = append(future, g)
			}
			categorize(g.Children)
		}
	}
	categorize(allGoals)

	return today, tomorrow, future, nil
}
