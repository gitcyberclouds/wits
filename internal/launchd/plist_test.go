//go:build darwin

package launchd

import (
	"reflect"
	"testing"
)

func TestParsePlistXML(t *testing.T) {
	t.Parallel()

	const data = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.example.agent</string>
	<key>Comment</key>
	<string>Example agent</string>
	<key>Program</key>
	<string>/usr/local/bin/agent</string>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>StartInterval</key>
	<integer>300</integer>
	<key>ProgramArguments</key>
	<array>
		<string>/usr/local/bin/agent</string>
		<string>--serve</string>
	</array>
	<key>WatchPaths</key>
	<array>
		<string>/etc/agent.conf</string>
	</array>
	<key>QueueDirectories</key>
	<array>
		<string>/var/spool/agent</string>
	</array>
</dict>
</plist>`

	info := &LaunchdInfo{}
	if err := parsePlistXML([]byte(data), info); err != nil {
		t.Fatalf("parsePlistXML returned error: %v", err)
	}

	if info.Label != "com.example.agent" {
		t.Errorf("Label = %q, want %q", info.Label, "com.example.agent")
	}
	if info.Comment != "Example agent" {
		t.Errorf("Comment = %q, want %q", info.Comment, "Example agent")
	}
	if info.Program != "/usr/local/bin/agent" {
		t.Errorf("Program = %q, want %q", info.Program, "/usr/local/bin/agent")
	}
	if !info.RunAtLoad {
		t.Error("RunAtLoad = false, want true")
	}
	if !info.KeepAlive {
		t.Error("KeepAlive = false, want true")
	}
	if info.StartInterval != 300 {
		t.Errorf("StartInterval = %d, want 300", info.StartInterval)
	}
	if want := []string{"/usr/local/bin/agent", "--serve"}; !reflect.DeepEqual(info.ProgramArguments, want) {
		t.Errorf("ProgramArguments = %v, want %v", info.ProgramArguments, want)
	}
	if want := []string{"/etc/agent.conf"}; !reflect.DeepEqual(info.WatchPaths, want) {
		t.Errorf("WatchPaths = %v, want %v", info.WatchPaths, want)
	}
	if want := []string{"/var/spool/agent"}; !reflect.DeepEqual(info.QueueDirectories, want) {
		t.Errorf("QueueDirectories = %v, want %v", info.QueueDirectories, want)
	}
}

func TestParsePlistXMLStartCalendarIntervalDict(t *testing.T) {
	t.Parallel()

	const data = `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.example.cron</string>
	<key>StartCalendarInterval</key>
	<dict>
		<key>Weekday</key>
		<integer>1</integer>
		<key>Hour</key>
		<integer>9</integer>
		<key>Minute</key>
		<integer>30</integer>
	</dict>
</dict>
</plist>`

	info := &LaunchdInfo{}
	if err := parsePlistXML([]byte(data), info); err != nil {
		t.Fatalf("parsePlistXML returned error: %v", err)
	}
	if info.Label != "com.example.cron" {
		t.Errorf("Label = %q, want %q", info.Label, "com.example.cron")
	}
	if want := "Mon at 09:30"; info.StartCalendarInterval != want {
		t.Errorf("StartCalendarInterval = %q, want %q", info.StartCalendarInterval, want)
	}
}

func TestParsePlistXMLStartCalendarIntervalArray(t *testing.T) {
	t.Parallel()

	const data = `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.example.multi</string>
	<key>StartCalendarInterval</key>
	<array>
		<dict>
			<key>Hour</key>
			<integer>8</integer>
		</dict>
		<dict>
			<key>Hour</key>
			<integer>20</integer>
		</dict>
	</array>
</dict>
</plist>`

	info := &LaunchdInfo{}
	if err := parsePlistXML([]byte(data), info); err != nil {
		t.Fatalf("parsePlistXML returned error: %v", err)
	}
	if want := "at 08:00; at 20:00"; info.StartCalendarInterval != want {
		t.Errorf("StartCalendarInterval = %q, want %q", info.StartCalendarInterval, want)
	}
}

func TestFormatCalendarInterval(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cal  map[string]int
		want string
	}{
		{"empty", map[string]int{}, ""},
		{"hour and minute", map[string]int{"Hour": 9, "Minute": 30}, "at 09:30"},
		{"hour only", map[string]int{"Hour": 14}, "at 14:00"},
		{"minute only", map[string]int{"Minute": 5}, "at *:05"},
		{"weekday only", map[string]int{"Weekday": 1}, "Mon"},
		{"weekday with time", map[string]int{"Weekday": 0, "Hour": 8, "Minute": 0}, "Sun at 08:00"},
		{"month and day", map[string]int{"Month": 6, "Day": 15}, "month 6 day 15"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := formatCalendarInterval(tc.cal); got != tc.want {
				t.Errorf("formatCalendarInterval(%v) = %q, want %q", tc.cal, got, tc.want)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		seconds int
		want    string
	}{
		{30, "30s"},
		{90, "1m"},
		{300, "5m"},
		{3600, "1h"},
		{7200, "2h"},
		{86400, "1d"},
	}

	for _, tc := range tests {
		if got := formatDuration(tc.seconds); got != tc.want {
			t.Errorf("formatDuration(%d) = %q, want %q", tc.seconds, got, tc.want)
		}
	}
}

func TestFormatTriggers(t *testing.T) {
	t.Parallel()

	info := &LaunchdInfo{
		RunAtLoad:             true,
		StartInterval:         300,
		StartCalendarInterval: "Mon at 09:30",
		WatchPaths:            []string{"/etc/foo"},
		QueueDirectories:      []string{"/var/q"},
	}
	want := []string{
		"RunAtLoad (starts at login/boot)",
		"StartInterval (every 5m)",
		"StartCalendarInterval (Mon at 09:30)",
		"WatchPaths: /etc/foo",
		"QueueDirectories: /var/q",
	}
	if got := info.FormatTriggers(); !reflect.DeepEqual(got, want) {
		t.Errorf("FormatTriggers() = %v, want %v", got, want)
	}

	if got := (&LaunchdInfo{}).FormatTriggers(); len(got) != 0 {
		t.Errorf("FormatTriggers() on empty info = %v, want no triggers", got)
	}
}

func TestDomainDescription(t *testing.T) {
	t.Parallel()

	tests := []struct {
		domain string
		want   string
	}{
		{"system", "Launch Daemon"},
		{"gui/501", "Launch Agent"},
		{"user", "Launch Agent"},
		{"", "launchd service"},
	}

	for _, tc := range tests {
		info := &LaunchdInfo{Domain: tc.domain}
		if got := info.DomainDescription(); got != tc.want {
			t.Errorf("DomainDescription(%q) = %q, want %q", tc.domain, got, tc.want)
		}
	}
}
