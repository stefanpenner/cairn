package tui

import (
	"github.com/stefanpenner/cairn/pkg/store"
)

// TreeItem represents a flattened view of a goal in the tree.
type TreeItem struct {
	ID              string // unique path-based ID (e.g., "otr/ios")
	ParentID        string // parent's ID for search ancestor tracking
	Name            string
	Goal            *store.Goal
	Depth           int
	HasChildren     bool
	IsExpanded      bool
	IsSectionHeader bool // true for "TODAY", "TOMORROW", "FUTURE" headers
}

// BuildTreeItems converts a slice of Goals into TreeItems for TUI rendering.
func BuildTreeItems(goals []*store.Goal, expandedState map[string]bool) []*TreeItem {
	var items []*TreeItem
	for _, g := range goals {
		item := buildItem(g, 0, expandedState)
		items = append(items, item)
	}
	return items
}

func buildItem(g *store.Goal, depth int, expandedState map[string]bool) *TreeItem {
	item := &TreeItem{
		ID:          g.Path,
		Name:        displayName(g),
		Goal:        g,
		Depth:       depth,
		HasChildren: len(g.Children) > 0,
		IsExpanded:  expandedState[g.Path],
	}

	return item
}

func displayName(g *store.Goal) string {
	if g.Title != "" {
		return g.Title
	}
	return g.Slug
}

// FlattenVisibleItems returns a flat list of visible items based on expanded state.
// When groupByHorizon is false, items are listed in tree order.
// When true, items are grouped under TODAY / TOMORROW / FUTURE section headers.
func FlattenVisibleItems(goals []*store.Goal, expandedState map[string]bool) []TreeItem {
	var result []TreeItem
	flattenGoals(goals, 0, "", expandedState, &result)
	return result
}

// FlattenWithHorizonGroups groups top-level goals by horizon with section headers.
func FlattenWithHorizonGroups(goals []*store.Goal, expandedState map[string]bool) []TreeItem {
	var today, tomorrow, future []*store.Goal
	for _, g := range goals {
		switch g.Horizon {
		case store.HorizonToday:
			today = append(today, g)
		case store.HorizonTomorrow:
			tomorrow = append(tomorrow, g)
		default:
			future = append(future, g)
		}
	}

	var result []TreeItem

	if len(today) > 0 {
		result = append(result, TreeItem{
			ID:              "__header_today",
			Name:            "TODAY",
			IsSectionHeader: true,
			Goal:            &store.Goal{},
		})
		flattenGoals(today, 1, "__header_today", expandedState, &result)
	}

	if len(tomorrow) > 0 {
		result = append(result, TreeItem{
			ID:              "__header_tomorrow",
			Name:            "TOMORROW",
			IsSectionHeader: true,
			Goal:            &store.Goal{},
		})
		flattenGoals(tomorrow, 1, "__header_tomorrow", expandedState, &result)
	}

	if len(future) > 0 {
		result = append(result, TreeItem{
			ID:              "__header_future",
			Name:            "FUTURE",
			IsSectionHeader: true,
			Goal:            &store.Goal{},
		})
		flattenGoals(future, 1, "__header_future", expandedState, &result)
	}

	return result
}

func flattenGoals(goals []*store.Goal, depth int, parentID string, expandedState map[string]bool, result *[]TreeItem) {
	for _, g := range goals {
		item := TreeItem{
			ID:          g.Path,
			ParentID:    parentID,
			Name:        displayName(g),
			Goal:        g,
			Depth:       depth,
			HasChildren: len(g.Children) > 0,
			IsExpanded:  expandedState[g.Path],
		}
		*result = append(*result, item)

		if item.HasChildren && item.IsExpanded {
			flattenGoals(g.Children, depth+1, g.Path, expandedState, result)
		}
	}
}

// FilterVisibleItems filters already-flattened visible items to only include
// items whose ID is in matchIDs or ancestorIDs.
func FilterVisibleItems(items []TreeItem, matchIDs, ancestorIDs map[string]bool) []TreeItem {
	var result []TreeItem
	for _, item := range items {
		if matchIDs[item.ID] || ancestorIDs[item.ID] {
			result = append(result, item)
		}
	}
	return result
}
