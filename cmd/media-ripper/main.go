package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/Bparsons0904/ripper/internal/config"
)

var (
	// Color palette
	primaryBlue   = lipgloss.Color("39")   // Bright blue
	lightBlue     = lipgloss.Color("75")   // Light blue
	darkBlue      = lipgloss.Color("25")   // Dark blue
	accent        = lipgloss.Color("99")   // Purple
	gray          = lipgloss.Color("250")  // Lighter gray
	green         = lipgloss.Color("46")   // Success green

	// Main container with blue border
	containerStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryBlue).
			Padding(1, 2).
			Margin(1, 2)

	// Title styling
	titleStyle = lipgloss.NewStyle().
			Foreground(accent).
			Bold(true).
			Align(lipgloss.Center).
			Margin(0, 0, 1, 0)

	// Subtitle styling
	subtitleStyle = lipgloss.NewStyle().
			Foreground(lightBlue).
			Align(lipgloss.Center).
			Italic(true).
			Margin(0, 0, 2, 0)

	// Content sections
	welcomeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Bold(true).
			Margin(0, 0, 1, 0)

	descriptionStyle = lipgloss.NewStyle().
			Foreground(gray).
			Margin(0, 0, 2, 0)

	featuresHeaderStyle = lipgloss.NewStyle().
			Foreground(lightBlue).
			Bold(true).
			Margin(0, 0, 1, 0)

	featureStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			MarginLeft(2)

	statusStyle = lipgloss.NewStyle().
			Foreground(green).
			Bold(true).
			Margin(2, 0, 1, 0)

	helpStyle = lipgloss.NewStyle().
			Foreground(gray).
			Align(lipgloss.Center).
			Italic(true).
			Margin(1, 0, 0, 0)
)

type Screen int

const (
	WelcomeScreen Screen = iota
	SettingsMenuScreen
	PathsSettingsScreen
	CDRippingSettingsScreen
	ToolsSettingsScreen
	UISettingsScreen
)

type model struct {
	ready         bool
	config        *config.Config
	currentScreen Screen
	selectedItem  int
	isEditing     bool
	editValue     string
}

func initialModel() model {
	// Initialize configuration on startup
	cfg, err := config.InitializeConfig()
	if err != nil {
		fmt.Printf("Error initializing config: %v\n", err)
		cfg = config.DefaultConfig() // fallback to defaults
	}
	
	return model{
		ready:         true,
		config:        cfg,
		currentScreen: WelcomeScreen,
		selectedItem:  0,
		isEditing:     false,
		editValue:     "",
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.currentScreen {
		case WelcomeScreen:
			return m.updateWelcome(msg)
		case SettingsMenuScreen:
			return m.updateSettingsMenu(msg)
		case PathsSettingsScreen:
			return m.updatePathsSettings(msg)
		// Add other screen handlers as needed
		default:
			return m.updateWelcome(msg)
		}
	}
	return m, nil
}

