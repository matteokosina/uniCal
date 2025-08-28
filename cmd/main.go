package main

import (
	"io"
	"log"
	"net/http"
	"os"

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
		log.Fatal("Failed to create ical directory:", err)
	}

	if err := saveFilteredICal(filteredCal, outputFile); err != nil {
		log.Fatal("Failed to save filtered iCal:", err)
	}

	log.Println("Filtered iCal saved to:", outputFile)
}
