package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/stefanpenner/cairn/pkg/store"
)

// FileChangedMsg is sent when the file watcher detects changes.
type FileChangedMsg struct{}

// SyncDoneMsg is sent when git sync completes.
type SyncDoneMsg struct {
	Err error
}

// EditorFinishedMsg is sent when $EDITOR returns.
type EditorFinishedMsg struct {
	Err error
}

// Model is the Bubble Tea model for the productivity TUI.
type Model struct {
	store         *store.Store
	keys          KeyMap
	width         int
	height        int
	goals         []*store.Goal
	queue         *store.Queue
	visibleItems  []TreeItem
	expandedState map[string]bool
	cursor        int
	activeQueue   int
	focusedPane   int // 0 = tree, 1 = notes
	notesScroll   int

	// Modal state
	showHelpModal     bool
	showDeleteConfirm bool
	deleteTarget      string

	// Move mode
	isMoveMode bool
	moveTarget string // path of the goal being moved

	// Input mode (for adding goals)
	isInputMode      bool
	textInput        textinput.Model
	inputParent      string // parent path for new goal, "" for top-level
	inputDepth       int    // indentation depth for the input line in the tree
	inputInsertAfter int    // visible items index to insert input after

	// Rename mode
	isRenameMode   bool
	renameGoalPath string

	// Inline edit mode
	isEditing    bool
	noteEditor   textarea.Model
	editGoalPath string // path of the goal being edited

	// Search state
	isSearching    bool
	searchQuery    string
	searchMatchIDs map[string]bool // IDs of items matching query
	searchAncIDs   map[string]bool // IDs of ancestor items (for context)

	// Status message
	statusMsg     string
	statusTimeout time.Time

	// Cached glamour renderer (expensive to create)
	glamourRenderer *glamour.TermRenderer
	glamourWidth    int

	// Track whether all items are expanded for toggle
	allExpanded bool
}

