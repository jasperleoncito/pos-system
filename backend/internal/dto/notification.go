package dto

type NotificationPrefsRequest struct {
	EmailLowStock     *bool `json:"email_low_stock"`
	EmailAttendance   *bool `json:"email_attendance"`
	EmailDailySummary *bool `json:"email_daily_summary"`
}
