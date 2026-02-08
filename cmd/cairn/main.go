package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stefanpenner/cairn/pkg/store"
	gsync "github.com/stefanpenner/cairn/pkg/sync"
	"github.com/stefanpenner/cairn/pkg/tui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	dataDir := getDataDir()
	s, err := store.NewStore(dataDir)
	if err != nil {
		return err
	}

	args := os.Args[1:]
	jsonOutput := hasFlag(args, "--json")
	args = removeFlag(args, "--json")

	if len(args) == 0 {
		return runTUI(s)
	}

	switch args[0] {
	case "queue":
		return cmdQueue(s, jsonOutput)
	case "list":
		return cmdList(s, jsonOutput)
	case "status":
		if len(args) < 2 {
			return fmt.Errorf("usage: cairn status <goal-path>")
		}
		return cmdStatus(s, args[1], jsonOutput)
	case "complete":
		if len(args) < 2 {
			return fmt.Errorf("usage: cairn complete <goal-path>")
		}
		return cmdSetStatus(s, args[1], store.StatusComplete, jsonOutput)
	case "incomplete":
		if len(args) < 2 {
			return fmt.Errorf("usage: cairn incomplete <goal-path>")
		}
		return cmdSetStatus(s, args[1], store.StatusIncomplete, jsonOutput)
	case "add":
		if len(args) < 2 {
			return fmt.Errorf("usage: cairn add [parent] <slug>")
		}
		parent := ""
		slug := args[1]
		if len(args) >= 3 {
			parent = args[1]
			slug = args[2]
		}
		return cmdAdd(s, parent, slug, jsonOutput)
	case "note":
		if len(args) < 3 {
			return fmt.Errorf("usage: cairn note <goal-path> <text>")
		}
		text := strings.Join(args[2:], " ")
		return cmdNote(s, args[1], text, jsonOutput)
	case "delete":
		if len(args) < 2 {
			return fmt.Errorf("usage: cairn delete <goal-path>")
		}
		return cmdDelete(s, args[1], jsonOutput)
	case "init":
		remote := ""
		for i, a := range args {
			if a == "--remote" && i+1 < len(args) {
				remote = args[i+1]
			}
		}
		return gsync.InitRepo(dataDir, remote)
	case "sync":
		return gsync.SyncRepo(dataDir)
	case "horizon":
		if len(args) < 3 {
			return fmt.Errorf("usage: cairn horizon <goal-path> <today|tomorrow|future>")
		}
		return cmdHorizon(s, args[1], args[2], jsonOutput)
	case "search":
		if len(args) < 2 {
			return fmt.Errorf("usage: cairn search <query>")
		}
		return cmdSearch(s, strings.Join(args[1:], " "), jsonOutput)
	default:
		return fmt.Errorf("unknown command: %s\nUsage: cairn [queue|list|status|complete|incomplete|add|note|delete|init|sync|horizon|search]", args[0])
	}
}

func getDataDir() string {
	// Check env var
	if dir := os.Getenv("CAIRN_DIR"); dir != "" {
		return dir
	}
	// Check --dir flag
	for i, a := range os.Args {
		if a == "--dir" && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
	}
	// Default
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cairn")
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

func removeFlag(args []string, flag string) []string {
	var result []string
	for _, a := range args {
		if a != flag {
			result = append(result, a)
		}
	}
	return result
}

func runTUI(s *store.Store) error {
	m := tui.NewModel(s)
	p := tea.NewProgram(m, tea.WithAltScreen())

	// Start file watcher
	cleanup, err := tui.StartWatcher(s.Root, p)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: file watcher failed: %v\n", err)
	} else {
		defer cleanup()
	}

	_, err = p.Run()
	return err
}

// CLI Commands

func cmdQueue(s *store.Store, jsonOut bool) error {
	q, err := s.LoadQueue()
	if err != nil {
		return err
	}

	if jsonOut {
		return outputJSON(q)
	}

	if len(q.Items) == 0 {
		fmt.Println("Queue is empty. Edit ~/.cairn/queue.md to add items.")
		return nil
	}

	for i, item := range q.Items {
		// Try to load goal to get status
		g, err := s.LoadGoal(item)
		status := "?"
		if err == nil {
			if g.IsComplete() {
				status = "✓"
			} else {
				status = "○"
			}
		}
		fmt.Printf("%d. %s %s\n", i+1, status, item)
	}
	return nil
}

func cmdList(s *store.Store, jsonOut bool) error {
	goals, err := s.LoadGoalTree()
	if err != nil {
		return err
	}

	if jsonOut {
		return outputJSON(goalsToMap(goals))
	}

	printGoalTree(goals, 0)
	return nil
}

