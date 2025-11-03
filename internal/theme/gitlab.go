package theme

import (
	"image/color"

	"github.com/charmbracelet/fang"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"               // v1 - used by huh
	lipglossv2 "github.com/charmbracelet/lipgloss/v2" // v2 - used by fang
	"github.com/muesli/termenv"
)

// GitLabColors contains the GitLab product color palette.
// These colors are designed for accessibility across different terminal backgrounds.
type GitLabColors struct {
	// Brand colors
	Orange color.Color // GitLab brand orange (#FC6D26)
	Purple color.Color // GitLab brand purple (#7759C2 light, #A989F5 dark)

	// Semantic colors
	Blue  color.Color // Blue for current/active states (#1068BF light, #4285F4 dark)
	Green color.Color // Green for success (#217645 light, #34D058 dark)
	Red   color.Color // Red for errors (#C91C00 light, #F97583 dark)

	// Neutral colors
	Text   color.Color // High-contrast text (#171321 light, #FAFAFA dark)
	Subtle color.Color // Subtle elements (#6B6B73 light, #B0B0B0 dark)
}

// NewGitLabColors creates GitLab brand colors using lightDarkFunc for accessibility.
// The lightDarkFunc automatically selects colors based on terminal background.
func NewGitLabColors(lightDarkFunc lipglossv2.LightDarkFunc) GitLabColors {
	return GitLabColors{
		Orange: lightDarkFunc(lipglossv2.Color("#FC6D26"), lipglossv2.Color("#FC6D26")),
		Purple: lightDarkFunc(lipglossv2.Color("#7759C2"), lipglossv2.Color("#A989F5")),
		Blue:   lightDarkFunc(lipglossv2.Color("#1068BF"), lipglossv2.Color("#4285F4")),
		Green:  lightDarkFunc(lipglossv2.Color("#217645"), lipglossv2.Color("#34D058")),
		Red:    lightDarkFunc(lipglossv2.Color("#C91C00"), lipglossv2.Color("#F97583")),
		Text:   lightDarkFunc(lipglossv2.Color("#171321"), lipglossv2.Color("#FAFAFA")),
		Subtle: lightDarkFunc(lipglossv2.Color("#6B6B73"), lipglossv2.Color("#B0B0B0")),
	}
}

// FangColorScheme returns a fang color scheme with GitLab product colors.
// This maintains the existing fang styling established in the CLI.
func FangColorScheme(lightDarkFunc lipglossv2.LightDarkFunc) fang.ColorScheme {
	colors := NewGitLabColors(lightDarkFunc)
	scheme := fang.DefaultColorScheme(lightDarkFunc)

	// Apply GitLab product color semantics
	scheme.Title = colors.Text            // Main command titles
	scheme.Command = colors.Purple        // Subcommands - purple for brand consistency
	scheme.Flag = colors.Purple           // Flags - purple for brand consistency
	scheme.FlagDefault = colors.Subtle    // Default flag values - subtle
	scheme.Description = colors.Text      // Command descriptions - high contrast
	scheme.Program = colors.Orange        // Program name (glab) - brand consistency
	scheme.Argument = colors.Blue         // Command arguments - blue for interactive elements
	scheme.DimmedArgument = colors.Subtle // Optional/dimmed arguments - reduced prominence
	scheme.QuotedString = colors.Text     // Quoted strings - high contrast
	scheme.Comment = colors.Subtle        // Comments - reduced visual weight
	scheme.Help = colors.Text             // Help text - maximum readability
	scheme.Dash = colors.Subtle           // Dashes and separators - subtle structure
	scheme.ErrorHeader = [2]color.Color{lipglossv2.Color("#FFFFFF"), lipglossv2.Color("#C91C00")}
	scheme.ErrorDetails = colors.Red // Error details in GitLab red

	return scheme
}

