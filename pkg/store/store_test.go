package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := NewStore(dir)
	require.NoError(t, err)
	return s
}

func TestCreateGoal(t *testing.T) {
	s := setupTestStore(t)

	goal, err := s.CreateGoal("", "my-project")
	require.NoError(t, err)
	assert.Equal(t, "my-project", goal.Slug)
	assert.Equal(t, "my-project", goal.Path)
	assert.Equal(t, StatusIncomplete, goal.Status)
	assert.Equal(t, HorizonFuture, goal.Horizon)

	// File should exist
	_, err = os.Stat(filepath.Join(s.GoalsDir(), "my-project", "goal.md"))
	assert.NoError(t, err)
}

func TestCreateSubGoal(t *testing.T) {
	s := setupTestStore(t)

	_, err := s.CreateGoal("", "otr")
	require.NoError(t, err)

	child, err := s.CreateGoal("otr", "ios")
	require.NoError(t, err)
	assert.Equal(t, "ios", child.Slug)
	assert.Equal(t, filepath.Join("otr", "ios"), child.Path)
}

func TestCreateGoalDuplicate(t *testing.T) {
	s := setupTestStore(t)

	_, err := s.CreateGoal("", "otr")
	require.NoError(t, err)

	_, err = s.CreateGoal("", "otr")
	assert.Error(t, err)
}

func TestLoadGoalTree(t *testing.T) {
	s := setupTestStore(t)

	_, err := s.CreateGoal("", "otr")
	require.NoError(t, err)
	_, err = s.CreateGoal("otr", "ios")
	require.NoError(t, err)
	_, err = s.CreateGoal("otr", "android")
	require.NoError(t, err)
	_, err = s.CreateGoal("", "infra")
	require.NoError(t, err)

	goals, err := s.LoadGoalTree()
	require.NoError(t, err)
	assert.Len(t, goals, 2) // otr and infra

	// Find otr
	var otr *Goal
	for _, g := range goals {
		if g.Slug == "otr" {
			otr = g
			break
		}
	}
	require.NotNil(t, otr)
	assert.Len(t, otr.Children, 2)
}

func TestToggleStatus(t *testing.T) {
	s := setupTestStore(t)

	_, err := s.CreateGoal("", "test")
	require.NoError(t, err)

	// incomplete → in-progress
	goal, err := s.ToggleStatus("test")
	require.NoError(t, err)
	assert.Equal(t, StatusInProgress, goal.Status)

	// in-progress → complete
	goal, err = s.ToggleStatus("test")
	require.NoError(t, err)
	assert.Equal(t, StatusComplete, goal.Status)

	// complete → incomplete
	goal, err = s.ToggleStatus("test")
	require.NoError(t, err)
	assert.Equal(t, StatusIncomplete, goal.Status)
}

func TestSetHorizon(t *testing.T) {
	s := setupTestStore(t)

	_, err := s.CreateGoal("", "test")
	require.NoError(t, err)

	goal, err := s.SetHorizon("test", HorizonToday)
	require.NoError(t, err)
	assert.Equal(t, HorizonToday, goal.Horizon)

	// Reload and verify persistence
	goal, err = s.LoadGoal("test")
	require.NoError(t, err)
	assert.Equal(t, HorizonToday, goal.Horizon)
}

func TestAddNote(t *testing.T) {
	s := setupTestStore(t)

	_, err := s.CreateGoal("", "test")
	require.NoError(t, err)

	goal, err := s.AddNote("test", "First note")
	require.NoError(t, err)
	assert.Contains(t, goal.Body, "- First note")

	// Add another note on same day
	goal, err = s.AddNote("test", "Second note")
	require.NoError(t, err)
	assert.Contains(t, goal.Body, "- First note")
	assert.Contains(t, goal.Body, "- Second note")
}

func TestDeleteGoal(t *testing.T) {
	s := setupTestStore(t)

	_, err := s.CreateGoal("", "test")
	require.NoError(t, err)
	_, err = s.CreateGoal("test", "child")
	require.NoError(t, err)

	err = s.DeleteGoal("test")
	assert.NoError(t, err)

	// Should be gone
	_, err = s.LoadGoal("test")
	assert.Error(t, err)
}