// NewModel creates a new TUI model.
func NewModel(s *store.Store) Model {
	ti := textinput.New()
	ti.Placeholder = "goal-name"
	ti.CharLimit = 64

	m := Model{
		store:         s,
		keys:          DefaultKeyMap(),
		expandedState: make(map[string]bool),
		textInput:     ti,
	}
	return m
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.WindowSize()
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Pre-create glamour renderer at the right width
		rightWidth := msg.Width - (msg.Width / 4) - 1 - 2
		if rightWidth < 20 {
			rightWidth = 20
		}
		m.getGlamourRenderer(rightWidth)
		// Resize editor if active
		if m.isEditing {
			editorWidth := msg.Width - (msg.Width / 4) - 1
			if editorWidth < 20 {
				editorWidth = 20
			}
			m.noteEditor.SetWidth(editorWidth)
			contentHeight := msg.Height - 5
			editorHeight := contentHeight - 4 - 1 // header estimate + file path
			if editorHeight < 3 {
				editorHeight = 3
			}
			m.noteEditor.SetHeight(editorHeight)
		}
		m.reload()
		return m, tea.ClearScreen

	case FileChangedMsg:
		m.reload()
		return m, nil

	case SyncDoneMsg:
		if msg.Err != nil {
			m.setStatus("Sync failed: " + msg.Err.Error())
		} else {
			m.setStatus("Synced successfully")
			m.reload()
		}
		return m, nil

	case EditorFinishedMsg:
		m.reload()
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	// Update text input if in input mode
	if m.isInputMode {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	// Update textarea if in edit mode
	if m.isEditing {
		var cmd tea.Cmd
		m.noteEditor, cmd = m.noteEditor.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Input mode handling
	if m.isInputMode {
		switch msg.Type {
		case tea.KeyEsc:
			m.isInputMode = false
			return m, nil
		case tea.KeyEnter:
			name := strings.TrimSpace(m.textInput.Value())
			if name != "" {
				_, err := m.store.CreateGoal(m.inputParent, name)
				if err != nil {
					m.setStatus("Error: " + err.Error())
				} else {
					m.setStatus("Created: " + name)
					m.reload()
				}
			}
			m.isInputMode = false
			return m, nil
		default:
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}
	}

	// Rename mode handling
	if m.isRenameMode {
		switch msg.Type {
		case tea.KeyEsc:
			m.isRenameMode = false
			return m, nil
		case tea.KeyEnter:
			newTitle := strings.TrimSpace(m.textInput.Value())
			if newTitle != "" {
				goal, err := m.store.LoadGoal(m.renameGoalPath)
				if err != nil {
					m.setStatus("Error: " + err.Error())
				} else {
					goal.Title = newTitle
					if err := m.store.SaveGoal(goal); err != nil {
						m.setStatus("Error: " + err.Error())
					} else {
						m.setStatus("Renamed to: " + newTitle)
						m.reload()
					}
				}
			}
			m.isRenameMode = false
			return m, nil
		default:
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}
	}

	// Inline edit mode handling
	if m.isEditing {
		return m.handleEditMode(msg)
	}

	// Search input mode handling
	if m.isSearching {
		return m.handleSearchInput(msg)
	}

	// Help modal
	if m.showHelpModal {
		switch msg.String() {
		case "esc", "enter", "?", "q":
			m.showHelpModal = false
		}
		return m, nil
	}

	// Move mode handling
	if m.isMoveMode {
		return m.handleMoveMode(msg)
	}

	// Delete confirmation
	if m.showDeleteConfirm {
		switch msg.String() {
		case "y", "Y":
			if err := m.store.DeleteGoal(m.deleteTarget); err != nil {
				m.setStatus("Delete failed: " + err.Error())
			} else {
				m.setStatus("Deleted: " + m.deleteTarget)
				m.reload()
				if m.cursor >= len(m.visibleItems) && m.cursor > 0 {
					m.cursor--
				}
			}
			m.showDeleteConfirm = false
		case "n", "N", "esc":
			m.showDeleteConfirm = false
		}
		return m, nil
	}

	// If search filter is active (not typing), Esc/Enter clears it
	if m.searchQuery != "" && (msg.Type == tea.KeyEsc || msg.Type == tea.KeyEnter) {
		var curID string
		if m.cursor >= 0 && m.cursor < len(m.visibleItems) {
			curID = m.visibleItems[m.cursor].ID
		}
		m.searchQuery = ""
		m.searchMatchIDs = nil
		m.searchAncIDs = nil
		m.rebuildVisible()
		if curID != "" {
			for i, item := range m.visibleItems {
				if item.ID == curID {
					m.cursor = i
					break
				}
			}
		}
		return m, nil
	}

	// Normal mode
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Up):
		if m.focusedPane == 1 {
			// Scroll notes panel up
			if m.notesScroll > 0 {
				m.notesScroll--
			}
		} else {
			if m.cursor > 0 {
				m.cursor--
				// Skip section headers
				if m.cursor >= 0 && m.cursor < len(m.visibleItems) && m.visibleItems[m.cursor].IsSectionHeader {
					if m.cursor > 0 {
						m.cursor--
					} else {
						m.cursor++
					}
				}
			}
			m.notesScroll = 0
		}

	case key.Matches(msg, m.keys.Down):
		if m.focusedPane == 1 {
			// Scroll notes panel down
			m.notesScroll++
		} else {
			if m.cursor < len(m.visibleItems)-1 {
				m.cursor++
				// Skip section headers
				if m.cursor < len(m.visibleItems) && m.visibleItems[m.cursor].IsSectionHeader {
					if m.cursor < len(m.visibleItems)-1 {
						m.cursor++
					} else {
						m.cursor--
					}
				}
			}
			m.notesScroll = 0
		}

	case key.Matches(msg, m.keys.Right):
		if m.cursor < len(m.visibleItems) {
			item := m.visibleItems[m.cursor]
			if item.HasChildren {
				m.expandedState[item.ID] = true
				m.rebuildVisible()
			}
		}

	case key.Matches(msg, m.keys.Left):
		if m.cursor < len(m.visibleItems) {
			item := m.visibleItems[m.cursor]
			if item.IsExpanded {
				m.expandedState[item.ID] = false
				m.rebuildVisible()
			}
		}

	case key.Matches(msg, m.keys.Enter):
		if m.cursor < len(m.visibleItems) {
			item := m.visibleItems[m.cursor]
			if item.IsSectionHeader {
				// no-op on section headers
			} else if item.HasChildren {
				m.expandedState[item.ID] = !m.expandedState[item.ID]
				m.rebuildVisible()
			}
		}

	case key.Matches(msg, m.keys.Space):
		if m.cursor < len(m.visibleItems) {
			item := m.visibleItems[m.cursor]
			_, err := m.store.ToggleStatus(item.Goal.Path)
			if err != nil {
				m.setStatus("Error: " + err.Error())
			} else {
				m.reload()
			}
		}

	case key.Matches(msg, m.keys.Tab):
		m.focusedPane = (m.focusedPane + 1) % 2

	case key.Matches(msg, m.keys.NextQueue):
		if m.queue != nil && len(m.queue.Items) > 0 {
			m.activeQueue = (m.activeQueue + 1) % len(m.queue.Items)
			m.cursor = 0
			m.rebuildVisible()
		}

	case key.Matches(msg, m.keys.PrevQueue):
		if m.queue != nil && len(m.queue.Items) > 0 {
			m.activeQueue = (m.activeQueue - 1 + len(m.queue.Items)) % len(m.queue.Items)
			m.cursor = 0
			m.rebuildVisible()
		}

	case key.Matches(msg, m.keys.InlineEdit):
		if m.cursor < len(m.visibleItems) {
			item := m.visibleItems[m.cursor]
			if item.IsSectionHeader {
				break
			}
			m.enterEditMode(item.Goal)
			return m, textarea.Blink
		}

	case key.Matches(msg, m.keys.ExternalEdit):
		if m.cursor < len(m.visibleItems) {
			item := m.visibleItems[m.cursor]
			if item.IsSectionHeader {
				break
			}
			return m, m.openEditor(item.Goal)
		}

	case key.Matches(msg, m.keys.AddTop):
		m.isInputMode = true
		m.textInput.Reset()
		m.textInput.Focus()
		m.inputParent = ""
		m.inputDepth = 0
		m.inputInsertAfter = len(m.visibleItems) - 1
		m.textInput.Placeholder = "top-level goal name"
		return m, textinput.Blink

	case key.Matches(msg, m.keys.Add):
		m.isInputMode = true
		m.textInput.Reset()
		m.textInput.Focus()
		if m.cursor < len(m.visibleItems) {
			parent := m.visibleItems[m.cursor]
			m.inputParent = parent.Goal.Path
			m.inputDepth = parent.Depth + 1

			// Expand parent so children are visible
			if parent.HasChildren && !parent.IsExpanded {
				m.expandedState[parent.ID] = true
				m.rebuildVisible()
			}

			// Find last visible descendant of parent to place input after
			m.inputInsertAfter = m.cursor
			for j := m.cursor + 1; j < len(m.visibleItems); j++ {
				if m.visibleItems[j].Depth <= parent.Depth {
					break
				}
				m.inputInsertAfter = j
			}

			m.textInput.Placeholder = "sub-goal name under " + parent.Name
		} else {
			m.inputParent = ""
			m.inputDepth = 0
			m.inputInsertAfter = len(m.visibleItems) - 1
			m.textInput.Placeholder = "top-level goal name"
		}
		return m, textinput.Blink

	case key.Matches(msg, m.keys.Rename):
		if m.cursor < len(m.visibleItems) {
			item := m.visibleItems[m.cursor]
			if item.IsSectionHeader {
				break
			}
			m.isRenameMode = true
			m.renameGoalPath = item.Goal.Path
			m.textInput.Reset()
			m.textInput.SetValue(item.Name)
			m.textInput.Focus()
			m.textInput.Placeholder = "new title"
			return m, textinput.Blink
		}

	case key.Matches(msg, m.keys.Delete):
		if m.cursor < len(m.visibleItems) {
			m.deleteTarget = m.visibleItems[m.cursor].Goal.Path
			m.showDeleteConfirm = true
		}

	case key.Matches(msg, m.keys.ToggleExpand):
		if m.allExpanded {
			m.expandedState = make(map[string]bool)
			m.allExpanded = false
		} else {
			m.expandAll()
			m.allExpanded = true
		}
		m.rebuildVisible()

	case key.Matches(msg, m.keys.Reload):
		m.reload()
		m.setStatus("Reloaded")

	case key.Matches(msg, m.keys.Sync):
		return m, m.doSync()

	case key.Matches(msg, m.keys.Move):
		if m.cursor < len(m.visibleItems) {
			m.isMoveMode = true
			m.moveTarget = m.visibleItems[m.cursor].Goal.Path
			m.setStatus("Move mode: j/k reorder, h unparent, l reparent, enter/esc exit")
		}

	case key.Matches(msg, m.keys.Search):
		m.isSearching = true
		m.searchQuery = ""
		m.searchMatchIDs = nil
		m.searchAncIDs = nil

	case key.Matches(msg, m.keys.Help):
		m.showHelpModal = !m.showHelpModal

	case key.Matches(msg, m.keys.Today):
		if m.cursor < len(m.visibleItems) {
			item := m.visibleItems[m.cursor]
			_, err := m.store.SetHorizon(item.Goal.Path, store.HorizonToday)
			if err != nil {
				m.setStatus("Error: " + err.Error())
			} else {
				m.setStatus(item.Name + " → today")
				m.reload()
			}
		}

	case key.Matches(msg, m.keys.Tomorrow):
		if m.cursor < len(m.visibleItems) {
			item := m.visibleItems[m.cursor]
			_, err := m.store.SetHorizon(item.Goal.Path, store.HorizonTomorrow)
			if err != nil {
				m.setStatus("Error: " + err.Error())
			} else {
				m.setStatus(item.Name + " → tomorrow")
				m.reload()
			}
		}

	case key.Matches(msg, m.keys.Future):
		if m.cursor < len(m.visibleItems) {
			item := m.visibleItems[m.cursor]
			_, err := m.store.SetHorizon(item.Goal.Path, store.HorizonFuture)
			if err != nil {
				m.setStatus("Error: " + err.Error())
			} else {
				m.setStatus(item.Name + " → future")
				m.reload()
			}
		}
	}

	return m, nil
}

