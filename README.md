# uniCal 📆📦

Ever wanted to subscribe to an iCal calendar but only need specific events, such as a university schedule where you don't attend every course?  
**uniCal** is a Go application that downloads an iCal file from a given URL, filters out unwanted events using a blacklist, and creates a new, customized iCal file.

## Features

- Downloads iCal data from a specified URL.
- Filters events by titles listed in a `blacklist.yaml` file.
- Saves the filtered calendar to an `ical` directory.

## Configuration

Update `config/blacklist.yaml` with your iCal URL and the event titles to exclude. Example:

```yaml
origin_url: "https://example.com/calendar.ics"
blacklist:
  - "Event Title 1"
  - "Event Title 2"
```

The filtered calendar will be saved as `filtered_calendar.ics` in the `ical` directory. Use the Github raw URL of that file and add it to your calendar.
Use this URL and adjust it to your forked repo:

`https://raw.githubusercontent.com/<gh-username>/uniCal/main/ical/filtered_calendar.ics`

## License

Licensed under the MIT License. See the LICENSE file for details.
