package tui

import "github.com/charmbracelet/lipgloss"

// Color palette — adapted from gha-analyzer
var (
	ColorPurple      = lipgloss.Color("#7D56F4")
	ColorGreen       = lipgloss.Color("#25A065")
	ColorBlue        = lipgloss.Color("#4285F4")
	ColorRed         = lipgloss.Color("#E05252")
	ColorYellow      = lipgloss.Color("#E5C07B")
	ColorGray        = lipgloss.Color("#626262")
	ColorGrayDim     = lipgloss.Color("#404040")
	ColorWhite       = lipgloss.Color("#FFFFFF")
	ColorOffWhite    = lipgloss.Color("#D0D0D0")
	ColorMagenta     = lipgloss.Color("#C678DD")
	ColorSelectionBg = lipgloss.Color("#2D3B4D")
	ColorCyan        = lipgloss.Color("#56B6C2")
	ColorOrange      = lipgloss.Color("#D19A66")
	ColorMoveBg      = lipgloss.Color("#3E2F1F")
)

// Header styles
var (
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPurple)

	HeaderCountStyle = lipgloss.NewStyle().
				Foreground(ColorGray)

	FooterStyle = lipgloss.NewStyle().
			Foreground(ColorGray)
)

// Tab styles
var (
	ActiveTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWhite).
			Background(ColorPurple).
			Padding(0, 1)

	InactiveTabStyle = lipgloss.NewStyle().
				Foreground(ColorGray).
				Padding(0, 1)
)

// Tree item styles
var (
	SelectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWhite).
			Background(ColorSelectionBg)

	NormalStyle = lipgloss.NewStyle()

	CompleteStyle = lipgloss.NewStyle().
			Foreground(ColorGreen)

	InProgressStyle = lipgloss.NewStyle().
			Foreground(ColorYellow)

	IncompleteStyle = lipgloss.NewStyle().
			Foreground(ColorOffWhite)

	MoveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorOrange).
			Background(ColorMoveBg)

	DepthIndent = "  "
)

// Horizon styles
var (
	HorizonTodayStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorRed)

	HorizonTomorrowStyle = lipgloss.NewStyle().
				Foreground(ColorYellow)

	HorizonFutureStyle = lipgloss.NewStyle().
				Foreground(ColorGray)
)

// Panel styles
var (
	PanelBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorGrayDim)

	NotesPanelStyle = lipgloss.NewStyle().
			Padding(0, 1)
)

// Modal styles
var (
	ModalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPurple).
			Padding(1, 2)

	ModalTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPurple)

	ModalLabelStyle = lipgloss.NewStyle().
			Foreground(ColorGray).
			Width(14)

	ModalValueStyle = lipgloss.NewStyle().
			Foreground(ColorWhite)
)

// Input styles
var (
	InputPromptStyle = lipgloss.NewStyle().
				Foreground(ColorPurple).
				Bold(true)

	InputStyle = lipgloss.NewStyle().
			Foreground(ColorWhite)
)

// Search styles
var (
	ColorSearchRowBg  = lipgloss.Color("#1E1A2E")
	ColorSearchCharBg = lipgloss.Color("#2E2545")

	SearchBarStyle = lipgloss.NewStyle().
			Foreground(ColorWhite)

	SearchRowStyle = lipgloss.NewStyle().
			Background(ColorSearchRowBg)

	SearchCharStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPurple).
			Background(ColorSearchCharBg)

	SearchCharSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorPurple).
				Background(ColorSelectionBg)

	SearchCountStyle = lipgloss.NewStyle().
				Foreground(ColorGray)
)

// Status icons
const (
	IconComplete   = "✓"
	IconInProgress = "◐"
	IconIncomplete = "○"
	IconExpanded   = "▼"
	IconCollapsed  = "▶"
	IconMove       = "↕"
)