// handleEditMode handles key messages while inline editing.
func (m Model) handleEditMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyEsc:
		// Save and exit
		m.saveInlineEdit()
		m.isEditing = false
		m.noteEditor.Blur()
		m.reload()
		m.setStatus("Saved")
		return m, nil

	case msg.Type == tea.KeyCtrlS:
		// Save but stay in edit mode
		m.saveInlineEdit()
		m.reload()
		m.setStatus("Saved")
		return m, nil

	case msg.Type == tea.KeyCtrlC:
		// Cancel without saving
		m.isEditing = false
		m.noteEditor.Blur()
		m.setStatus("Edit cancelled")
		return m, nil

	default:
		var cmd tea.Cmd
		m.noteEditor, cmd = m.noteEditor.Update(msg)
		return m, cmd
	}
}

// handleSearchInput handles key messages while typing in the search bar.
func (m Model) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		// Exit search and clear filter
		m.isSearching = false
		m.searchQuery = ""
		m.searchMatchIDs = nil
		m.searchAncIDs = nil
		m.rebuildVisible()
		return m, nil

	case tea.KeyEnter, tea.KeyDown, tea.KeyTab:
		// Exit search input but keep filter active
		m.isSearching = false
		return m, nil

	case tea.KeyBackspace:
		if len(m.searchQuery) > 0 {
			_, size := utf8.DecodeLastRuneInString(m.searchQuery)
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-size]
		}
		m.applySearchFilter()
		m.rebuildVisible()
		return m, nil

	default:
		if msg.Type == tea.KeyRunes {
			m.searchQuery += string(msg.Runes)
			m.applySearchFilter()
			m.rebuildVisible()
		}
		return m, nil
	}
}