func (m model) updatePathsSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	pathsFields := []string{"Music Directory", "Movies Directory", "Config Directory", "Log File"}
	
	if m.isEditing {
		// Handle editing mode
		switch msg.String() {
		case "enter":
			// Save the edited value
			switch m.selectedItem {
			case 0:
				m.config.Paths.Music = m.editValue
			case 1:
				m.config.Paths.Movies = m.editValue
			case 2:
				m.config.Paths.Config = m.editValue
			case 3:
				m.config.Paths.LogFile = m.editValue
			}
			// Save config to file
			if err := m.config.Save(config.GetConfigPath()); err != nil {
				fmt.Printf("Error saving config: %v\n", err)
			}
			m.isEditing = false
			m.editValue = ""
			return m, nil
		case "esc":
			// Cancel editing
			m.isEditing = false
			m.editValue = ""
			return m, nil
		default:
			// Handle text input
			if msg.String() == "backspace" {
				if len(m.editValue) > 0 {
					m.editValue = m.editValue[:len(m.editValue)-1]
				}
			} else if len(msg.String()) == 1 {
				// Add character to edit value
				m.editValue += msg.String()
			}
			return m, nil
		}
	} else {
		// Handle navigation mode
		switch msg.String() {
		case "q", "esc":
			m.currentScreen = SettingsMenuScreen
			return m, nil
		case "up", "k":
			if m.selectedItem > 0 {
				m.selectedItem--
			}
			return m, nil
		case "down", "j":
			if m.selectedItem < len(pathsFields)-1 {
				m.selectedItem++
			}
			return m, nil
		case "enter":
			// Start editing the selected field
			m.isEditing = true
			// Set current value as edit value
			switch m.selectedItem {
			case 0:
				m.editValue = m.config.Paths.Music
			case 1:
				m.editValue = m.config.Paths.Movies
			case 2:
				m.editValue = m.config.Paths.Config
			case 3:
				m.editValue = m.config.Paths.LogFile
			}
			return m, nil
		}
	}
	return m, nil
}

func (m model) updateWelcome(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "s":
		m.currentScreen = SettingsMenuScreen
		m.selectedItem = 0
		return m, nil
	}
	return m, nil
}

func (m model) updateSettingsMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	settingsOptions := []string{"Paths", "CD Ripping", "Tools", "UI Settings"}
	
	switch msg.String() {
	case "q", "esc":
		m.currentScreen = WelcomeScreen
		return m, nil
	case "up", "k":
		if m.selectedItem > 0 {
			m.selectedItem--
		}
		return m, nil
	case "down", "j":
		if m.selectedItem < len(settingsOptions)-1 {
			m.selectedItem++
		}
		return m, nil
	case "enter":
		// Navigate to specific settings screen
		switch m.selectedItem {
		case 0:
			m.currentScreen = PathsSettingsScreen
		case 1:
			m.currentScreen = CDRippingSettingsScreen
		case 2:
			m.currentScreen = ToolsSettingsScreen
		case 3:
			m.currentScreen = UISettingsScreen
		}
		m.selectedItem = 0
		return m, nil
	}
	return m, nil
}

func (m model) View() string {
	if !m.ready {
		loading := lipgloss.NewStyle().
			Foreground(primaryBlue).
			Bold(true).
			Align(lipgloss.Center).
			Margin(10, 0)
		return loading.Render("Loading...")
	}
	
	switch m.currentScreen {
	case WelcomeScreen:
		return m.renderWelcome()
	case SettingsMenuScreen:
		return m.renderSettingsMenu()
	case PathsSettingsScreen:
		return m.renderPathsSettings()
	default:
		return m.renderWelcome()
	}
}

func (m model) renderWelcome() string {

	// Header section
	title := titleStyle.Render("🎵 Media Ripper TUI")
	subtitle := subtitleStyle.Render("A Terminal UI for ripping CDs, DVDs, and Blu-rays")

	// Welcome section
	welcome := welcomeStyle.Render("Welcome to Media Ripper!")
	description := descriptionStyle.Render(
		"This application will help you rip your media collection with an intuitive\n" +
			"terminal interface. Built with Go and powered by the Charm library.",
	)

	// Features section
	featuresHeader := featuresHeaderStyle.Render("✨ Features coming soon:")
	features := []string{
		"🔍 Drive detection and selection",
		"🎵 Audio CD ripping with abcde",
		"🎬 DVD/Blu-ray ripping with MakeMKV",
		"📊 Real-time progress tracking",
		"⚙️  Configurable settings",
		"🐳 Container support",
	}

	var featureList string
	for _, feature := range features {
		featureList += featureStyle.Render(feature) + "\n"
	}

	// Status section with config info
	configPath := "~/.config/media-ripper/config.toml"
	if m.config != nil {
		configPath = config.GetConfigPath()
	}
	status := statusStyle.Render("🎉 Status: Configuration loaded!")
	configInfo := descriptionStyle.Render(fmt.Sprintf("Config: %s", configPath))

	// Help section
	help := helpStyle.Render("Press 's' for Settings, 'q' or Ctrl+C to quit")

	// Combine all content
	content := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s",
		title,
		subtitle,
		welcome,
		description,
		featuresHeader,
		featureList,
		status,
		configInfo,
		help,
	)

	// Wrap everything in the container with blue border
	return containerStyle.Render(content)
}

