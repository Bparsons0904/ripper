package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true).
			Margin(1, 0)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Margin(0, 0, 1, 0)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Margin(1, 0)
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
		return "Loading..."
	}

	title := titleStyle.Render("🎵 Media Ripper TUI")
	subtitle := subtitleStyle.Render("A Terminal UI for ripping CDs, DVDs, and Blu-rays")

	content := `
Welcome to Media Ripper!

This application will help you rip your media collection with an intuitive
terminal interface. Built with Go and powered by the Charm library.

Features coming soon:
• Drive detection and selection
• Audio CD ripping with abcde
• DVD/Blu-ray ripping with MakeMKV  
• Real-time progress tracking
• Configurable settings
• Container support

Status: Initial setup complete! 🎉
`

	help := helpStyle.Render("Press 'q' or Ctrl+C to quit")

	return fmt.Sprintf("%s\n%s\n%s\n%s", title, subtitle, content, help)
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v", err)
		os.Exit(1)
	}
}

