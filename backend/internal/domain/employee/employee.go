// Package employee defines staff profiles, weekly schedules, and
// attendance (clock in/out) records.
package employee

import (
	"context"
	"time"
)

// Salary types.
const (
	SalaryHourly  = "hourly"
	SalaryDaily   = "daily"
	SalaryMonthly = "monthly"
)

// Attendance statuses.
const (
	AttendancePending  = "pending"
	AttendanceApproved = "approved"
)

type Employee struct {
	ID         string     `json:"id"`
	UserID     *string    `json:"user_id"`
	UserEmail  string     `json:"user_email,omitempty"`
	FullName   string     `json:"full_name"`
	Position   string     `json:"position"`
	Phone      string     `json:"phone"`
	Email      string     `json:"email"`
	Address    string     `json:"address"`
	SalaryType string     `json:"salary_type"`
	SalaryRate int64      `json:"salary_rate"` // centavos
	HireDate   *time.Time `json:"hire_date"`
	PhotoPath  string     `json:"photo_path"`
	ThumbPath  string     `json:"thumb_path"`
	Notes      string     `json:"notes"`
	IsActive   bool       `json:"is_active"`
	CreatedAt  time.Time  `json:"created_at"`
}

// ScheduleDay is one working day in an employee's weekly template.
type ScheduleDay struct {
	ID           string `json:"id,omitempty"`
	EmployeeID   string `json:"employee_id,omitempty"`
	DayOfWeek    int    `json:"day_of_week"` // 0 = Sunday
	StartTime    string `json:"start_time"`  // "HH:MM"
	EndTime      string `json:"end_time"`    // "HH:MM"
	GraceMinutes int    `json:"grace_minutes"`
}

type Attendance struct {
	ID              string     `json:"id"`
	EmployeeID      string     `json:"employee_id"`
	EmployeeName    string     `json:"employee_name,omitempty"`
	ClockIn         time.Time  `json:"clock_in"`
	ClockOut        *time.Time `json:"clock_out"`
	ScheduledStart  *time.Time `json:"scheduled_start"`
	ScheduledEnd    *time.Time `json:"scheduled_end"`
	BreakStart      *time.Time `json:"break_start"`
	BreakMinutes    int        `json:"break_minutes"`
	LateMinutes     int        `json:"late_minutes"`
	EarlyOutMinutes int        `json:"early_out_minutes"`
	OvertimeMinutes int        `json:"overtime_minutes"`
	Status          string     `json:"status"`
	ApprovedBy      *string    `json:"approved_by"`
	ApprovedAt      *time.Time `json:"approved_at"`
	Notes           string     `json:"notes"`
	CreatedAt       time.Time  `json:"created_at"`
}

// AttendanceFilter narrows attendance listings.
type AttendanceFilter struct {
	EmployeeID string
	From       *time.Time
	To         *time.Time
	Limit      int
}

type Repository interface {
	Create(ctx context.Context, tenantID string, e *Employee) error
	GetByID(ctx context.Context, tenantID, id string) (*Employee, error)
	GetByUserID(ctx context.Context, tenantID, userID string) (*Employee, error)
	List(ctx context.Context, tenantID, search string) ([]Employee, error)
	Update(ctx context.Context, tenantID string, e *Employee) error
	SoftDelete(ctx context.Context, tenantID, id string) error

	ListSchedule(ctx context.Context, tenantID, employeeID string) ([]ScheduleDay, error)
	ReplaceSchedule(ctx context.Context, tenantID, employeeID string, days []ScheduleDay) error

	CreateAttendance(ctx context.Context, tenantID string, a *Attendance) error
	GetAttendance(ctx context.Context, tenantID, id string) (*Attendance, error)
	GetOpenAttendance(ctx context.Context, tenantID, employeeID string) (*Attendance, error)
	UpdateAttendance(ctx context.Context, tenantID string, a *Attendance) error
	ListAttendance(ctx context.Context, tenantID string, f AttendanceFilter) ([]Attendance, error)
	Approve(ctx context.Context, tenantID, id, approverUserID string) error
}
