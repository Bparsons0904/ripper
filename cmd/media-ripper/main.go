package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

type model struct {
	ready bool
}

func initialModel() model {
	return model{
		ready: true,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
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

	// Status section
	status := statusStyle.Render("🎉 Status: Initial setup complete!")

	// Help section
	help := helpStyle.Render("Press 'q' or Ctrl+C to quit")

	// Combine all content
	content := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s",
		title,
		subtitle,
		welcome,
		description,
		featuresHeader,
		featureList,
		status,
		help,
	)

	// Wrap everything in the container with blue border
	return containerStyle.Render(content)
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v", err)
		os.Exit(1)
	}
}