func printGoalTree(goals []*store.Goal, depth int) {
	for _, g := range goals {
		indent := strings.Repeat("  ", depth)
		status := "○"
		if g.IsComplete() {
			status = "✓"
		}
		horizon := ""
		if g.Horizon == store.HorizonToday {
			horizon = " [today]"
		} else if g.Horizon == store.HorizonTomorrow {
			horizon = " [tomorrow]"
		}
		fmt.Printf("%s%s %s%s\n", indent, status, g.Title, horizon)
		printGoalTree(g.Children, depth+1)
	}
}

func cmdStatus(s *store.Store, goalPath string, jsonOut bool) error {
	g, err := s.LoadGoal(goalPath)
	if err != nil {
		return err
	}

	if jsonOut {
		return outputJSON(goalToMap(g))
	}

	status := "incomplete"
	if g.IsComplete() {
		status = "complete"
	}
	fmt.Printf("%s: %s\n", g.Title, status)
	if g.Horizon != "" {
		fmt.Printf("Horizon: %s\n", g.Horizon)
	}
	if len(g.Tags) > 0 {
		fmt.Printf("Tags: %s\n", strings.Join(g.Tags, ", "))
	}
	if g.Body != "" {
		fmt.Println()
		fmt.Println(g.Body)
	}
	return nil
}

func cmdSetStatus(s *store.Store, goalPath string, status store.GoalStatus, jsonOut bool) error {
	g, err := s.LoadGoal(goalPath)
	if err != nil {
		return err
	}

	g.Status = status
	if err := s.SaveGoal(g); err != nil {
		return err
	}

	if jsonOut {
		return outputJSON(goalToMap(g))
	}

	fmt.Printf("%s → %s\n", g.Title, status)
	return nil
}

func cmdAdd(s *store.Store, parent, slug string, jsonOut bool) error {
	g, err := s.CreateGoal(parent, slug)
	if err != nil {
		return err
	}

	if jsonOut {
		return outputJSON(goalToMap(g))
	}

	fmt.Printf("Created: %s\n", g.Path)
	return nil
}

func cmdNote(s *store.Store, goalPath, text string, jsonOut bool) error {
	g, err := s.AddNote(goalPath, text)
	if err != nil {
		return err
	}

	if jsonOut {
		return outputJSON(goalToMap(g))
	}

	fmt.Printf("Note added to %s\n", g.Title)
	return nil
}

func cmdDelete(s *store.Store, goalPath string, jsonOut bool) error {
	if err := s.DeleteGoal(goalPath); err != nil {
		return err
	}

	if jsonOut {
		return outputJSON(map[string]string{"deleted": goalPath})
	}

	fmt.Printf("Deleted: %s\n", goalPath)
	return nil
}

func cmdHorizon(s *store.Store, goalPath, horizon string, jsonOut bool) error {
	var h store.Horizon
	switch horizon {
	case "today":
		h = store.HorizonToday
	case "tomorrow":
		h = store.HorizonTomorrow
	case "future":
		h = store.HorizonFuture
	default:
		return fmt.Errorf("invalid horizon: %s (use today, tomorrow, or future)", horizon)
	}

	g, err := s.SetHorizon(goalPath, h)
	if err != nil {
		return err
	}

	if jsonOut {
		return outputJSON(goalToMap(g))
	}

	fmt.Printf("%s → %s\n", g.Title, horizon)
	return nil
}

func cmdSearch(s *store.Store, query string, jsonOut bool) error {
	matches, err := s.SearchNotes(query)
	if err != nil {
		return err
	}

	if jsonOut {
		return outputJSON(goalsToMap(matches))
	}

	if len(matches) == 0 {
		fmt.Println("No matches found.")
		return nil
	}

	for _, g := range matches {
		fmt.Printf("%s (%s)\n", g.Title, g.Path)
	}
	return nil
}

// JSON helpers

func outputJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func goalToMap(g *store.Goal) map[string]interface{} {
	m := map[string]interface{}{
		"title":   g.Title,
		"status":  string(g.Status),
		"path":    g.Path,
		"horizon": string(g.Horizon),
		"tags":    g.Tags,
		"links":   g.Links,
		"body":    g.Body,
	}
	if !g.Created.IsZero() {
		m["created"] = g.Created.Format("2006-01-02T15:04:05Z")
	}
	if !g.Updated.IsZero() {
		m["updated"] = g.Updated.Format("2006-01-02T15:04:05Z")
	}
	return m
}

func goalsToMap(goals []*store.Goal) []map[string]interface{} {
	var result []map[string]interface{}
	for _, g := range goals {
		m := goalToMap(g)
		if len(g.Children) > 0 {
			m["children"] = goalsToMap(g.Children)
		}
		result = append(result, m)
	}
	return result
}
