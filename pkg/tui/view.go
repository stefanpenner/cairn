package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/stefanpenner/cairn/pkg/store"
)

const minWidth = 40
const minHeight = 10

// View implements tea.Model.
func (m Model) View() string {
	w := m.width
	h := m.height
	if w < minWidth {
		w = minWidth
	}
	if h < minHeight {
		h = minHeight
	}

	if m.showHelpModal {
		modal := m.renderHelpModal()
		return placeOverlay(modal, w, h)
	}

	if m.showDeleteConfirm {
		modal := m.renderDeleteModal()
		return placeOverlay(modal, w, h)
	}

	var b strings.Builder

	// Header
	header := m.renderHeader(w)
	b.WriteString(header)
	b.WriteString("\n")

	// Queue tabs
	tabs := m.renderQueueTabs(w)
	b.WriteString(tabs)
	b.WriteString("\n")

	// Separator
	b.WriteString(strings.Repeat("─", w))
	b.WriteString("\n")

	headerLines := 3
	footerLines := 2

	// Search bar takes a line if active
	searchActive := m.isSearching || m.searchQuery != ""
	if searchActive {
		headerLines++
	}

	contentHeight := h - headerLines - footerLines

	// Search bar
	if searchActive {
		b.WriteString(m.renderSearchBar(w))
		b.WriteString("\n")
	}

	// Two-panel layout — thin divider (just │, no padding spaces)
	leftWidth := w / 4
	rightWidth := w - leftWidth - 1 // 1 char for divider
	if leftWidth < 20 {
		leftWidth = 20
	}
	if rightWidth < 20 {
		rightWidth = 20
	}

	leftPanel := m.renderTreePanel(leftWidth, contentHeight)
	rightPanel := m.renderNotesPanel(rightWidth, contentHeight)

	// Join panels side by side with thin divider
	sepColor := ColorGrayDim
	if m.focusedPane == 1 || m.isEditing {
		sepColor = ColorPurple
	}
	sep := lipgloss.NewStyle().Foreground(sepColor).Render("│")
	for i := 0; i < contentHeight; i++ {
		leftLine := getLine(leftPanel, i, leftWidth)
		rightLine := getLine(rightPanel, i, rightWidth)
		b.WriteString(leftLine)
		b.WriteString(sep)
		b.WriteString(rightLine)
		b.WriteString("\n")
	}

	// Separator
	b.WriteString(strings.Repeat("─", w))
	b.WriteString("\n")

	// Footer
	footer := m.renderFooter(w)
	b.WriteString(footer)

	return b.String()
}

func (m Model) renderHeader(width int) string {
	title := HeaderStyle.Render("Productivity")

	// Stats
	totalGoals := countGoals(m.goals)
	completeGoals := countComplete(m.goals)
	stats := HeaderCountStyle.Render(fmt.Sprintf("%d/%d goals complete", completeGoals, totalGoals))

	// Status message
	status := ""
	if m.statusMsg != "" && time.Now().Before(m.statusTimeout) {
		status = "  " + lipgloss.NewStyle().Foreground(ColorCyan).Render(m.statusMsg)
	}

	gap := width - lipgloss.Width(title) - lipgloss.Width(stats) - lipgloss.Width(status)
	if gap < 1 {
		gap = 1
	}

	return title + strings.Repeat(" ", gap) + status + stats
}

func (m Model) renderQueueTabs(width int) string {
	if m.queue == nil || len(m.queue.Items) == 0 {
		return FooterStyle.Render("Queue: (empty — add goals to queue.md)")
	}

	var tabs []string
	tabs = append(tabs, FooterStyle.Render("Queue: "))
	for i, item := range m.queue.Items {
		if i == m.activeQueue {
			tabs = append(tabs, ActiveTabStyle.Render(item))
		} else {
			tabs = append(tabs, InactiveTabStyle.Render(item))
		}
	}
	return strings.Join(tabs, "")
}

func (m Model) renderSearchBar(width int) string {
	prefix := SearchBarStyle.Render(" / ")
	query := SearchBarStyle.Render(m.searchQuery)
	cursor := ""
	if m.isSearching {
		cursor = SearchBarStyle.Render("█")
	}

	matchCount := len(m.searchMatchIDs)
	countStr := ""
	if m.searchQuery != "" {
		countStr = SearchCountStyle.Render(fmt.Sprintf(" %d matches", matchCount))
	}

	left := prefix + query + cursor
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(countStr)
	padWidth := width - leftWidth - rightWidth
	if padWidth < 1 {
		padWidth = 1
	}

	return left + strings.Repeat(" ", padWidth) + countStr
}

