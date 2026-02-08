package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all key bindings for the TUI.
type KeyMap struct {
	Up           key.Binding
	Down         key.Binding
	Left         key.Binding
	Right        key.Binding
	Enter        key.Binding
	Space        key.Binding
	Tab          key.Binding
	NextQueue    key.Binding
	PrevQueue    key.Binding
	InlineEdit   key.Binding
	ExternalEdit key.Binding
	Add          key.Binding
	AddTop       key.Binding
	Delete       key.Binding
	Rename       key.Binding
	ToggleExpand key.Binding
	Reload       key.Binding
	Sync         key.Binding
	Help         key.Binding
	Move         key.Binding
	Search       key.Binding
	Quit         key.Binding
	Today        key.Binding
	Tomorrow     key.Binding
	Future       key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "collapse"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "expand"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "toggle expand"),
		),
		Space: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "toggle status"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch pane"),
		),
		NextQueue: key.NewBinding(
			key.WithKeys("]"),
			key.WithHelp("]", "next queue"),
		),
		PrevQueue: key.NewBinding(
			key.WithKeys("["),
			key.WithHelp("[", "prev queue"),
		),
		InlineEdit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "inline edit"),
		),
		ExternalEdit: key.NewBinding(
			key.WithKeys("E"),
			key.WithHelp("E", "$EDITOR"),
		),
		Add: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add sub-goal"),
		),
		AddTop: key.NewBinding(
			key.WithKeys("A"),
			key.WithHelp("A", "add top-level goal"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Rename: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "rename goal"),
		),
		ToggleExpand: key.NewBinding(
			key.WithKeys("C"),
			key.WithHelp("C", "toggle expand/collapse all"),
		),
		Reload: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "reload"),
		),
		Sync: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "git sync"),
		),
		Move: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "move mode"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Today: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "set today"),
		),
		Tomorrow: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "set tomorrow"),
		),
		Future: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "set future"),
		),
	}
}

// ShortHelp returns the footer help text.
func (k KeyMap) ShortHelp() string {
	return "↑↓ nav  tab pane  e edit  E $EDITOR  space toggle  / search  r rename  a/A add  m move  ? help"
}

// FullHelp returns all key bindings for the help modal.
func (k KeyMap) FullHelp() [][]string {
	return [][]string{
		{"↑/k", "Move up"},
		{"↓/j", "Move down"},
		{"←/h", "Collapse / go to parent"},
		{"→/l", "Expand"},
		{"enter", "Toggle expand/collapse"},
		{"space", "Toggle complete/incomplete"},
		{"tab", "Switch pane (tree / notes)"},
		{"]", "Next queue item"},
		{"[", "Previous queue item"},
		{"e", "Inline edit notes"},
		{"E", "Edit in $EDITOR"},
		{"/", "Search tree"},
		{"a", "Add sub-goal under selection"},
		{"A", "Add top-level goal"},
		{"r", "Rename goal"},
		{"d", "Delete goal (with confirmation)"},
		{"C", "Toggle expand/collapse all"},
		{"m", "Enter move mode (reorder/reparent)"},
		{"1/2/3", "Set horizon: today/tomorrow/future"},
		{"R", "Reload from filesystem"},
		{"s", "Git sync"},
		{"?", "Toggle help"},
		{"q", "Quit"},
	}
}
