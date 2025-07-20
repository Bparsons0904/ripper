package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/Bparsons0904/ripper/internal/config"
	"github.com/Bparsons0904/ripper/internal/drives"
	"github.com/Bparsons0904/ripper/internal/ripper"
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

// Message types for async operations
type cdDetectedMsg struct {
	cdInfo *ripper.CDInfo
	err    error
}

type rippingProgressMsg ripper.ProgressInfo

type spinnerTickMsg struct{}

type Screen int

const (
	WelcomeScreen Screen = iota
	CDRippingScreen
	SettingsMenuScreen
	DrivesSettingsScreen
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
	availableDrives []drives.DriveInfo
	isRipping     bool
	rippingProgress int
	rippingStatus string
	cdRipper      *ripper.CDRipper
	cdInfo        *ripper.CDInfo
	spinnerFrame  int
}

func initialModel() model {
	// Initialize configuration on startup
	cfg, err := config.InitializeConfig()
	if err != nil {
		fmt.Printf("Error initializing config: %v\n", err)
		cfg = config.DefaultConfig() // fallback to defaults
	}
	
	// Detect available drives
	availableDrives, err := drives.DetectDrives()
	if err != nil {
		fmt.Printf("Warning: Could not detect drives: %v\n", err)
		availableDrives = []drives.DriveInfo{}
	}
	
	// Initialize CD ripper
	cdRipper := ripper.NewCDRipper(cfg)

	return model{
		ready:           true,
		config:          cfg,
		currentScreen:   WelcomeScreen,
		selectedItem:    0,
		isEditing:       false,
		editValue:       "",
		availableDrives: availableDrives,
		isRipping:       false,
		rippingProgress: 0,
		rippingStatus:   "",
		cdRipper:        cdRipper,
		cdInfo:          nil,
		spinnerFrame:    0,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

// Commands for async operations
func detectCDCmd(cdRipper *ripper.CDRipper) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		cdInfo, err := cdRipper.DetectCD()
		return cdDetectedMsg{cdInfo: cdInfo, err: err}
	})
}

func startRippingCmd(cdRipper *ripper.CDRipper, cdInfo *ripper.CDInfo) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		go func() {
			fmt.Printf("DEBUG: RipCD goroutine started\n")
			if err := cdRipper.RipCD(cdInfo); err != nil {
				fmt.Printf("DEBUG: RipCD failed: %v\n", err)
			}
		}()
		// Return a message to indicate ripping started
		return rippingProgressMsg(ripper.ProgressInfo{
			Status: "Ripping started...",
			Progress: 0,
		})
	})
}

func listenForProgressCmd(progressCh <-chan ripper.ProgressInfo) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		progress := <-progressCh
		return rippingProgressMsg(progress)
	})
}

func spinnerCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*200, func(time.Time) tea.Msg {
		return spinnerTickMsg{}
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinnerTickMsg:
		if m.isRipping {
			m.spinnerFrame++
			return m, spinnerCmd()
		}
		return m, nil
	case cdDetectedMsg:
		if msg.err != nil {
			m.rippingStatus = fmt.Sprintf("Error detecting CD: %v", msg.err)
			m.cdInfo = nil
		} else {
			m.cdInfo = msg.cdInfo
			m.rippingStatus = ""  // Clear status once CD is detected successfully
		}
		return m, nil
	case rippingProgressMsg:
		progress := ripper.ProgressInfo(msg)
		m.rippingProgress = progress.Progress
		m.rippingStatus = progress.Status
		if progress.Error != nil {
			m.isRipping = false
			m.rippingStatus = fmt.Sprintf("Error: %v", progress.Error)
			return m, nil
		}
		if progress.Progress >= 100 {
			m.isRipping = false
			return m, nil
		}
		// Only continue listening for progress if we're actually ripping
		if m.isRipping {
			return m, listenForProgressCmd(m.cdRipper.GetProgressChannel())
		}
		return m, nil
	case tea.KeyMsg:
		switch m.currentScreen {
		case WelcomeScreen:
			return m.updateWelcome(msg)
		case CDRippingScreen:
			return m.updateCDRipping(msg)
		case SettingsMenuScreen:
			return m.updateSettingsMenu(msg)
		case DrivesSettingsScreen:
			return m.updateDrivesSettings(msg)
		case PathsSettingsScreen:
			return m.updatePathsSettings(msg)
		case CDRippingSettingsScreen:
			return m.updateCDRippingSettings(msg)
		case ToolsSettingsScreen:
			return m.updateToolsSettings(msg)
		case UISettingsScreen:
			return m.updateUISettings(msg)
		// Add other screen handlers as needed
		default:
			return m.updateWelcome(msg)
		}
	}
	return m, nil
}

