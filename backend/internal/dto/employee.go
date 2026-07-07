package dto

type EmployeeRequest struct {
	FullName   string `json:"full_name" binding:"required,min=1,max=120"`
	Position   string `json:"position" binding:"max=120"`
	Phone      string `json:"phone" binding:"max=40"`
	Email      string `json:"email" binding:"omitempty,email"`
	Address    string `json:"address" binding:"max=300"`
	SalaryType string `json:"salary_type" binding:"required,oneof=hourly daily monthly"`
	SalaryRate int64  `json:"salary_rate" binding:"min=0"`
	HireDate   string `json:"hire_date" binding:"omitempty,datetime=2006-01-02"`
	Notes      string `json:"notes" binding:"max=500"`
	IsActive   *bool  `json:"is_active"`
	UserEmail  string `json:"user_email" binding:"omitempty,email"` // optional login link
}

type ScheduleDayInput struct {
	DayOfWeek    int    `json:"day_of_week" binding:"min=0,max=6"`
	StartTime    string `json:"start_time" binding:"required"`
	EndTime      string `json:"end_time" binding:"required"`
	GraceMinutes int    `json:"grace_minutes" binding:"min=0,max=240"`
}

// SaveScheduleRequest replaces the weekly template; an empty list clears it.
type SaveScheduleRequest struct {
	Days []ScheduleDayInput `json:"days" binding:"dive"`
}
