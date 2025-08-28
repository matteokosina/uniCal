package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"time"

	"gopkg.in/yaml.v3"

	ics "github.com/arran4/golang-ical"
)

type Config struct {
	OriginURL string   `yaml:"origin_url"`
	Blacklist []string `yaml:"blacklist"`
}

func loadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func fetchICal(url string) (*ics.Calendar, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	cal, err := ics.ParseCalendar(resp.Body)
	if err != nil {
		return nil, err
	}
	return cal, nil
}

func filterEvents(cal *ics.Calendar, blacklist []string) *ics.Calendar {
	filteredCal := ics.NewCalendar()
	for _, event := range cal.Events() {
		blacklisted := false
		for _, title := range blacklist {
			if prop := event.GetProperty(ics.ComponentPropertySummary); prop != nil && prop.Value == title {
				blacklisted = true
				break
			}
		}
		if !blacklisted {
			filteredCal.AddVEvent(event)
		}
	}
	return filteredCal
}

func printFirstWeekEvents(cal *ics.Calendar) {
	now := time.Now()
	weekLater := now.AddDate(0, 0, 7)
	events := []struct {
		Title string
		Start time.Time
	}{}

	log.Println("Collecting events within the next week...")
	for _, event := range cal.Events() {
		start, err := event.GetStartAt()
		if err != nil {
			log.Println("Error parsing event start time:", err)
			continue
		}
		if start.After(now) && start.Before(weekLater) {
			events = append(events, struct {
				Title string
				Start time.Time
			}{
				Title: event.GetProperty(ics.ComponentPropertySummary).Value,
				Start: start,
			})
			log.Printf("Event collected: %s at %s\n", event.GetProperty(ics.ComponentPropertySummary).Value, start)
		}
	}

	log.Printf("Total events collected: %d\n", len(events))
	sort.Slice(events, func(i, j int) bool {
		return events[i].Start.Before(events[j].Start)
	})

	for _, e := range events {
		fmt.Println("Event:", e.Title)
		fmt.Println("Start:", e.Start)
	}
}

func saveFilteredICal(cal *ics.Calendar, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(cal.Serialize())
	if err != nil {
		return err
	}

	return nil
}

func main() {
	config, err := loadConfig("config/blacklist.yaml")
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	cal, err := fetchICal(config.OriginURL)
	if err != nil {
		log.Fatal("Failed to fetch iCal:", err)
	}

	filteredCal := filterEvents(cal, config.Blacklist)

	outputDir := "ical"
	outputFile := outputDir + "/filtered_calendar.ics"
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		log.Fatal("Failed to create output directory:", err)
	}

	if err := saveFilteredICal(filteredCal, outputFile); err != nil {
		log.Fatal("Failed to save filtered iCal:", err)
	}

	log.Println("Filtered iCal saved to:", outputFile)
	// printFirstWeekEvents(filteredCal)
}
