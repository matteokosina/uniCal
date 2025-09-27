package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

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
	err := initConfig()
	if err != nil {
		log.Fatal("Failed to load config via viper:", err)
	}

	rapla, err := NewRaplaUrl(viper.GetViper().GetString("origin_url"))
	if err != nil {
		log.Fatal("Failed to fetch iCal:", err)
	}

	// Check if command line arguments are provided for timeframe
	if len(os.Args) >= 3 {
		// Parse start and end dates from command line arguments
		startStr := os.Args[1]
		endStr := os.Args[2]

		startTime, err := time.Parse("2006-01-02", startStr)
		if err != nil {
			log.Fatal("Failed to parse start date (use YYYY-MM-DD format):", err)
		}

		endTime, err := time.Parse("2006-01-02", endStr)
		if err != nil {
			log.Fatal("Failed to parse end date (use YYYY-MM-DD format):", err)
		}

		// Get unique event names in specified timeframe
		eventNames := rapla.getEventsInTimespan(startTime, endTime)
		log.Printf("Found %d unique events in timeframe %s to %s", len(eventNames), startStr, endStr)

		fmt.Println("Unique event names in timeframe:")
		for _, name := range eventNames {
			fmt.Printf("- %s\n", name)
		}
		return // Exit early when showing timeframe events
	} else {
		// No timeframe specified, apply filtering
		rapla.filterEvents(viper.GetViper().GetStringSlice("blocklist"))
		log.Println("Applied blocklist filtering")
	}

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