func (m model) updateUISettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	uiFields := []string{"Theme", "Refresh Rate (ms)"}
	
	if m.isEditing {
		// Handle editing mode
		switch msg.String() {
		case "enter":
			// Save the edited value
			switch m.selectedItem {
			case 0: // Theme
				if m.editValue != "" {
					m.config.UI.Theme = m.editValue
				}
			case 1: // Refresh Rate
				if val := parseInt(m.editValue); val >= 50 && val <= 1000 {
					m.config.UI.RefreshRate = val
				}
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
			if m.selectedItem < len(uiFields)-1 {
				m.selectedItem++
			}
			return m, nil
		case "enter":
			// Start editing the selected field
			m.isEditing = true
			// Set current value as edit value
			switch m.selectedItem {
			case 0:
				m.editValue = m.config.UI.Theme
			case 1:
				m.editValue = fmt.Sprintf("%d", m.config.UI.RefreshRate)
			}
			return m, nil
		}
	}
	return m, nil
}

func (m model) updateCDRippingSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cdFields := []string{"Retry Count", "Retry Delay (sec)", "Initial Wait (sec)", "Auto Eject", "Output Format", "CDDB Method"}
	
	if m.isEditing {
		// Handle editing mode
		switch msg.String() {
		case "enter":
			// Save the edited value
			switch m.selectedItem {
			case 0: // Retry Count
				if val := parseInt(m.editValue); val >= 0 && val <= 10 {
					m.config.CDRipping.RetryCount = val
				}
			case 1: // Retry Delay
				if val := parseInt(m.editValue); val >= 0 && val <= 60 {
					m.config.CDRipping.RetryDelay = val
				}
			case 2: // Initial Wait
				if val := parseInt(m.editValue); val >= 0 && val <= 120 {
					m.config.CDRipping.InitialWait = val
				}
			case 4: // Output Format
				validFormats := []string{"flac", "mp3", "ogg", "wav"}
				for _, format := range validFormats {
					if m.editValue == format {
						m.config.CDRipping.OutputFormat = format
						break
					}
				}
			case 5: // CDDB Method
				validMethods := []string{"musicbrainz", "cddb", "none"}
				for _, method := range validMethods {
					if m.editValue == method {
						m.config.CDRipping.CDDBMethod = method
						break
					}
				}
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
			// Handle text input for editable fields
			if m.selectedItem != 3 { // Skip auto eject (it's toggled, not typed)
				if msg.String() == "backspace" {
					if len(m.editValue) > 0 {
						m.editValue = m.editValue[:len(m.editValue)-1]
					}
				} else if len(msg.String()) == 1 {
					// Add character to edit value
					m.editValue += msg.String()
				}
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
			if m.selectedItem < len(cdFields)-1 {
				m.selectedItem++
			}
			return m, nil
		case "enter", " ":
			if m.selectedItem == 3 { // Auto Eject - toggle boolean
				m.config.CDRipping.AutoEject = !m.config.CDRipping.AutoEject
				// Save config immediately for toggles
				if err := m.config.Save(config.GetConfigPath()); err != nil {
					fmt.Printf("Error saving config: %v\n", err)
				}
				return m, nil
			} else if m.selectedItem == 4 { // Output Format - cycle through options
				formats := []string{"flac", "mp3", "ogg", "wav"}
				currentIndex := -1
				for i, format := range formats {
					if m.config.CDRipping.OutputFormat == format {
						currentIndex = i
						break
					}
				}
				nextIndex := (currentIndex + 1) % len(formats)
				m.config.CDRipping.OutputFormat = formats[nextIndex]
				// Save config immediately
				if err := m.config.Save(config.GetConfigPath()); err != nil {
					fmt.Printf("Error saving config: %v\n", err)
				}
				return m, nil
			} else if m.selectedItem == 5 { // CDDB Method - cycle through options
				methods := []string{"musicbrainz", "cddb", "none"}
				currentIndex := -1
				for i, method := range methods {
					if m.config.CDRipping.CDDBMethod == method {
						currentIndex = i
						break
					}
				}
				nextIndex := (currentIndex + 1) % len(methods)
				m.config.CDRipping.CDDBMethod = methods[nextIndex]
				// Save config immediately
				if err := m.config.Save(config.GetConfigPath()); err != nil {
					fmt.Printf("Error saving config: %v\n", err)
				}
				return m, nil
			} else {
				// Start editing the selected field (numeric fields only)
				m.isEditing = true
				// Set current value as edit value
				switch m.selectedItem {
				case 0:
					m.editValue = fmt.Sprintf("%d", m.config.CDRipping.RetryCount)
				case 1:
					m.editValue = fmt.Sprintf("%d", m.config.CDRipping.RetryDelay)
				case 2:
					m.editValue = fmt.Sprintf("%d", m.config.CDRipping.InitialWait)
				}
				return m, nil
			}
		}
	}
	return m, nil
}

// Helper function to parse integers safely
func parseInt(s string) int {
	val := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			val = val*10 + int(r-'0')
		} else {
			return -1 // Invalid input
		}
	}
	return val
}