func TestQueue(t *testing.T) {
	s := setupTestStore(t)

	q, err := s.LoadQueue()
	require.NoError(t, err)
	assert.Empty(t, q.Items) // empty when no file

	q.Items = []string{"otr", "infra"}
	err = s.SaveQueue(q)
	require.NoError(t, err)

	q2, err := s.LoadQueue()
	require.NoError(t, err)
	assert.Equal(t, []string{"otr", "infra"}, q2.Items)
}

func TestSearchNotes(t *testing.T) {
	s := setupTestStore(t)

	_, err := s.CreateGoal("", "project-a")
	require.NoError(t, err)
	_, err = s.AddNote("project-a", "Fix the authentication bug")
	require.NoError(t, err)

	_, err = s.CreateGoal("", "project-b")
	require.NoError(t, err)
	_, err = s.AddNote("project-b", "Write documentation")
	require.NoError(t, err)

	matches, err := s.SearchNotes("authentication")
	require.NoError(t, err)
	assert.Len(t, matches, 1)
	assert.Equal(t, "project-a", matches[0].Slug)
}

func TestReorderGoal(t *testing.T) {
	s := setupTestStore(t)

	// Create three top-level goals: alpha, beta, gamma (alphabetical order by default)
	_, err := s.CreateGoal("", "alpha")
	require.NoError(t, err)
	_, err = s.CreateGoal("", "beta")
	require.NoError(t, err)
	_, err = s.CreateGoal("", "gamma")
	require.NoError(t, err)

	// Default order: alpha, beta, gamma
	goals, err := s.LoadGoalTree()
	require.NoError(t, err)
	require.Len(t, goals, 3)
	assert.Equal(t, "alpha", goals[0].Slug)
	assert.Equal(t, "beta", goals[1].Slug)
	assert.Equal(t, "gamma", goals[2].Slug)

	// Move beta up (swap with alpha)
	err = s.ReorderGoal("beta", -1)
	require.NoError(t, err)

	goals, err = s.LoadGoalTree()
	require.NoError(t, err)
	assert.Equal(t, "beta", goals[0].Slug)
	assert.Equal(t, "alpha", goals[1].Slug)
	assert.Equal(t, "gamma", goals[2].Slug)

	// Move beta down (swap with alpha, so back to alpha, beta order for first two)
	err = s.ReorderGoal("beta", 1)
	require.NoError(t, err)

	goals, err = s.LoadGoalTree()
	require.NoError(t, err)
	assert.Equal(t, "alpha", goals[0].Slug)
	assert.Equal(t, "beta", goals[1].Slug)
	assert.Equal(t, "gamma", goals[2].Slug)

	// Moving alpha up should be a no-op (already at top)
	err = s.ReorderGoal("alpha", -1)
	require.NoError(t, err)

	goals, err = s.LoadGoalTree()
	require.NoError(t, err)
	assert.Equal(t, "alpha", goals[0].Slug)
}

func TestReorderSubGoal(t *testing.T) {
	s := setupTestStore(t)

	_, err := s.CreateGoal("", "parent")
	require.NoError(t, err)
	_, err = s.CreateGoal("parent", "aaa")
	require.NoError(t, err)
	_, err = s.CreateGoal("parent", "bbb")
	require.NoError(t, err)
	_, err = s.CreateGoal("parent", "ccc")
	require.NoError(t, err)

	// Move ccc up
	err = s.ReorderGoal(filepath.Join("parent", "ccc"), -1)
	require.NoError(t, err)

	goals, err := s.LoadGoalTree()
	require.NoError(t, err)
	require.Len(t, goals, 1)
	require.Len(t, goals[0].Children, 3)
	assert.Equal(t, "aaa", goals[0].Children[0].Slug)
	assert.Equal(t, "ccc", goals[0].Children[1].Slug)
	assert.Equal(t, "bbb", goals[0].Children[2].Slug)
}