func (m Model) renderTreePanel(width, height int) string {
	var lines []string

	// Reserve last line for directory path
	treeHeight := height - 1
	if treeHeight < 1 {
		treeHeight = 1
	}

	if len(m.visibleItems) == 0 {
		lines = append(lines, FooterStyle.Render("No goals yet. Press 'a' to add one."))
	}

	// Scrolling window
	startIdx := 0
	endIdx := len(m.visibleItems)
	if len(m.visibleItems) > treeHeight {
		half := treeHeight / 2
		startIdx = m.cursor - half
		if startIdx < 0 {
			startIdx = 0
		}
		endIdx = startIdx + treeHeight
		if endIdx > len(m.visibleItems) {
			endIdx = len(m.visibleItems)
			startIdx = endIdx - treeHeight
			if startIdx < 0 {
				startIdx = 0
			}
		}
	}

	for i := startIdx; i < endIdx; i++ {
		item := m.visibleItems[i]

		if item.IsSectionHeader {
			lines = append(lines, m.renderSectionHeader(item, width))
			continue
		}

		isSelected := i == m.cursor

		// Show inline rename input for the target item
		if m.isRenameMode && item.Goal.Path == m.renameGoalPath {
			indent := strings.Repeat(DepthIndent, item.Depth)
			prompt := InputPromptStyle.Render("✎ ")
			lines = append(lines, indent+prompt+m.textInput.View())
			continue
		}

		line := m.renderTreeItem(item, isSelected, width)
		lines = append(lines, line)

		// Insert input line at the correct position
		if m.isInputMode && i == m.inputInsertAfter {
			indent := strings.Repeat(DepthIndent, m.inputDepth)
			prompt := InputPromptStyle.Render("> ")
			lines = append(lines, indent+prompt+m.textInput.View())
		}
	}

	// Fallback for input when insert point is out of range or no items
	if m.isInputMode && (len(m.visibleItems) == 0 || m.inputInsertAfter < startIdx || m.inputInsertAfter >= endIdx) {
		indent := strings.Repeat(DepthIndent, m.inputDepth)
		prompt := InputPromptStyle.Render("> ")
		lines = append(lines, indent+prompt+m.textInput.View())
	}

	// Pad to treeHeight so the path line lands at the bottom
	for len(lines) < treeHeight {
		lines = append(lines, "")
	}

	// Directory path at bottom
	dirPath := m.store.GoalsDir()
	pathLine := lipgloss.NewStyle().Foreground(ColorGrayDim).Render(fileHyperlink(dirPath))
	lines = append(lines, pathLine)

	return strings.Join(lines, "\n")
}

func (m Model) renderSectionHeader(item TreeItem, width int) string {
	var style lipgloss.Style
	switch item.Name {
	case "TODAY":
		style = HorizonTodayStyle
	case "TOMORROW":
		style = HorizonTomorrowStyle
	default:
		style = HorizonFutureStyle
	}

	label := style.Bold(true).Render("── " + item.Name + " ")
	labelWidth := lipgloss.Width(label)
	remaining := width - labelWidth
	if remaining > 0 {
		label += lipgloss.NewStyle().Foreground(ColorGrayDim).Render(strings.Repeat("─", remaining))
	}
	return label
}

func (m Model) renderTreeItem(item TreeItem, isSelected bool, width int) string {
	indent := strings.Repeat(DepthIndent, item.Depth)

	// Expand/collapse icon
	var expandIcon string
	if item.HasChildren {
		if item.IsExpanded {
			expandIcon = IconExpanded + " "
		} else {
			expandIcon = IconCollapsed + " "
		}
	} else {
		expandIcon = "  "
	}

	// Status icon
	var statusIcon string
	if item.Goal.IsComplete() {
		statusIcon = CompleteStyle.Render(IconComplete)
	} else if item.Goal.IsInProgress() {
		statusIcon = InProgressStyle.Render(IconInProgress)
	} else {
		statusIcon = IncompleteStyle.Render(IconIncomplete)
	}

	// Move mode indicator
	movePrefix := ""
	isMoveTarget := m.isMoveMode && item.Goal.Path == m.moveTarget
	if isMoveTarget {
		movePrefix = IconMove + " "
	}

	// Search match highlighting
	isSearchMatch := m.searchMatchIDs[item.ID]
	name := item.Name
	if isSearchMatch && m.searchQuery != "" {
		if isSelected {
			name = highlightMatch(name, m.searchQuery, SearchCharSelectedStyle, SelectedStyle)
		} else {
			name = highlightMatch(name, m.searchQuery, SearchCharStyle, SearchRowStyle)
		}
	}

	line := indent + movePrefix + expandIcon + statusIcon + " " + name

	// Pad to width
	lineWidth := lipgloss.Width(line)
	if lineWidth < width {
		line += strings.Repeat(" ", width-lineWidth)
	}

	if isMoveTarget {
		line = MoveStyle.Render(line)
	} else if isSearchMatch && !isSelected {
		line = SearchRowStyle.Render(line)
	} else if isSelected {
		line = SelectedStyle.Render(line)
	}

	return line
}

