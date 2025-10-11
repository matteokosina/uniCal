package app

import (
	"log"
	"net/http"
	"os"
	"strings"

	ics "github.com/arran4/golang-ical"
)

type Rapla struct {
	cal *ics.Calendar
}

// Creating a new Rapla instance based on a provided URL
func FetchNewRaplaInstance(url string) (*Rapla, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	cal, err := ics.ParseCalendar(resp.Body)
	if err != nil {
		return nil, err
	}
	return &Rapla{cal: cal}, nil
}

// Save the filtered calendar to a file
func (rapla *Rapla) SaveFilteredICal(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(rapla.cal.Serialize())
	if err != nil {
		return err
	}

	return nil
}

// Functions that operate on the calendar

// Filter events based on provided blocklist
func (rapla *Rapla) FilterEvents(blocklist []string, notes map[string]string) {
	// Create a new calendar and copy relevant properties from the original
	filteredCal := ics.NewCalendar()
	notesAdded := 0 // Counter for notes added
	for _, event := range rapla.cal.Events() {
		blocklisted := false
		for _, title := range blocklist {
			// Check if the event's summary matches any title in the blocklist
			if prop := event.GetProperty(ics.ComponentPropertySummary); prop != nil && prop.Value == title {
				blocklisted = true
				break
			}
		}
		if !blocklisted {
			// Add notes if applicable
			event, updated := rapla.addNotesToEvent(event, notes)
			if updated {
				notesAdded++
			}
			// Add event to the new calendar
			filteredCal.AddVEvent(event)
		}
	}
	if notesAdded > 0 {
		log.Printf("Added notes to %d events", notesAdded)
	}
	rapla.cal = filteredCal
}

func (rapla *Rapla) addNotesToEvent(event *ics.VEvent, notes map[string]string) (*ics.VEvent, bool) {
	eventTitle := event.GetProperty(ics.ComponentPropertySummary).Value
	if eventTitle != "" {
		if note, exists := notes[strings.ToLower(eventTitle)]; exists {
			// Get existing description
			existingDescription := ""
			if descProp := event.GetProperty(ics.ComponentPropertyDescription); descProp != nil {
				existingDescription = descProp.Value
			}
			// Append the note to the existing description (using proper line breaks)
			newDescription := existingDescription
			if existingDescription != "" {
				newDescription += "\\n\\n--- Notes ---\\n"
			} else {
				newDescription = "--- Notes ---\\n"
			}
			newDescription += note
			// Update the event's description property
			event.RemoveProperty(ics.ComponentPropertyDescription)
			event.SetProperty(ics.ComponentPropertyDescription, newDescription)
			return event, true
		}
	}
	return event, false
}