// enterEditMode sets up the textarea for inline editing of a goal's notes.
func (m *Model) enterEditMode(goal *store.Goal) {
	ta := textarea.New()
	ta.ShowLineNumbers = false
	ta.SetValue(goal.Body)

	// Size the editor to the right panel, leaving room for header and file path
	rightWidth := m.width - (m.width / 4) - 1
	if rightWidth < 20 {
		rightWidth = 20
	}

	// Estimate header height (title + metadata + links + glamour spacing)
	headerLines := 3 // title line + blank + meta line (rough estimate)
	if len(goal.Links) > 0 {
		headerLines += len(goal.Links) + 1
	}

	contentHeight := m.height - 5 // outer chrome (header/tabs/seps/footer)
	editorHeight := contentHeight - headerLines - 1 // -1 for file path line
	if editorHeight < 3 {
		editorHeight = 3
	}
	ta.SetWidth(rightWidth)
	ta.SetHeight(editorHeight)
	ta.Focus()

	m.isEditing = true
	m.noteEditor = ta
	m.editGoalPath = goal.Path
	m.focusedPane = 1
}

// saveInlineEdit saves the textarea content back to the goal file.
func (m *Model) saveInlineEdit() {
	goal, err := m.store.LoadGoal(m.editGoalPath)
	if err != nil {
		m.setStatus("Save error: " + err.Error())
		return
	}
	goal.Body = m.noteEditor.Value()
	if err := m.store.SaveGoal(goal); err != nil {
		m.setStatus("Save error: " + err.Error())
	}
}