// HuhTheme returns a huh theme with GitLab product colors.
// This ensures consistent branding across interactive prompts.
// Note: huh uses lipgloss v1, which doesn't support adaptive colors like v2,
// so we detect the terminal background and choose appropriate colors.
func HuhTheme() *huh.Theme {
	theme := huh.ThemeBase()

	// Detect terminal background to choose appropriate colors
	isDark := termenv.HasDarkBackground()

	// GitLab brand colors adapted for terminal background
	var gitlabPurple, gitlabOrange, gitlabBlue, gitlabRed, gitlabSubtle lipgloss.Color

	if isDark {
		// Dark terminal: use lighter, brighter colors for visibility
		gitlabPurple = lipgloss.Color("#A989F5") // Lighter purple for dark backgrounds
		gitlabOrange = lipgloss.Color("#FC6D26") // Orange works on both
		gitlabBlue = lipgloss.Color("#4285F4")   // Lighter blue
		gitlabRed = lipgloss.Color("#F97583")    // Lighter red
		gitlabSubtle = lipgloss.Color("#B0B0B0") // Lighter gray
	} else {
		// Light terminal: use darker colors for contrast
		gitlabPurple = lipgloss.Color("#7759C2") // Darker purple for light backgrounds
		gitlabOrange = lipgloss.Color("#FC6D26") // Orange works on both
		gitlabBlue = lipgloss.Color("#1068BF")   // Darker blue
		gitlabRed = lipgloss.Color("#C91C00")    // Darker red
		gitlabSubtle = lipgloss.Color("#6B6B73") // Darker gray
	}

	// Focused field styles (when user is interacting)
	theme.Focused.Base = theme.Focused.Base.BorderForeground(gitlabPurple)
	theme.Focused.Title = theme.Focused.Title.Foreground(gitlabPurple).Bold(true)
	theme.Focused.Description = theme.Focused.Description.Foreground(gitlabSubtle)
	theme.Focused.SelectedOption = theme.Focused.SelectedOption.Foreground(gitlabBlue)
	theme.Focused.UnselectedOption = theme.Focused.UnselectedOption.Foreground(gitlabSubtle)
	theme.Focused.SelectedPrefix = lipgloss.NewStyle().Foreground(gitlabOrange).SetString("✓ ")
	theme.Focused.UnselectedPrefix = lipgloss.NewStyle().Foreground(gitlabSubtle).SetString("  ")

	// Text input styling
	theme.Focused.TextInput.Cursor = theme.Focused.TextInput.Cursor.Foreground(gitlabOrange)
	theme.Focused.TextInput.Placeholder = theme.Focused.TextInput.Placeholder.Foreground(gitlabSubtle)
	theme.Focused.TextInput.Prompt = theme.Focused.TextInput.Prompt.Foreground(gitlabPurple)

	// Blurred field styles (when not active)
	theme.Blurred.Base = theme.Blurred.Base.BorderForeground(gitlabSubtle)
	theme.Blurred.Title = theme.Blurred.Title.Foreground(gitlabSubtle)
	theme.Blurred.Description = theme.Blurred.Description.Foreground(gitlabSubtle)
	theme.Blurred.TextInput.Prompt = theme.Blurred.TextInput.Prompt.Foreground(gitlabSubtle)

	// Confirm button styles
	// Focused button: GitLab orange background with dark text for strong visual distinction
	// Using dark text (#171321) instead of white for better contrast on orange
	theme.Focused.FocusedButton = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#171321")).
		Background(gitlabOrange).
		Bold(true).
		Padding(0, 2).
		MarginRight(1)
	// Blurred button: subtle gray background with white text
	// Using a very subtle background so both buttons have similar visual weight
	var subtleBackground lipgloss.Color
	if isDark {
		subtleBackground = lipgloss.Color("#3a3a3a") // Subtle gray for dark terminals
	} else {
		subtleBackground = lipgloss.Color("#e0e0e0") // Subtle gray for light terminals
	}
	theme.Focused.BlurredButton = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(subtleBackground).
		Padding(0, 2).
		MarginRight(1)

	// Error styling
	theme.Focused.ErrorIndicator = theme.Focused.ErrorIndicator.Foreground(gitlabRed)
	theme.Focused.ErrorMessage = theme.Focused.ErrorMessage.Foreground(gitlabRed)
	theme.Blurred.ErrorIndicator = theme.Blurred.ErrorIndicator.Foreground(gitlabRed)
	theme.Blurred.ErrorMessage = theme.Blurred.ErrorMessage.Foreground(gitlabRed)

	return theme
}