func (m Model) renderNotesPanel(width, height int) string {
	if m.cursor >= len(m.visibleItems) || len(m.visibleItems) == 0 {
		return FooterStyle.Render(" Select a goal to view notes")
	}

	item := m.visibleItems[m.cursor]
	if item.IsSectionHeader {
		return FooterStyle.Render(" Select a goal to view notes")
	}
	goal := item.Goal

	// Reserve last line for file path
	bodyHeight := height - 1
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	// Build header markdown (title + metadata + links) — shared between view and edit
	header := m.renderGoalHeader(goal)

	// File path line at the bottom
	filePath := goal.FilePath
	if filePath == "" {
		filePath = goal.Path + "/goal.md"
	}
	pathLine := lipgloss.NewStyle().Foreground(ColorGrayDim).Render(fileHyperlink(filePath))

	if m.isEditing {
		// Render header, then textarea, then file path
		var headerRendered string
		if m.glamourRenderer != nil {
			var err error
			headerRendered, err = m.glamourRenderer.Render(header)
			if err != nil {
				headerRendered = header
			}
		} else {
			headerRendered = header
		}
		headerRendered = strings.TrimRight(headerRendered, "\n ")
		headerLines := strings.Split(headerRendered, "\n")

		var lines []string
		lines = append(lines, headerLines...)
		editorLines := strings.Split(m.noteEditor.View(), "\n")
		lines = append(lines, editorLines...)

		// Truncate to bodyHeight
		if len(lines) > bodyHeight {
			lines = lines[:bodyHeight]
		}
		// Pad to pin file path at the bottom
		for len(lines) < bodyHeight {
			lines = append(lines, "")
		}
		lines = append(lines, pathLine)
		return strings.Join(lines, "\n")
	}

	// Normal view mode — full markdown
	var md strings.Builder
	md.WriteString(header)

	if goal.Body != "" {
		md.WriteString(goal.Body)
		if !strings.HasSuffix(goal.Body, "\n") {
			md.WriteString("\n")
		}
	}

	// Render with glamour (cached renderer)
	var rendered string
	if m.glamourRenderer != nil {
		var err error
		rendered, err = m.glamourRenderer.Render(md.String())
		if err != nil {
			rendered = md.String()
		}
	} else {
		rendered = md.String()
	}

	// Trim trailing whitespace and split to lines
	rendered = strings.TrimRight(rendered, "\n ")
	lines := strings.Split(rendered, "\n")

	// Apply scroll offset
	scroll := m.notesScroll
	if scroll > len(lines)-1 {
		scroll = len(lines) - 1
	}
	if scroll < 0 {
		scroll = 0
	}
	lines = lines[scroll:]

	// Truncate to bodyHeight
	if len(lines) > bodyHeight {
		lines = lines[:bodyHeight]
	}

	// Pad to pin file path at the bottom
	for len(lines) < bodyHeight {
		lines = append(lines, "")
	}
	lines = append(lines, pathLine)

	return strings.Join(lines, "\n")
}

// renderGoalHeader builds the markdown header (title, metadata, links) for a goal.
func (m Model) renderGoalHeader(goal *store.Goal) string {
	var md strings.Builder

	md.WriteString("# " + goal.Title + "\n\n")

	var meta []string
	if goal.Horizon != "" {
		meta = append(meta, "**Horizon:** "+string(goal.Horizon))
	}
	if goal.Status != "" {
		meta = append(meta, "**Status:** "+string(goal.Status))
	}
	if len(goal.Tags) > 0 {
		meta = append(meta, "**Tags:** "+strings.Join(goal.Tags, ", "))
	}
	if len(meta) > 0 {
		md.WriteString(strings.Join(meta, " | ") + "\n\n")
	}

	if len(goal.Links) > 0 {
		for k, v := range goal.Links {
			md.WriteString("- **" + k + ":** " + v + "\n")
		}
		md.WriteString("\n")
	}

	return md.String()
}

