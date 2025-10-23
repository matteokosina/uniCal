package configer

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
)

type TUIConfig struct {
	OriginURL string            `yaml:"origin_url"`
	Blocklist []string          `yaml:"blocklist"`
	Notes     map[string]string `yaml:"notes,omitempty"`
}

type Event struct {
	title       string
	start       time.Time
	description string
	blocked     bool
	notes       string
}

func (e Event) FilterValue() string { return e.title }
func (e Event) Title() string       { return e.title }
func (e Event) Description() string {
	status := "✓"
	if e.blocked {
		status = "✗"
	}
	startStr := e.start.Format("2006-01-02 15:04")
	return fmt.Sprintf("%s %s - %s", status, startStr, e.description)
}

type model struct {
	state       string // "menu", "url_input", "events", "notes_input"
	urlInput    textinput.Model
	eventsList  list.Model
	notesInput  textinput.Model
	config      *TUIConfig
	events      []Event
	selectedIdx int
	message     string
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Enter iCal URL..."
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 80

	ni := textinput.New()
	ni.Placeholder = "Enter notes for this event..."
	ni.CharLimit = 512
	ni.Width = 80

	config := &TUIConfig{
		Blocklist: []string{},
		Notes:     make(map[string]string),
	}

	// Try to load existing config
	if existingConfig, err := loadTUIConfig("config/blocklist.yaml"); err == nil {
		config = existingConfig
		ti.SetValue(config.OriginURL)
	}

	return model{
		state:      "menu",
		urlInput:   ti,
		notesInput: ni,
		config:     config,
		eventsList: list.New([]list.Item{}, list.NewDefaultDelegate(), 80, 20),
	}
}

func loadTUIConfig(path string) (*TUIConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var config TUIConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	if config.Notes == nil {
		config.Notes = make(map[string]string)
	}
	return &config, nil
}

func saveTUIConfig(config *TUIConfig) error {
	if err := os.MkdirAll("config", os.ModePerm); err != nil {
		return err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile("config/blocklist.yaml", data, 0644)
}

func fetchEvents(url string) ([]Event, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	cal, err := ics.ParseCalendar(resp.Body)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	eventMap := make(map[string]Event) // Use map to ensure distinct events by title

	for _, event := range cal.Events() {
		title := ""
		if prop := event.GetProperty(ics.ComponentPropertySummary); prop != nil {
			title = prop.Value
		}

		// Skip events with empty titles
		if title == "" {
			continue
		}

		desc := ""
		if prop := event.GetProperty(ics.ComponentPropertyDescription); prop != nil {
			desc = prop.Value
		}

		start, err := event.GetStartAt()
		if err != nil {
			continue // Skip events with invalid start times
		}

		// Only include events that are today or in the future
		if start.Before(today) {
			continue
		}

		// For recurring events, keep the one with the earliest future date
		if existingEvent, exists := eventMap[title]; exists {
			if start.Before(existingEvent.start) {
				eventMap[title] = Event{
					title:       title,
					start:       start,
					description: desc,
				}
			}
		} else {
			eventMap[title] = Event{
				title:       title,
				start:       start,
				description: desc,
			}
		}
	}

	// Convert map to slice
	var events []Event
	for _, event := range eventMap {
		events = append(events, event)
	}

	// Sort events by start time (earliest first)
	sort.Slice(events, func(i, j int) bool {
		return events[i].start.Before(events[j].start)
	})

	return events, nil
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case "menu":
			switch msg.String() {
			case "1":
				m.state = "url_input"
				m.urlInput.Focus()
			case "2":
				if m.config.OriginURL != "" {
					m.message = "Fetching events from calendar..."
					events, err := fetchEvents(m.config.OriginURL)
					if err != nil {
						m.message = fmt.Sprintf("Error fetching events: %v", err)
					} else {
						m.events = events
						m.updateEventsList()
						m.state = "events"
						m.eventsList.SetShowHelp(true)
						m.message = fmt.Sprintf("Loaded %d unique upcoming events", len(events))
					}
				} else {
					m.message = "Please set URL first"
				}
			case "3":
				if err := saveTUIConfig(m.config); err != nil {
					m.message = fmt.Sprintf("Error saving config: %v", err)
				} else {
					m.message = "Configuration saved!"
				}
			case "q", "ctrl+c":
				return m, tea.Quit
			}

		case "url_input":
			switch msg.String() {
			case "enter":
				m.config.OriginURL = m.urlInput.Value()
				m.state = "menu"
				m.message = "URL updated!"
			case "esc":
				m.state = "menu"
			default:
				m.urlInput, cmd = m.urlInput.Update(msg)
			}

		case "events":
			switch msg.String() {
			case "enter":
				if len(m.events) > 0 {
					selected := m.eventsList.SelectedItem().(Event)
					m.selectedIdx = m.eventsList.Index()

					// Check if event is already in blocklist
					isBlocked := false
					for _, blocked := range m.config.Blocklist {
						if blocked == selected.title {
							isBlocked = true
							break
						}
					}

					if !isBlocked {
						m.config.Blocklist = append(m.config.Blocklist, selected.title)
						m.events[m.selectedIdx].blocked = true
					} else {
						// Remove from blocklist
						newBlocklist := []string{}
						for _, blocked := range m.config.Blocklist {
							if blocked != selected.title {
								newBlocklist = append(newBlocklist, blocked)
							}
						}
						m.config.Blocklist = newBlocklist
						m.events[m.selectedIdx].blocked = false
					}
					m.updateEventsList()
				}
			case "n":
				if len(m.events) > 0 {
					selected := m.eventsList.SelectedItem().(Event)
					m.selectedIdx = m.eventsList.Index()
					m.notesInput.SetValue(m.config.Notes[selected.title])
					m.notesInput.Focus()
					m.state = "notes_input"
				}
			case "esc":
				m.state = "menu"
			default:
				m.eventsList, cmd = m.eventsList.Update(msg)
			}

		case "notes_input":
			switch msg.String() {
			case "enter":
				if len(m.events) > 0 {
					selected := m.events[m.selectedIdx]
					notes := m.notesInput.Value()
					if notes == "" {
						delete(m.config.Notes, selected.title)
					} else {
						m.config.Notes[selected.title] = notes
					}
					m.events[m.selectedIdx].notes = notes
					m.updateEventsList()
				}
				m.state = "events"
			case "esc":
				m.state = "events"
			default:
				m.notesInput, cmd = m.notesInput.Update(msg)
			}
		}
	}

	return m, cmd
}

