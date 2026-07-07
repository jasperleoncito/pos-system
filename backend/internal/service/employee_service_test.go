package service

import (
	"testing"
	"time"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/employee"
)

func TestScheduleWindow(t *testing.T) {
	manila, err := time.LoadLocation("Asia/Manila")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	week := []employee.ScheduleDay{
		{DayOfWeek: 1, StartTime: "09:00", EndTime: "17:00", GraceMinutes: 10}, // Monday
		{DayOfWeek: 2, StartTime: "10:30", EndTime: "18:30", GraceMinutes: 5},  // Tuesday
	}

	tests := []struct {
		name      string
		now       time.Time
		wantOK    bool
		wantStart string
		wantEnd   string
		wantGrace int
	}{
		{
			name:      "matches Monday shift in tenant timezone",
			now:       time.Date(2026, 7, 6, 9, 25, 0, 0, manila), // a Monday
			wantOK:    true,
			wantStart: "09:00",
			wantEnd:   "17:00",
			wantGrace: 10,
		},
		{
			name:      "matches Tuesday shift with its own grace",
			now:       time.Date(2026, 7, 7, 8, 0, 0, 0, manila), // a Tuesday
			wantOK:    true,
			wantStart: "10:30",
			wantEnd:   "18:30",
			wantGrace: 5,
		},
		{
			name:   "returns not-ok on a day off",
			now:    time.Date(2026, 7, 5, 9, 0, 0, 0, manila), // a Sunday
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, grace, ok := scheduleWindow(week, tt.now)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if got := start.Format("15:04"); got != tt.wantStart {
				t.Errorf("start = %s, want %s", got, tt.wantStart)
			}
			if got := end.Format("15:04"); got != tt.wantEnd {
				t.Errorf("end = %s, want %s", got, tt.wantEnd)
			}
			if grace != tt.wantGrace {
				t.Errorf("grace = %d, want %d", grace, tt.wantGrace)
			}
			if start.Location() != manila {
				t.Errorf("start location = %v, want tenant timezone", start.Location())
			}
			if !start.Before(end) {
				t.Errorf("start %v should be before end %v", start, end)
			}
		})
	}
}

func TestScheduleWindowLatenessMath(t *testing.T) {
	manila, _ := time.LoadLocation("Asia/Manila")
	week := []employee.ScheduleDay{{DayOfWeek: 1, StartTime: "09:00", EndTime: "17:00", GraceMinutes: 10}}

	tests := []struct {
		name     string
		clockIn  time.Time
		wantLate int
	}{
		{"on time", time.Date(2026, 7, 6, 8, 55, 0, 0, manila), 0},
		{"inside grace", time.Date(2026, 7, 6, 9, 9, 0, 0, manila), 0},
		{"just past grace", time.Date(2026, 7, 6, 9, 25, 0, 0, manila), 15},
		{"an hour late", time.Date(2026, 7, 6, 10, 10, 0, 0, manila), 60},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, _, grace, ok := scheduleWindow(week, tt.clockIn)
			if !ok {
				t.Fatal("expected a schedule match")
			}
			late := 0
			if d := tt.clockIn.Sub(start.Add(time.Duration(grace) * time.Minute)); d > 0 {
				late = int(d.Minutes())
			}
			if late != tt.wantLate {
				t.Errorf("late = %d minutes, want %d", late, tt.wantLate)
			}
		})
	}
}

func TestValidSalaryType(t *testing.T) {
	for _, valid := range []string{"hourly", "daily", "monthly"} {
		if !validSalaryType(valid) {
			t.Errorf("expected %q to be valid", valid)
		}
	}
	for _, invalid := range []string{"", "weekly", "HOURLY"} {
		if validSalaryType(invalid) {
			t.Errorf("expected %q to be invalid", invalid)
		}
	}
}