func (m model) renderPathsSettings() string {
	title := titleStyle.Render("📁 Paths Settings")
	subtitle := subtitleStyle.Render("Configure directory paths and log file location")
	
	pathsFields := []string{"Music Directory", "Movies Directory", "Config Directory", "Log File"}
	pathsValues := []string{
		m.config.Paths.Music,
		m.config.Paths.Movies,
		m.config.Paths.Config,
		m.config.Paths.LogFile,
	}
	
	var fields string
	for i, field := range pathsFields {
		value := pathsValues[i]
		if m.isEditing && i == m.selectedItem {
			// Show edit value with cursor
			value = m.editValue + "█" // Block cursor
		}
		
		if i == m.selectedItem {
			// Highlighted field
			fieldStyle := lipgloss.NewStyle().
				Foreground(accent).
				Bold(true).
				Margin(0, 2)
			valueStyle := lipgloss.NewStyle().
				Foreground(lightBlue).
				Background(lipgloss.Color("235")).
				Padding(0, 1).
				Margin(0, 2)
			
			if m.isEditing {
				// Editing mode styling
				valueStyle = valueStyle.Background(accent).Foreground(lipgloss.Color("0"))
			}
			
			fields += fieldStyle.Render("▶ "+field+":") + "\n"
			fields += valueStyle.Render(value) + "\n\n"
		} else {
			// Regular field
			fieldStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("255")).
				Margin(0, 2)
			valueStyle := lipgloss.NewStyle().
				Foreground(gray).
				Margin(0, 2)
			
			fields += fieldStyle.Render("  "+field+":") + "\n"
			fields += valueStyle.Render("  "+value) + "\n\n"
		}
	}
	
	var help string
	if m.isEditing {
		help = helpStyle.Render("Type to edit • Enter to save • Esc to cancel")
	} else {
		help = helpStyle.Render("↑/↓ or j/k to navigate • Enter to edit • Esc/q to go back")
	}
	
	content := fmt.Sprintf("%s\n%s\n\n%s%s",
		title,
		subtitle,
		fields,
		help,
	)
	
	return containerStyle.Render(content)
}

func (m model) renderSettingsMenu() string {
	title := titleStyle.Render("⚙️ Settings")
	subtitle := subtitleStyle.Render("Choose a category to configure")
	
	settingsOptions := []string{
		"📁 Paths",
		"💿 CD Ripping", 
		"🔧 Tools",
		"🎨 UI Settings",
	}
	
	var options string
	for i, option := range settingsOptions {
		if i == m.selectedItem {
			// Highlighted option
			selected := lipgloss.NewStyle().
				Foreground(accent).
				Bold(true).
				Background(lightBlue).
				Padding(0, 1).
				Margin(0, 2)
			options += selected.Render("▶ "+option) + "\n"
		} else {
			// Regular option
			regular := lipgloss.NewStyle().
				Foreground(lipgloss.Color("255")).
				Margin(0, 2)
			options += regular.Render("  "+option) + "\n"
		}
	}
	
	help := helpStyle.Render("↑/↓ or j/k to navigate • Enter to select • Esc/q to go back")
	
	content := fmt.Sprintf("%s\n%s\n\n%s\n%s",
		title,
		subtitle,
		options,
		help,
	)
	
	return containerStyle.Render(content)
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v", err)
		os.Exit(1)
	}
}

