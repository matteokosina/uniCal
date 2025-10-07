package main

import (
	"fmt"
	"log"
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type Config struct {
	OriginURL string   `yaml:"origin_url"`
	blocklist []string `yaml:"blocklist"`
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

func filterEvents(cal *ics.Calendar, blocklist []string) *ics.Calendar {
func filterEvents(cal *ics.Calendar, blocklist []string) *ics.Calendar {
	filteredCal := ics.NewCalendar()
	for _, event := range cal.Events() {
		blocklisted := false
		for _, title := range blocklist {
		blocklisted := false
		for _, title := range blocklist {
			if prop := event.GetProperty(ics.ComponentPropertySummary); prop != nil && prop.Value == title {
				blocklisted = true
				blocklisted = true
				break
			}
		}
		if !blocklisted {
		if !blocklisted {
			filteredCal.AddVEvent(event)
		}
// Initialize Viper and load configuration
func initConfig() error {
	// Set default values for configuration reading
	viper.SetConfigName("blocklist")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("config")

	// Read the config file
	if err := viper.ReadInConfig(); err != nil {
		log.Fatal("Failed to read config:", err)
	}

	// Enable reading of config file on change (dev purpose)
	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("Config file changed:", e.Name)
	})
	viper.WatchConfig()

	return nil
}

func main() {
	config, err := loadConfig("config/blocklist.yaml")
	config, err := loadConfig("config/blocklist.yaml")
	err := initConfig()
	if err != nil {
		log.Fatal("Failed to load config via viper:", err)
	}

	rapla_url := viper.GetViper().GetString("origin_url")
	if rapla_url == "" {
		log.Fatal("Origin URL is not set in the config file")
	}
	rapla, err := FetchNewRaplaInstance(rapla_url)
	if err != nil {
		log.Fatal("Failed to fetch iCal:", err)
	}

	filteredCal := filterEvents(cal, config.blocklist)
	blocklist := viper.GetViper().GetStringSlice("blocklist")
	if len(blocklist) == 0 {
		log.Println("Warning: Blocklist is empty, no events will be filtered")
	}
	rapla.filterEvents(blocklist)

	outputDir := "ical"
	outputFile := outputDir + "/filtered_calendar.ics"
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		log.Fatal("Failed to create ical directory:", err)
	}

	if err := rapla.saveFilteredICal(outputFile); err != nil {
		log.Fatal("Failed to save filtered iCal:", err)
	}

	log.Println("Filtered iCal saved to:", outputFile)
}