// applySearchFilter computes searchMatchIDs and searchAncIDs based on searchQuery.
func (m *Model) applySearchFilter() {
	if m.searchQuery == "" {
		m.searchMatchIDs = nil
		m.searchAncIDs = nil
		return
	}

	query := strings.ToLower(m.searchQuery)
	m.searchMatchIDs = make(map[string]bool)
	m.searchAncIDs = make(map[string]bool)

	// Walk all visible items looking for matches
	// We need to walk the full flattened tree (before filtering)
	var allItems []TreeItem
	allItems = FlattenWithHorizonGroups(m.goals, m.expandedState)
	// Also add items from non-grouped view if using queue
	if m.queue != nil && len(m.queue.Items) > 0 && m.activeQueue < len(m.queue.Items) {
		activeSlug := m.queue.Items[m.activeQueue]
		for _, g := range m.goals {
			if g.Slug == activeSlug {
				allItems = FlattenVisibleItems([]*store.Goal{g}, m.expandedState)
				break
			}
		}
	}

	for _, item := range allItems {
		if item.IsSectionHeader {
			continue
		}
		if strings.Contains(strings.ToLower(item.Name), query) {
			m.searchMatchIDs[item.ID] = true
			m.addSearchAncestors(item.ParentID, allItems)
		}
	}
}

// addSearchAncestors walks up the tree adding ancestor IDs and auto-expanding them.
func (m *Model) addSearchAncestors(parentID string, allItems []TreeItem) {
	if parentID == "" {
		return
	}
	if m.searchAncIDs[parentID] {
		return
	}
	m.searchAncIDs[parentID] = true
	m.expandedState[parentID] = true

	// Find the parent item and recurse up
	for _, item := range allItems {
		if item.ID == parentID {
			if item.ParentID != "" {
				m.addSearchAncestors(item.ParentID, allItems)
			}
			return
		}
	}
}

func (m Model) handleMoveMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		m.isMoveMode = false
		m.moveTarget = ""
		m.setStatus("Move cancelled")

	case msg.Type == tea.KeyEsc || msg.Type == tea.KeyEnter:
		m.isMoveMode = false
		m.moveTarget = ""
		m.setStatus("Move complete")

	case key.Matches(msg, m.keys.Down):
		// Try reorder down among siblings first
		moved := m.tryReorder(1)
		if !moved {
			// At bottom of siblings — shift to next horizon
			m.shiftHorizon(1)
		}

	case key.Matches(msg, m.keys.Up):
		// Try reorder up among siblings first
		moved := m.tryReorder(-1)
		if !moved {
			// At top of siblings — shift to previous horizon
			m.shiftHorizon(-1)
		}

	case key.Matches(msg, m.keys.Left):
		// Unparent: move to parent's parent (one level up)
		parentPath := filepath.Dir(m.moveTarget)
		if parentPath == "." {
			// Already top-level, nothing to do
			m.setStatus("Already at top level")
		} else {
			grandparentPath := filepath.Dir(parentPath)
			if grandparentPath == "." {
				grandparentPath = ""
			}
			if err := m.store.MoveGoal(m.moveTarget, grandparentPath); err != nil {
				m.setStatus("Move error: " + err.Error())
			} else {
				// Update moveTarget to reflect new path
				slug := filepath.Base(m.moveTarget)
				if grandparentPath == "" {
					m.moveTarget = slug
				} else {
					m.moveTarget = filepath.Join(grandparentPath, slug)
				}
				// Expand the new parent so we can see the moved item
				if grandparentPath != "" {
					m.expandedState[grandparentPath] = true
				}
				m.reload()
				m.moveCursorToGoal(m.moveTarget)
			}
		}

	case key.Matches(msg, m.keys.Right):
		// Reparent: move under the previous sibling
		slug := filepath.Base(m.moveTarget)

		// Find the previous sibling in the visible items
		prevSibling := m.findPreviousSibling(m.moveTarget)
		if prevSibling == "" {
			m.setStatus("No previous sibling to move under")
		} else {
			if err := m.store.MoveGoal(m.moveTarget, prevSibling); err != nil {
				m.setStatus("Move error: " + err.Error())
			} else {
				m.moveTarget = filepath.Join(prevSibling, slug)
				// Expand the new parent so we can see the moved item
				m.expandedState[prevSibling] = true
				m.reload()
				m.moveCursorToGoal(m.moveTarget)
			}
		}
	}

	return m, nil
}

