package main

import (
	"log"
	"os"
	app "uniCal/cmd/app"
	configer "uniCal/cmd/configer"

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
		log.Println("Config file changed:", e.Name)
	})
	viper.WatchConfig()

	return nil
}

func updateRoutine() {
	err := initConfig()
	if err != nil {
		log.Fatal("Failed to load config via viper:", err)
	}

	rapla_url := viper.GetViper().GetString("origin_url")
	if rapla_url == "" {
		log.Fatal("Origin URL is not set in the config file")
	}
	rapla, err := app.FetchNewRaplaInstance(rapla_url)
	if err != nil {
		log.Fatal("Failed to fetch iCal:", err)
	}

	// Read blocklist from config
	blocklist := viper.GetViper().GetStringSlice("blocklist")
	if len(blocklist) == 0 {
		log.Println("Warning: Blocklist is empty, no events will be filtered")
	}

	// Read notes from config
	notes := viper.GetViper().GetStringMapString("notes")
	if len(notes) == 0 {
		log.Println("Warning: Notes is empty, no notes will be added")
	}
	// Note: Notes are read in lower case

	rapla.FilterEvents(blocklist, notes)

	outputDir := "ical"
	outputFile := outputDir + "/filtered_calendar.ics"
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		log.Fatal("Failed to create ical directory:", err)
	}

	if err := rapla.SaveFilteredICal(outputFile); err != nil {
		log.Fatal("Failed to save filtered iCal:", err)
	}

	log.Println("Filtered iCal saved to:", outputFile)
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "config" {
		configer.InitializeAndRun()
	} else {
		updateRoutine()
	}
}