func (m model) updateToolsSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	toolsFields := []string{"ABCDE Path", "cd-discid Path", "MakeMKV Path"}
	
	if m.isEditing {
		// Handle editing mode
		switch msg.String() {
		case "enter":
			// Save the edited value
			switch m.selectedItem {
			case 0:
				m.config.Tools.AbcdePath = m.editValue
			case 1:
				m.config.Tools.CDDiscidPath = m.editValue
			case 2:
				m.config.Tools.MakeMKVPath = m.editValue
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
			if m.selectedItem < len(toolsFields)-1 {
				m.selectedItem++
			}
			return m, nil
		case "enter":
			// Start editing the selected field
			m.isEditing = true
			// Set current value as edit value
			switch m.selectedItem {
			case 0:
				m.editValue = m.config.Tools.AbcdePath
			case 1:
				m.editValue = m.config.Tools.CDDiscidPath
			case 2:
				m.editValue = m.config.Tools.MakeMKVPath
			}
			return m, nil
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

func (m model) updateDrivesSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		if m.selectedItem < len(m.availableDrives)-1 {
			m.selectedItem++
		}
		return m, nil
	case "enter", " ":
		if len(m.availableDrives) > 0 && m.selectedItem < len(m.availableDrives) {
			// Update config with selected drive
			selectedDrive := m.availableDrives[m.selectedItem]
			m.config.Drives.CDDrive = selectedDrive.Device
			
			// Save config
			if err := m.config.Save(config.GetConfigPath()); err != nil {
				fmt.Printf("Error saving config: %v\n", err)
			}
			
			// Return to settings menu
			m.currentScreen = SettingsMenuScreen
			return m, nil
		}
	case "r":
		// Refresh drive detection
		availableDrives, err := drives.DetectDrives()
		if err != nil {
			fmt.Printf("Warning: Could not detect drives: %v\n", err)
		} else {
			m.availableDrives = availableDrives
			// Reset selection if it's out of bounds
			if m.selectedItem >= len(m.availableDrives) {
				m.selectedItem = 0
			}
		}
		return m, nil
	}
	return m, nil
}

func (m model) updateWelcome(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "r":
		m.currentScreen = CDRippingScreen
		m.selectedItem = 0
		return m, nil
	case "s":
		m.currentScreen = SettingsMenuScreen
		m.selectedItem = 0
		return m, nil
	}
	return m, nil
}

func (m model) updateSettingsMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	settingsOptions := []string{"Drives", "Paths", "CD Ripping", "Tools", "UI Settings"}
	
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
			m.currentScreen = DrivesSettingsScreen
		case 1:
			m.currentScreen = PathsSettingsScreen
		case 2:
			m.currentScreen = CDRippingSettingsScreen
		case 3:
			m.currentScreen = ToolsSettingsScreen
		case 4:
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
	case CDRippingScreen:
		return m.renderCDRipping()
	case SettingsMenuScreen:
		return m.renderSettingsMenu()
	case DrivesSettingsScreen:
		return m.renderDrivesSettings()
	case PathsSettingsScreen:
		return m.renderPathsSettings()
	case CDRippingSettingsScreen:
		return m.renderCDRippingSettings()
	case ToolsSettingsScreen:
		return m.renderToolsSettings()
	case UISettingsScreen:
		return m.renderUISettings()
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
	help := helpStyle.Render("Press 'r' to Rip CD, 's' for Settings, 'q' or Ctrl+C to quit")

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