func (m *model) updateEventsList() {
	items := make([]list.Item, len(m.events))
	for i, event := range m.events {
		// Check if event is blocked
		for _, blocked := range m.config.Blocklist {
			if blocked == event.title {
				event.blocked = true
				break
			}
		}
		// Add notes if available
		if notes, exists := m.config.Notes[event.title]; exists {
			event.notes = notes
		}
		items[i] = event
	}
	m.eventsList.SetItems(items)
}

func (m model) View() string {
	var sections []string

	// Compact design for standard terminals (80 chars)
	terminalWidth := 77

	// Compact side-by-side layout
	leftWidth := 75
	gap := 2

	// Compact UNICAL logo
	logoStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226")).
		Background(lipgloss.Color("233")).
		Padding(1).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("226")).
		Width(leftWidth).
		Height(9).
		Align(lipgloss.Center)

	// Compact ASCII logo
	asciiLogo := `██   ██ ███  ██ ██  ██████ ███████ ██
██   ██ ████ ██ ██ ██      ██   ██ ██
██   ██ ██ ████ ██ ██      ███████ ██
██   ██ ██  ███ ██ ██      ██   ██ ██
      █████  ██   ██ ██  ██████ ██   ██ ███████`

	// Create the boxes
	logoBox := logoStyle.Render(asciiLogo)

	// Create spacer
	spacer := strings.Repeat(" ", gap)

	// Join horizontally with proper alignment
	headerSection := lipgloss.JoinHorizontal(lipgloss.Top, logoBox, spacer)

	sections = append(sections, headerSection)

	// Message display
	if m.message != "" {
		msgStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			Background(lipgloss.Color("235")).
			Padding(0, 2).
			Margin(0, 0, 1, 0).
			BorderLeft(true).
			BorderForeground(lipgloss.Color("82"))

		sections = append(sections, msgStyle.Render("ℹ️  "+m.message))
	}

	// Main content based on state
	switch m.state {
	case "menu":
		// Compact menu
		menuStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("235")).
			Padding(1, 2).
			Margin(0, 0, 1, 0).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Width(terminalWidth - 2)

		highlightStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)

		urlStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Italic(true)

		var menuContent strings.Builder
		menuContent.WriteString("📋 Main Menu\n\n")
		menuContent.WriteString(fmt.Sprintf("%s\n", highlightStyle.Render("1. Set iCal URL")))

		if m.config.OriginURL != "" {
			displayURL := m.config.OriginURL
			if len(displayURL) > 50 {
				displayURL = displayURL[:47] + "..."
			}
			menuContent.WriteString(fmt.Sprintf("   %s\n", urlStyle.Render("Current: "+displayURL)))
		} else {
			menuContent.WriteString("   (No URL configured)\n")
		}

		menuContent.WriteString("\n")
		menuContent.WriteString(fmt.Sprintf("%s\n", highlightStyle.Render("2. Manage Events & Blocklist")))
		menuContent.WriteString("\n")
		menuContent.WriteString(fmt.Sprintf("%s\n", highlightStyle.Render("3. Save Configuration")))
		menuContent.WriteString("\n")
		menuContent.WriteString(fmt.Sprintf("%s\n", highlightStyle.Render("q. Quit")))

		sections = append(sections, menuStyle.Render(menuContent.String()))

		// Compact status boxes
		if len(m.config.Blocklist) > 0 || len(m.config.Notes) > 0 {
			var statusBoxes []string

			if len(m.config.Blocklist) > 0 {
				blocklistStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("196")).
					Background(lipgloss.Color("235")).
					Border(lipgloss.RoundedBorder()).
					BorderForeground(lipgloss.Color("196")).
					Padding(1).
					Width(35).
					Height(6)

				var blocklistContent strings.Builder
				blocklistContent.WriteString("🚫 Blocked Events\n")
				for i, item := range m.config.Blocklist {
					if i >= 3 {
						blocklistContent.WriteString(fmt.Sprintf("   ... +%d more", len(m.config.Blocklist)-3))
						break
					}
					if len(item) > 25 {
						item = item[:22] + "..."
					}
					blocklistContent.WriteString(fmt.Sprintf("   • %s\n", item))
				}

				statusBoxes = append(statusBoxes, blocklistStyle.Render(blocklistContent.String()))
			}

			if len(m.config.Notes) > 0 {
				notesStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("226")).
					Background(lipgloss.Color("235")).
					Border(lipgloss.RoundedBorder()).
					BorderForeground(lipgloss.Color("226")).
					Padding(1).
					Width(35).
					Height(6)

				var notesContent strings.Builder
				notesContent.WriteString("📝 Events with Notes\n")
				count := 0
				for title, note := range m.config.Notes {
					if count >= 2 {
						notesContent.WriteString(fmt.Sprintf("   ... +%d more", len(m.config.Notes)-2))
						break
					}
					if len(title) > 25 {
						title = title[:22] + "..."
					}
					if len(note) > 20 {
						note = note[:17] + "..."
					}
					notesContent.WriteString(fmt.Sprintf("   • %s\n     %s\n", title, note))
					count++
				}

				statusBoxes = append(statusBoxes, notesStyle.Render(notesContent.String()))
			}

			// Join status boxes horizontally if both exist
			if len(statusBoxes) > 1 {
				statusSection := lipgloss.JoinHorizontal(lipgloss.Top, statusBoxes[0], strings.Repeat(" ", 2), statusBoxes[1])
				sections = append(sections, statusSection)
			} else if len(statusBoxes) == 1 {
				sections = append(sections, statusBoxes[0])
			}
		}

	case "url_input":
		inputStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("235")).
			Padding(1, 2).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("39")).
			Width(terminalWidth - 2)

		content := "🔗 Enter iCal URL (ESC to cancel):\n\n" + m.urlInput.View()
		sections = append(sections, inputStyle.Render(content))

	case "events":
		eventsStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("235")).
			Padding(1, 2).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("86")).
			Width(terminalWidth - 2)

		var eventsContent strings.Builder
		eventsContent.WriteString("📅 Events Management\n\n")
		eventsContent.WriteString("Controls:\n")
		eventsContent.WriteString("  • ENTER: Toggle blocklist (✓=allowed, ✗=blocked)\n")
		eventsContent.WriteString("  • N: Add/edit notes\n")
		eventsContent.WriteString("  • ↑/↓: Navigate • ESC: Save\n\n")
		eventsContent.WriteString(m.eventsList.View())

		sections = append(sections, eventsStyle.Render(eventsContent.String()))

	case "notes_input":
		notesStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("235")).
			Padding(1, 2).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("226")).
			Width(terminalWidth - 2)

		content := "📝 Add notes (ENTER to save, ESC to cancel):\n\n" + m.notesInput.View()
		sections = append(sections, notesStyle.Render(content))
	}

	return strings.Join(sections, "\n")
}

func InitializeAndRun() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