func (m Model) renderFooter(width int) string {
	help := m.keys.ShortHelp()
	if m.isInputMode || m.isRenameMode {
		help = "enter confirm  esc cancel"
	} else if m.isEditing {
		help = "esc save & exit  ctrl+s save  ctrl+c cancel"
	} else if m.isSearching {
		help = "type to search  enter/↓ keep filter  esc clear"
	} else if m.searchQuery != "" {
		help = "esc/enter clear filter  ↑↓ nav"
	} else if m.isMoveMode {
		help = "↑↓ reorder  ← unparent  → reparent  enter/esc exit move"
	} else if m.focusedPane == 1 {
		help = "↑↓ scroll notes  tab tree  e edit  E $EDITOR  ? help"
	}
	return FooterStyle.Render(help)
}

func (m Model) renderHelpModal() string {
	var b strings.Builder

	b.WriteString(ModalTitleStyle.Render("Keyboard Shortcuts"))
	b.WriteString("\n\n")

	keyStyle := lipgloss.NewStyle().Foreground(ColorBlue).Width(16)
	descStyle := lipgloss.NewStyle().Foreground(ColorWhite)

	for _, binding := range m.keys.FullHelp() {
		b.WriteString(keyStyle.Render(binding[0]))
		b.WriteString(descStyle.Render(binding[1]))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(FooterStyle.Render("Press Esc or ? to close"))

	return ModalStyle.Render(b.String())
}

func (m Model) renderDeleteModal() string {
	var b strings.Builder

	b.WriteString(ModalTitleStyle.Render("Delete Goal"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("Delete '%s' and all sub-goals?\n\n", m.deleteTarget))
	b.WriteString(lipgloss.NewStyle().Foreground(ColorGreen).Render("[y]") + " Yes  ")
	b.WriteString(lipgloss.NewStyle().Foreground(ColorRed).Render("[n]") + " No")

	return ModalStyle.Render(b.String())
}

// highlightMatch splits name into before/match/after and styles the match portion
// with charStyle, and the rest with rowStyle. The match is case-insensitive.
func highlightMatch(name, query string, charStyle, rowStyle lipgloss.Style) string {
	lower := strings.ToLower(name)
	idx := strings.Index(lower, strings.ToLower(query))
	if idx < 0 {
		return rowStyle.Render(name)
	}
	before := name[:idx]
	match := name[idx : idx+len(query)]
	after := name[idx+len(query):]

	var result string
	if before != "" {
		result += rowStyle.Render(before)
	}
	result += charStyle.Render(match)
	if after != "" {
		result += rowStyle.Render(after)
	}
	return result
}

// fileHyperlink wraps a file path in an OSC 8 terminal hyperlink so it's clickable.
func fileHyperlink(path string) string {
	url := "file://" + path
	return fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", url, path)
}

// Helper functions

func getLine(block string, idx int, width int) string {
	lines := strings.Split(block, "\n")
	if idx < len(lines) {
		line := lines[idx]
		lineWidth := lipgloss.Width(line)
		if lineWidth < width {
			return line + strings.Repeat(" ", width-lineWidth)
		}
		return line
	}
	return strings.Repeat(" ", width)
}

func placeOverlay(modal string, width, height int) string {
	modalLines := strings.Split(modal, "\n")

	topPadding := (height - len(modalLines)) / 2
	if topPadding < 0 {
		topPadding = 0
	}

	leftPadding := (width - lipgloss.Width(modalLines[0])) / 2
	if leftPadding < 0 {
		leftPadding = 0
	}

	var result strings.Builder
	for i := 0; i < topPadding; i++ {
		result.WriteString("\n")
	}

	for _, line := range modalLines {
		result.WriteString(strings.Repeat(" ", leftPadding))
		result.WriteString(line)
		result.WriteString("\n")
	}

	return result.String()
}

func countGoals(goals []*store.Goal) int {
	count := 0
	for _, g := range goals {
		count++
		count += countGoals(g.Children)
	}
	return count
}

func countComplete(goals []*store.Goal) int {
	count := 0
	for _, g := range goals {
		if g.IsComplete() {
			count++
		}
		count += countComplete(g.Children)
	}
	return count
}