func (m model) renderUISettings() string {
	title := titleStyle.Render("🎨 UI Settings")
	subtitle := subtitleStyle.Render("Configure user interface preferences")
	
	uiFields := []string{"Theme", "Refresh Rate (ms)"}
	uiValues := []string{
		m.config.UI.Theme,
		fmt.Sprintf("%d", m.config.UI.RefreshRate),
	}
	
	var fields string
	for i, field := range uiFields {
		value := uiValues[i]
		
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
			fields += valueStyle.Render(value) + "\n\n"
		}
	}
	
	// Add helpful hints
	hintsStyle := lipgloss.NewStyle().
		Foreground(gray).
		Italic(true).
		Margin(1, 2)
	hints := hintsStyle.Render(
		"Hints: Theme can be any string • Refresh Rate must be between 50-1000ms for smooth performance",
	)
	
	var help string
	if m.isEditing {
		help = helpStyle.Render("Type to edit • Enter to save • Esc to cancel")
	} else {
		help = helpStyle.Render("↑/↓ or j/k to navigate • Enter to edit • Esc/q to go back")
	}
	
	content := fmt.Sprintf("%s\n%s\n\n%s%s\n%s",
		title,
		subtitle,
		fields,
		hints,
		help,
	)
	
	return containerStyle.Render(content)
}

func (m model) renderToolsSettings() string {
	title := titleStyle.Render("🔧 Tools Settings")
	subtitle := subtitleStyle.Render("Configure external tool paths (leave empty for auto-detection)")
	
	toolsFields := []string{"ABCDE Path", "cd-discid Path", "MakeMKV Path"}
	toolsValues := []string{
		m.config.Tools.AbcdePath,
		m.config.Tools.CDDiscidPath,
		m.config.Tools.MakeMKVPath,
	}
	
	var fields string
	for i, field := range toolsFields {
		value := toolsValues[i]
		if value == "" {
			value = "(auto-detect)"
		}
		
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
			} else if toolsValues[i] == "" {
				// Special styling for auto-detect
				valueStyle = valueStyle.Foreground(gray).Italic(true)
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
			
			if toolsValues[i] == "" {
				// Special styling for auto-detect
				valueStyle = valueStyle.Italic(true)
			}
			
			fields += fieldStyle.Render("  "+field+":") + "\n"
			fields += valueStyle.Render(value) + "\n\n"
		}
	}
	
	// Add helpful hints
	hintsStyle := lipgloss.NewStyle().
		Foreground(gray).
		Italic(true).
		Margin(1, 2)
	hints := hintsStyle.Render(
		"Hints: Leave paths empty for automatic detection in PATH • Use absolute paths like /usr/bin/abcde",
	)
	
	var help string
	if m.isEditing {
		help = helpStyle.Render("Type path or clear for auto-detect • Enter to save • Esc to cancel")
	} else {
		help = helpStyle.Render("↑/↓ or j/k to navigate • Enter to edit • Esc/q to go back")
	}
	
	content := fmt.Sprintf("%s\n%s\n\n%s%s\n%s",
		title,
		subtitle,
		fields,
		hints,
		help,
	)
	
	return containerStyle.Render(content)
}

