package main

import (
	"fmt"
	"log"
	"os"

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

	rapla_url := viper.GetViper().GetString("origin_url")
	if rapla_url == "" {
		log.Fatal("Origin URL is not set in the config file")
	}
	rapla, err := FetchNewRaplaInstance(rapla_url)
	if err != nil {
		log.Fatal("Failed to fetch iCal:", err)
	}

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
