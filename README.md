# uniCal 📆📦

Ever wanted to subscribe to an iCal calendar but only need specific events, such as a university schedule where you don't attend every course?  
**uniCal** is a Go application that downloads an iCal file from a given URL, filters out unwanted events using a blocklist, and creates a new, customized iCal file.

## Features

- Downloads iCal data from a specified URL.
- Filters events by titles listed in a `blocklist.yaml` file.
- Saves the filtered calendar to an `ical` directory.
- **Terminal UI** for easy configuration and event management.
- Add custom notes to events.
- Interactive blocklist management with search functionality.

## Installation

```bash
git clone https://github.com/matteokosina/uniCal.git
cd uniCal
make install-deps
```

## Usage

### Configuration UI (Recommended)

Launch the interactive terminal UI to configure your calendar:

```bash
make config
# or directly: go run ./cmd/unical-config/main.go
```

The TUI provides:

1. **Set iCal URL** - Configure the source calendar URL
2. **Manage Events & Blocklist** - View events, toggle blocklist status, and add notes
   - Press `ENTER` to toggle an event in/out of the blocklist
   - Press `N` to add or edit notes for an event
   - Events marked with ✗ are blocked, ✓ are allowed
3. **Save Configuration** - Save your settings to `config/blocklist.yaml`

### Command Line Usage

Run the calendar filter directly:

```bash
make run
# or directly: go run ./cmd/unical/main.go
```

### Manual Configuration

Update `config/blocklist.yaml` with your iCal URL and the event titles to exclude. Example:

```yaml
origin_url: "https://example.com/calendar.ics"
blocklist:
  - "Event Title 0"
  - "Event Title 1"
notes:
  "Important Meeting": "Don't forget to bring laptop"
```

## Output

The filtered calendar will be saved as `filtered_calendar.ics` in the `ical` directory. Use the Github raw URL of that file and add it to your calendar.
Use this URL and adjust it to your forked repo:

`https://raw.githubusercontent.com/<gh-username>/uniCal/calendar-updates/ical/filtered_calendar.ics`

## Keeping your Calendar Up-To-Date

The Github Action defined under `.github/workflows` refreshes and syncs with the upstream calendar. It can be run on schedule via a `Cron-Job`. Adjust this to your desired refresh-rate.

## License

Licensed under the MIT License. See the LICENSE file for details.
