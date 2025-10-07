package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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

	var events []Event
	for _, event := range cal.Events() {
		title := ""
		if prop := event.GetProperty(ics.ComponentPropertySummary); prop != nil {
			title = prop.Value
		}

		desc := ""
		if prop := event.GetProperty(ics.ComponentPropertyDescription); prop != nil {
			desc = prop.Value
		}

		start, err := event.GetStartAt()
		if err != nil {
			start = time.Now()
		}

		events = append(events, Event{
			title:       title,
			start:       start,
			description: desc,
		})
	}

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
					events, err := fetchEvents(m.config.OriginURL)
					if err != nil {
						m.message = fmt.Sprintf("Error fetching events: %v", err)
					} else {
						m.events = events
						m.updateEventsList()
						m.state = "events"
						m.eventsList.SetShowHelp(true)
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
	var s strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginBottom(1)

	menuStyle := lipgloss.NewStyle().
		MarginLeft(2).
		MarginBottom(1)

	s.WriteString(titleStyle.Render("🗓️  uniCal Configuration"))
	s.WriteString("\n")

	if m.message != "" {
		msgStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			MarginBottom(1)
		s.WriteString(msgStyle.Render("ℹ️  " + m.message))
		s.WriteString("\n")
	}

	switch m.state {
	case "menu":
		s.WriteString(menuStyle.Render("1. Set iCal URL"))
		if m.config.OriginURL != "" {
			s.WriteString(menuStyle.Render(fmt.Sprintf("   Current: %s", m.config.OriginURL)))
		}
		s.WriteString("\n")
		s.WriteString(menuStyle.Render("2. Manage Events & Blocklist"))
		s.WriteString("\n")
		s.WriteString(menuStyle.Render("3. Save Configuration"))
		s.WriteString("\n")
		s.WriteString(menuStyle.Render("q. Quit"))
		s.WriteString("\n\n")

		if len(m.config.Blocklist) > 0 {
			s.WriteString(menuStyle.Render("Current Blocklist:"))
			for _, item := range m.config.Blocklist {
				s.WriteString(menuStyle.Render(fmt.Sprintf("  • %s", item)))
			}
		}

	case "url_input":
		s.WriteString("Enter iCal URL (ESC to cancel):\n\n")
		s.WriteString(m.urlInput.View())

	case "events":
		s.WriteString("Events (ENTER to toggle blocklist, N for notes, ESC to return):\n\n")
		s.WriteString(m.eventsList.View())

	case "notes_input":
		s.WriteString("Add notes for event (ENTER to save, ESC to cancel):\n\n")
		s.WriteString(m.notesInput.View())
	}

	return s.String()
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())

	if err := p.Start(); err != nil {
		log.Fatal(err)
	}
}