func TestMoveGoalUnparent(t *testing.T) {
	s := setupTestStore(t)

	_, err := s.CreateGoal("", "parent")
	require.NoError(t, err)
	_, err = s.CreateGoal("parent", "child")
	require.NoError(t, err)

	// Move child to top level (unparent)
	err = s.MoveGoal(filepath.Join("parent", "child"), "")
	require.NoError(t, err)

	// Verify child is now top-level
	goals, err := s.LoadGoalTree()
	require.NoError(t, err)
	slugs := make([]string, len(goals))
	for i, g := range goals {
		slugs[i] = g.Slug
	}
	assert.Contains(t, slugs, "child")
	assert.Contains(t, slugs, "parent")

	// Verify parent has no children
	for _, g := range goals {
		if g.Slug == "parent" {
			assert.Empty(t, g.Children)
		}
	}

	// Verify child directory exists at new location
	_, err = os.Stat(filepath.Join(s.GoalsDir(), "child", "goal.md"))
	assert.NoError(t, err)

	// Verify old location is gone
	_, err = os.Stat(filepath.Join(s.GoalsDir(), "parent", "child"))
	assert.True(t, os.IsNotExist(err))
}

func TestMoveGoalReparent(t *testing.T) {
	s := setupTestStore(t)

	_, err := s.CreateGoal("", "alpha")
	require.NoError(t, err)
	_, err = s.CreateGoal("", "beta")
	require.NoError(t, err)

	// Move beta under alpha
	err = s.MoveGoal("beta", "alpha")
	require.NoError(t, err)

	goals, err := s.LoadGoalTree()
	require.NoError(t, err)

	// Only alpha should be top-level now
	topSlugs := make([]string, len(goals))
	for i, g := range goals {
		topSlugs[i] = g.Slug
	}
	assert.Contains(t, topSlugs, "alpha")
	assert.NotContains(t, topSlugs, "beta")

	// alpha should have beta as a child
	var alpha *Goal
	for _, g := range goals {
		if g.Slug == "alpha" {
			alpha = g
		}
	}
	require.NotNil(t, alpha)
	require.Len(t, alpha.Children, 1)
	assert.Equal(t, "beta", alpha.Children[0].Slug)
}

func TestMoveGoalIntoSelfFails(t *testing.T) {
	s := setupTestStore(t)

	_, err := s.CreateGoal("", "parent")
	require.NoError(t, err)
	_, err = s.CreateGoal("parent", "child")
	require.NoError(t, err)

	// Moving parent into its own child should fail
	err = s.MoveGoal("parent", filepath.Join("parent", "child"))
	assert.Error(t, err)
}

func TestChildrenOrderRoundTrip(t *testing.T) {
	s := setupTestStore(t)

	_, err := s.CreateGoal("", "project")
	require.NoError(t, err)
	_, err = s.CreateGoal("project", "aaa")
	require.NoError(t, err)
	_, err = s.CreateGoal("project", "bbb")
	require.NoError(t, err)
	_, err = s.CreateGoal("project", "ccc")
	require.NoError(t, err)

	// Set custom children order
	goal, err := s.LoadGoal("project")
	require.NoError(t, err)
	goal.ChildrenOrder = []string{"ccc", "aaa", "bbb"}
	err = s.SaveGoal(goal)
	require.NoError(t, err)

	// Reload and verify order is respected
	goals, err := s.LoadGoalTree()
	require.NoError(t, err)
	require.Len(t, goals, 1)
	require.Len(t, goals[0].Children, 3)
	assert.Equal(t, "ccc", goals[0].Children[0].Slug)
	assert.Equal(t, "aaa", goals[0].Children[1].Slug)
	assert.Equal(t, "bbb", goals[0].Children[2].Slug)
}

func TestGoalsByHorizon(t *testing.T) {
	s := setupTestStore(t)

	_, err := s.CreateGoal("", "urgent")
	require.NoError(t, err)
	_, err = s.SetHorizon("urgent", HorizonToday)
	require.NoError(t, err)

	_, err = s.CreateGoal("", "soon")
	require.NoError(t, err)
	_, err = s.SetHorizon("soon", HorizonTomorrow)
	require.NoError(t, err)

	_, err = s.CreateGoal("", "later")
	require.NoError(t, err)
	// default horizon is future

	today, tomorrow, future, err := s.GoalsByHorizon()
	require.NoError(t, err)
	assert.Len(t, today, 1)
	assert.Len(t, tomorrow, 1)
	assert.Len(t, future, 1)
}