// tryReorder attempts to reorder the move target among its siblings.
// Returns true if the goal actually moved, false if it was already at the boundary.
func (m *Model) tryReorder(delta int) bool {
	// Check if the goal is at the boundary before calling ReorderGoal
	goal := m.findGoalByPath(m.goals, m.moveTarget)
	if goal == nil {
		return false
	}

	// Find siblings
	parentPath := filepath.Dir(m.moveTarget)
	if parentPath == "." {
		parentPath = ""
	}
	var siblings []*store.Goal
	if parentPath == "" {
		siblings = m.goals
	} else {
		parent := m.findGoalByPath(m.goals, parentPath)
		if parent != nil {
			siblings = parent.Children
		}
	}

	// Find current index among siblings
	idx := -1
	for i, s := range siblings {
		if s.Path == m.moveTarget {
			idx = i
			break
		}
	}
	if idx == -1 {
		return false
	}

	newIdx := idx + delta
	if newIdx < 0 || newIdx >= len(siblings) {
		return false
	}

	if err := m.store.ReorderGoal(m.moveTarget, delta); err != nil {
		m.setStatus("Move error: " + err.Error())
		return false
	}
	m.reload()
	m.moveCursorToGoal(m.moveTarget)
	return true
}

var horizonOrder = []store.Horizon{store.HorizonToday, store.HorizonTomorrow, store.HorizonFuture}

// shiftHorizon changes the move target's horizon to the next/previous one.
func (m *Model) shiftHorizon(delta int) {
	goal := m.findGoalByPath(m.goals, m.moveTarget)
	if goal == nil {
		return
	}

	// Only shift horizon for top-level goals
	parentPath := filepath.Dir(m.moveTarget)
	if parentPath != "." {
		return
	}

	currentIdx := 0
	for i, h := range horizonOrder {
		if h == goal.Horizon {
			currentIdx = i
			break
		}
	}

	newIdx := currentIdx + delta
	if newIdx < 0 || newIdx >= len(horizonOrder) {
		return
	}

	newHorizon := horizonOrder[newIdx]
	_, err := m.store.SetHorizon(m.moveTarget, newHorizon)
	if err != nil {
		m.setStatus("Move error: " + err.Error())
		return
	}

	m.setStatus(filepath.Base(m.moveTarget) + " → " + string(newHorizon))
	m.reload()
	m.moveCursorToGoal(m.moveTarget)
}

// moveCursorToGoal positions the cursor on the given goal path in the visible items.
func (m *Model) moveCursorToGoal(goalPath string) {
	for i, item := range m.visibleItems {
		if item.Goal.Path == goalPath {
			m.cursor = i
			return
		}
	}
}

// findPreviousSibling returns the path of the previous sibling of the goal.
// It looks at the goal tree data (not just visible items) to find the actual sibling.
func (m *Model) findPreviousSibling(goalPath string) string {
	slug := filepath.Base(goalPath)
	parentPath := filepath.Dir(goalPath)
	if parentPath == "." {
		parentPath = ""
	}

	// Find siblings from the goal tree
	var siblings []*store.Goal
	if parentPath == "" {
		siblings = m.goals
	} else {
		parent := m.findGoalByPath(m.goals, parentPath)
		if parent == nil {
			return ""
		}
		siblings = parent.Children
	}

	for i, sib := range siblings {
		if sib.Slug == slug && i > 0 {
			return siblings[i-1].Path
		}
	}
	return ""
}