func (m model) renderCDRippingSettings() string {
	title := titleStyle.Render("💿 CD Ripping Settings")
	subtitle := subtitleStyle.Render("Configure CD ripping behavior and formats")
	
	cdFields := []string{"Retry Count", "Retry Delay (sec)", "Initial Wait (sec)", "Auto Eject", "Output Format", "CDDB Method"}
	cdValues := []string{
		fmt.Sprintf("%d", m.config.CDRipping.RetryCount),
		fmt.Sprintf("%d", m.config.CDRipping.RetryDelay),
		fmt.Sprintf("%d", m.config.CDRipping.InitialWait),
		fmt.Sprintf("%t", m.config.CDRipping.AutoEject),
		m.config.CDRipping.OutputFormat,
		m.config.CDRipping.CDDBMethod,
	}
	
	var fields string
	for i, field := range cdFields {
		value := cdValues[i]
		
		// Special handling for boolean fields
		if i == 3 { // Auto Eject
			if m.config.CDRipping.AutoEject {
				value = "✓ Yes" // Checkmark
			} else {
				value = "✗ No" // X mark
			}
		}
		
		// Special handling for editing mode
		if m.isEditing && i == m.selectedItem && i != 3 && i != 4 && i != 5 {
			// Show edit value with cursor (skip for boolean and selectable)
			value = m.editValue + "█" // Block cursor
		}
		
		// Add cycling indicators for selectable options
		if i == 4 { // Output Format
			formats := []string{"flac", "mp3", "ogg", "wav"}
			for j, format := range formats {
				if format == value {
					value = fmt.Sprintf("%s (%d/%d)", value, j+1, len(formats))
					break
				}
			}
		} else if i == 5 { // CDDB Method
			methods := []string{"musicbrainz", "cddb", "none"}
			for j, method := range methods {
				if method == value {
					value = fmt.Sprintf("%s (%d/%d)", value, j+1, len(methods))
					break
				}
			}
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
			
			if m.isEditing && i != 3 && i != 4 && i != 5 {
				// Editing mode styling (skip for boolean and selectable)
				valueStyle = valueStyle.Background(accent).Foreground(lipgloss.Color("0"))
			} else if i == 3 {
				// Special styling for boolean toggle
				valueStyle = valueStyle.Background(green).Foreground(lipgloss.Color("0"))
			} else if i == 4 || i == 5 {
				// Special styling for selectable options
				valueStyle = valueStyle.Background(lightBlue).Foreground(lipgloss.Color("0"))
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
			fields += valueStyle.Render(value) + "\n\n"
		}
	}
	
	// Add validation hints
	hintsStyle := lipgloss.NewStyle().
		Foreground(gray).
		Italic(true).
		Margin(1, 2)
	hints := hintsStyle.Render(
		"Hints: Retry Count (0-10) • Delays in seconds • Formats: flac, mp3, ogg, wav • CDDB: musicbrainz, cddb, none",
	)
	
	var help string
	if m.isEditing {
		help = helpStyle.Render("Type to edit • Enter to save • Esc to cancel")
	} else {
		help = helpStyle.Render("↑/↓ or j/k to navigate • Enter/Space to edit/toggle/cycle • Esc/q to go back")
	}
	
	content := fmt.Sprintf("%s\n%s\n\n%s%s\n%s",
		title,
		subtitle,
		fields,
		hints,
		help,
	)
	
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
			fields += valueStyle.Render(value) + "\n\n"
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
		"💿 Drives",
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

func (m model) renderDrivesSettings() string {
	title := titleStyle.Render("💿 Drives Settings")
	subtitle := subtitleStyle.Render("Choose and configure optical drives")
	
	var content string
	
	if len(m.availableDrives) == 0 {
		noDrivesStyle := lipgloss.NewStyle().
			Foreground(accent).
			Bold(true).
			Align(lipgloss.Center).
			Margin(2, 0)
		
		noDriverMessage := noDrivesStyle.Render("No optical drives detected!")
		
		helpMessage := descriptionStyle.Render(
			"This could mean:\n" +
			"• No optical drives are connected\n" +
			"• Drives are not properly mounted\n" +
			"• You need elevated permissions to access drives\n\n" +
			"Press 'r' to refresh detection",
		)
		
		content = fmt.Sprintf("%s\n%s\n\n%s\n%s",
			title,
			subtitle,
			noDriverMessage,
			helpMessage,
		)
	} else {
		// Show current selection
		currentDriveStyle := lipgloss.NewStyle().
			Foreground(green).
			Bold(true).
			Margin(0, 2, 1, 2)
		
		currentSelection := currentDriveStyle.Render(
			fmt.Sprintf("Current: %s", m.config.Drives.CDDrive),
		)
		
		// List available drives
		var drivesList string
		for i, drive := range m.availableDrives {
			var driveInfo string
			
			// Format drive information
			mediaInfo := ""
			if drive.MediaType != "Unknown" {
				mediaInfo = fmt.Sprintf(" (%s)", drive.MediaType)
			}
			
			readOnlyInfo := ""
			if drive.IsReadOnly {
				readOnlyInfo = " [Read-Only]"
			}
			
			driveInfo = fmt.Sprintf("%s - %s%s%s", 
				drive.Device, 
				drive.Model, 
				mediaInfo,
				readOnlyInfo,
			)
			
			if i == m.selectedItem {
				// Highlighted drive
				selectedStyle := lipgloss.NewStyle().
					Foreground(accent).
					Bold(true).
					Background(lightBlue).
					Padding(0, 1).
					Margin(0, 2)
				drivesList += selectedStyle.Render("▶ "+driveInfo) + "\n"
			} else {
				// Regular drive
				regularStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("255")).
					Margin(0, 2)
				drivesList += regularStyle.Render("  "+driveInfo) + "\n"
			}
		}
		
		content = fmt.Sprintf("%s\n%s\n\n%s\n\n%s",
			title,
			subtitle,
			currentSelection,
			drivesList,
		)
	}
	
	// Help section
	var help string
	if len(m.availableDrives) == 0 {
		help = helpStyle.Render("Press 'r' to refresh • Esc/q to go back")
	} else {
		help = helpStyle.Render("↑/↓ or j/k to navigate • Enter/Space to select • 'r' to refresh • Esc/q to go back")
	}
	
	finalContent := fmt.Sprintf("%s\n%s", content, help)
	return containerStyle.Render(finalContent)
}

func (m model) updateCDRipping(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.isRipping {
		// During ripping, only allow quit
		switch msg.String() {
		case "q", "esc":
			fmt.Printf("DEBUG: User cancelled ripping\n")
			m.isRipping = false
			m.rippingProgress = 0
			m.rippingStatus = ""
			m.currentScreen = WelcomeScreen
			return m, nil
		}
		return m, nil
	}

	// Not ripping - normal navigation
	switch msg.String() {
	case "q", "esc":
		m.currentScreen = WelcomeScreen
		return m, nil
	case "enter", " ":
		// Simple rip start - no progress tracking for now
		fmt.Printf("DEBUG: Enter pressed - Drive: '%s', CDInfo: %v\n", m.config.Drives.CDDrive, m.cdInfo != nil)
		
		if m.config.Drives.CDDrive != "" && m.cdInfo != nil {
			fmt.Printf("DEBUG: Starting simple rip\n")
			m.isRipping = true
			m.rippingStatus = "Ripping in progress..."
			m.spinnerFrame = 0
			// Just start ripping without progress tracking
			go func() {
				fmt.Printf("DEBUG: Starting abcde command\n")
				fmt.Printf("DEBUG: Output dir: %s\n", m.config.Paths.Music)
				
				// Create output directory if it doesn't exist
				if err := os.MkdirAll(m.config.Paths.Music, 0755); err != nil {
					fmt.Printf("DEBUG: Failed to create output dir: %v\n", err)
					return
				}
				
				// Run abcde with proper output directory
				cmd := exec.Command("abcde", 
					"-d", m.config.Drives.CDDrive,
					"-o", m.config.CDRipping.OutputFormat,
					"-a", "default",  // Use default actions (cddb,read,encode,tag,move,clean)
				)
				
				// Set working directory to output location
				cmd.Dir = m.config.Paths.Music
				
				// Set environment variables for abcde
				cmd.Env = append(os.Environ(),
					"OUTPUTDIR="+m.config.Paths.Music,
					"OUTPUTFORMAT=${ARTISTFILE}/${ALBUMFILE}/${TRACKNUM}_${TRACKFILE}",
				)
				
				fmt.Printf("DEBUG: Running: %s in %s\n", cmd.String(), cmd.Dir)
				
				if err := cmd.Run(); err != nil {
					fmt.Printf("DEBUG: abcde failed: %v\n", err)
				} else {
					fmt.Printf("DEBUG: abcde completed successfully\n")
				}
			}()
			return m, spinnerCmd()
		} else {
			fmt.Printf("DEBUG: Cannot start rip - missing drive or CD info\n")
			m.rippingStatus = "Cannot start: No drive configured or CD not detected"
		}
	case "r":
		// Refresh CD info
		m.rippingStatus = "Detecting CD..."
		m.cdInfo = nil // Clear previous CD info
		return m, detectCDCmd(m.cdRipper)
	}
	return m, nil
}

func (m model) renderCDRipping() string {
	title := titleStyle.Render("💿 CD Ripping")
	
	if m.isRipping {
		// Show ripping progress with spinner
		subtitle := subtitleStyle.Render("Ripping in progress...")
		
		// Spinner animation
		spinnerFrames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		currentSpinner := spinnerFrames[m.spinnerFrame%len(spinnerFrames)]
		
		spinnerStyle := lipgloss.NewStyle().
			Foreground(accent).
			Bold(true).
			Margin(1, 2)
		
		spinnerDisplay := spinnerStyle.Render(fmt.Sprintf("%s %s", currentSpinner, m.rippingStatus))
		
		help := helpStyle.Render("Press 'q' or Esc to cancel ripping")
		
		content := fmt.Sprintf("%s\n%s\n\n%s\n\n%s",
			title,
			subtitle,
			spinnerDisplay,
			help,
		)
		
		return containerStyle.Render(content)
	}
	
	// Show CD info and ripping options
	subtitle := subtitleStyle.Render("Insert a CD and start ripping")
	
	// Current drive info
	currentDriveStyle := lipgloss.NewStyle().
		Foreground(lightBlue).
		Bold(true).
		Margin(1, 2)
	
	driveInfo := currentDriveStyle.Render(
		fmt.Sprintf("Drive: %s", m.config.Drives.CDDrive),
	)
	
	// CD detection status and info
	cdStatusStyle := lipgloss.NewStyle().
		Foreground(gray).
		Margin(0, 2, 1, 2)
	
	var cdStatus string
	var cdInfoDisplay string
	
	if m.config.Drives.CDDrive == "" {
		cdStatus = "❌ No drive configured - go to Settings > Drives"
	} else if m.cdInfo == nil {
		if m.rippingStatus == "" {
			cdStatus = "🔍 Press 'r' to detect CD"
		} else {
			cdStatus = m.rippingStatus
		}
	} else {
		cdStatus = "✅ CD detected"
		
		// Show CD information with constrained width
		cdInfoStyle := lipgloss.NewStyle().
			Foreground(lightBlue).
			Bold(true).
			Width(60).  // Constrain width to prevent overflow
			Margin(0, 2, 1, 2)
		
		var yearInfo string
		if m.cdInfo.Year != "" {
			yearInfo = fmt.Sprintf(" (%s)", m.cdInfo.Year)
		}
		
		// Truncate long text to prevent layout issues
		artist := m.cdInfo.Artist
		if len(artist) > 30 {
			artist = artist[:27] + "..."
		}
		
		album := m.cdInfo.Album
		if len(album) > 30 {
			album = album[:27] + "..."
		}
		
		// Show metadata source info
		var metadataSource string
		if m.cdInfo.Artist != "Unknown Artist" || m.cdInfo.Album != "Unknown Album" {
			metadataSource = fmt.Sprintf("Metadata: %s", m.config.CDRipping.CDDBMethod)
		} else {
			metadataSource = "Metadata: Basic disc info only"
		}
		
		cdInfoDisplay = cdInfoStyle.Render(fmt.Sprintf(
			"Artist: %s\nAlbum: %s%s\nTracks: %d\n%s",
			artist,
			album,
			yearInfo,
			m.cdInfo.TrackCount,
			metadataSource,
		))
	}
	
	cdStatusDisplay := cdStatusStyle.Render(cdStatus)
	
	// Ripping settings preview
	settingsStyle := lipgloss.NewStyle().
		Foreground(gray).
		Margin(1, 2)
	
	settingsInfo := settingsStyle.Render(fmt.Sprintf(
		"Format: %s • CDDB: %s • Output: %s",
		m.config.CDRipping.OutputFormat,
		m.config.CDRipping.CDDBMethod,
		m.config.Paths.Music,
	))
	
	// Action buttons (simulated)
	actionStyle := lipgloss.NewStyle().
		Foreground(accent).
		Bold(true).
		Background(lightBlue).
		Padding(0, 1).
		Margin(1, 2)
	
	var actionText string
	if m.config.Drives.CDDrive == "" {
		actionText = "Configure drive first"
	} else if m.cdInfo == nil {
		actionText = "Detect CD first (press 'r')"
	} else {
		actionText = "▶ Press Enter to start ripping"
	}
	
	action := actionStyle.Render(actionText)
	
	help := helpStyle.Render("Press 'r' to detect CD • Enter to start • Esc/q to go back")
	
	var content string
	if cdInfoDisplay != "" {
		content = fmt.Sprintf("%s\n%s\n\n%s\n%s\n%s\n\n%s\n\n%s",
			title,
			subtitle,
			driveInfo,
			cdStatusDisplay,
			cdInfoDisplay,
			action,
			help,
		)
	} else {
		content = fmt.Sprintf("%s\n%s\n\n%s\n%s\n%s\n\n%s\n\n%s",
			title,
			subtitle,
			driveInfo,
			cdStatusDisplay,
			settingsInfo,
			action,
			help,
		)
	}
	
	return containerStyle.Render(content)
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v", err)
		os.Exit(1)
	}
}