// findGoalByPath recursively searches for a goal by its path.
func (m *Model) findGoalByPath(goals []*store.Goal, path string) *store.Goal {
	for _, g := range goals {
		if g.Path == path {
			return g
		}
		if found := m.findGoalByPath(g.Children, path); found != nil {
			return found
		}
	}
	return nil
}

func (m *Model) reload() {
	goals, err := m.store.LoadGoalTree()
	if err != nil {
		m.setStatus("Load error: " + err.Error())
		return
	}
	m.goals = goals

	q, err := m.store.LoadQueue()
	if err != nil {
		q = &store.Queue{}
	}
	m.queue = q

	m.rebuildVisible()
}

func (m *Model) rebuildVisible() {
	// If we have a queue and an active queue item, show that goal's tree
	var goalsToShow []*store.Goal
	useHorizonGroups := false
	if m.queue != nil && len(m.queue.Items) > 0 && m.activeQueue < len(m.queue.Items) {
		activeSlug := m.queue.Items[m.activeQueue]
		for _, g := range m.goals {
			if g.Slug == activeSlug {
				goalsToShow = []*store.Goal{g}
				break
			}
		}
	}

	// If no queue match, show all goals grouped by horizon
	if len(goalsToShow) == 0 {
		goalsToShow = m.goals
		useHorizonGroups = true
	}

	if useHorizonGroups {
		m.visibleItems = FlattenWithHorizonGroups(goalsToShow, m.expandedState)
	} else {
		m.visibleItems = FlattenVisibleItems(goalsToShow, m.expandedState)
	}

	// Apply search filter if active
	if m.searchQuery != "" && (m.searchMatchIDs != nil || m.searchAncIDs != nil) {
		m.visibleItems = FilterVisibleItems(m.visibleItems, m.searchMatchIDs, m.searchAncIDs)
	}

	// Clamp cursor
	if m.cursor >= len(m.visibleItems) {
		m.cursor = len(m.visibleItems) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}

	// Skip section headers
	if m.cursor < len(m.visibleItems) && m.visibleItems[m.cursor].IsSectionHeader {
		for i := m.cursor; i < len(m.visibleItems); i++ {
			if !m.visibleItems[i].IsSectionHeader {
				m.cursor = i
				return
			}
		}
	}
}

func (m *Model) expandAll() {
	var expand func(goals []*store.Goal)
	expand = func(goals []*store.Goal) {
		for _, g := range goals {
			if len(g.Children) > 0 {
				m.expandedState[g.Path] = true
				expand(g.Children)
			}
		}
	}
	expand(m.goals)
	m.rebuildVisible()
}

// getGlamourRenderer returns a cached glamour renderer, creating one if needed
// or if the width changed.
func (m *Model) getGlamourRenderer(width int) *glamour.TermRenderer {
	if m.glamourRenderer != nil && m.glamourWidth == width {
		return m.glamourRenderer
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return nil
	}
	m.glamourRenderer = r
	m.glamourWidth = width
	return r
}

func (m *Model) setStatus(msg string) {
	m.statusMsg = msg
	m.statusTimeout = time.Now().Add(3 * time.Second)
}

func (m *Model) openEditor(g *store.Goal) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	filePath := g.FilePath
	if filePath == "" {
		// Ensure file exists
		if err := m.store.SaveGoal(g); err != nil {
			m.setStatus("Error saving: " + err.Error())
			return nil
		}
		filePath = g.FilePath
	}

	c := exec.Command(editor, filePath)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return EditorFinishedMsg{Err: err}
	})
}

func (m Model) doSync() tea.Cmd {
	return func() tea.Msg {
		dir := m.store.Root
		cmds := []struct {
			name string
			args []string
		}{
			{"git", []string{"-C", dir, "add", "-A"}},
			{"git", []string{"-C", dir, "commit", "-m", "sync " + time.Now().Format("2006-01-02 15:04:05")}},
			{"git", []string{"-C", dir, "pull", "--rebase"}},
			{"git", []string{"-C", dir, "push"}},
		}

		for _, c := range cmds {
			cmd := exec.Command(c.name, c.args...)
			if err := cmd.Run(); err != nil {
				// commit fails if nothing to commit — that's ok
				if c.args[2] != "commit" {
					return SyncDoneMsg{Err: err}
				}
			}
		}
		return SyncDoneMsg{}
	}
}
