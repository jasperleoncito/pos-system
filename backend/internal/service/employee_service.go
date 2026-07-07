package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/audit"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/auth"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/employee"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/storage"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/tenant"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/imageproc"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/queue"
)

// EmployeeService owns staff profiles, weekly schedules, and attendance.
// All clock math uses server time in the tenant's timezone.
type EmployeeService struct {
	repo        employee.Repository
	users       auth.UserRepository
	memberships tenant.MembershipRepository
	tenants     tenant.Repository
	store       storage.ObjectStorage
	jobs        Jobs
	auditor     *AuditService
	logger      *slog.Logger
}

// SetJobs wires the background queue for late-clock-in alerts.
func (s *EmployeeService) SetJobs(jobs Jobs) { s.jobs = jobs }

func NewEmployeeService(
	repo employee.Repository,
	users auth.UserRepository,
	memberships tenant.MembershipRepository,
	tenants tenant.Repository,
	store storage.ObjectStorage,
	auditor *AuditService,
	logger *slog.Logger,
) *EmployeeService {
	return &EmployeeService{
		repo: repo, users: users, memberships: memberships, tenants: tenants,
		store: store, auditor: auditor, logger: logger,
	}
}

func validSalaryType(t string) bool {
	return t == employee.SalaryHourly || t == employee.SalaryDaily || t == employee.SalaryMonthly
}

// resolveUserLink turns an optional login email into a user ID, requiring
// the account to already be a member of the tenant.
func (s *EmployeeService) resolveUserLink(ctx context.Context, tenantID, email string) (*string, error) {
	if email == "" {
		return nil, nil
	}
	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, apperror.Validation("no account exists with that email")
	}
	if _, err := s.memberships.Get(ctx, tenantID, u.ID); err != nil {
		return nil, apperror.Validation("that account is not a member of this restaurant")
	}
	return &u.ID, nil
}

// ---- employees ----

// EmployeeView decorates an employee with browser-facing photo URLs.
type EmployeeView struct {
	employee.Employee
	PhotoURL string `json:"photo_url"`
	ThumbURL string `json:"thumb_url"`
}

func (s *EmployeeService) view(e employee.Employee) EmployeeView {
	return EmployeeView{
		Employee: e,
		PhotoURL: s.store.PublicURL(e.PhotoPath),
		ThumbURL: s.store.PublicURL(e.ThumbPath),
	}
}

func (s *EmployeeService) viewByID(ctx context.Context, tenantID, id string) (*EmployeeView, error) {
	e, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	v := s.view(*e)
	return &v, nil
}

// EmployeeInput carries validated employee fields from the handler.
type EmployeeInput struct {
	FullName   string
	Position   string
	Phone      string
	Email      string
	Address    string
	SalaryType string
	SalaryRate int64
	HireDate   *time.Time
	Notes      string
	IsActive   bool
	UserEmail  string // optional login link
}

func (s *EmployeeService) Create(ctx context.Context, tenantID, userID string, in EmployeeInput) (*EmployeeView, error) {
	if !validSalaryType(in.SalaryType) {
		return nil, apperror.Validation("salary type must be hourly, daily, or monthly")
	}
	link, err := s.resolveUserLink(ctx, tenantID, in.UserEmail)
	if err != nil {
		return nil, err
	}
	e := &employee.Employee{
		UserID: link, FullName: in.FullName, Position: in.Position, Phone: in.Phone,
		Email: in.Email, Address: in.Address, SalaryType: in.SalaryType, SalaryRate: in.SalaryRate,
		HireDate: in.HireDate, Notes: in.Notes, IsActive: in.IsActive,
	}
	if err := s.repo.Create(ctx, tenantID, e); err != nil {
		return nil, err
	}
	s.auditor.Record(audit.Log{TenantID: tenantID, UserID: userID, Action: "employee.created",
		EntityType: "employee", EntityID: e.ID, After: map[string]any{"full_name": e.FullName}})
	return s.viewByID(ctx, tenantID, e.ID)
}

func (s *EmployeeService) List(ctx context.Context, tenantID, search string) ([]EmployeeView, error) {
	employees, err := s.repo.List(ctx, tenantID, search)
	if err != nil {
		return nil, err
	}
	views := make([]EmployeeView, len(employees))
	for i, e := range employees {
		views[i] = s.view(e)
	}
	return views, nil
}

func (s *EmployeeService) Get(ctx context.Context, tenantID, id string) (*EmployeeView, error) {
	return s.viewByID(ctx, tenantID, id)
}

func (s *EmployeeService) Update(ctx context.Context, tenantID, userID, id string, in EmployeeInput) (*EmployeeView, error) {
	if !validSalaryType(in.SalaryType) {
		return nil, apperror.Validation("salary type must be hourly, daily, or monthly")
	}
	e, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	link, err := s.resolveUserLink(ctx, tenantID, in.UserEmail)
	if err != nil {
		return nil, err
	}
	e.UserID = link
	e.FullName = in.FullName
	e.Position = in.Position
	e.Phone = in.Phone
	e.Email = in.Email
	e.Address = in.Address
	e.SalaryType = in.SalaryType
	e.SalaryRate = in.SalaryRate
	e.HireDate = in.HireDate
	e.Notes = in.Notes
	e.IsActive = in.IsActive
	if err := s.repo.Update(ctx, tenantID, e); err != nil {
		return nil, err
	}
	s.auditor.Record(audit.Log{TenantID: tenantID, UserID: userID, Action: "employee.updated",
		EntityType: "employee", EntityID: id, After: map[string]any{"full_name": e.FullName}})
	return s.viewByID(ctx, tenantID, id)
}

func (s *EmployeeService) Delete(ctx context.Context, tenantID, userID, id string) error {
	if err := s.repo.SoftDelete(ctx, tenantID, id); err != nil {
		return err
	}
	s.auditor.Record(audit.Log{TenantID: tenantID, UserID: userID, Action: "employee.deleted",
		EntityType: "employee", EntityID: id})
	return nil
}

// UploadPhoto optimizes and stores an employee photo.
func (s *EmployeeService) UploadPhoto(ctx context.Context, tenantID, userID, id string, data []byte) (*EmployeeView, error) {
	e, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	result, err := imageproc.Optimize(data)
	if err != nil {
		return nil, err
	}
	version := time.Now().Unix()
	imageKey := storage.TenantKey(tenantID, storage.FolderEmployees, fmt.Sprintf("%s-%d.webp", id, version))
	thumbKey := storage.TenantKey(tenantID, storage.FolderEmployees, fmt.Sprintf("%s-%d-thumb.webp", id, version))
	if err := s.store.Put(ctx, imageKey, result.WebP, "image/webp"); err != nil {
		return nil, apperror.Internal(err)
	}
	if err := s.store.Put(ctx, thumbKey, result.ThumbWebP, "image/webp"); err != nil {
		return nil, apperror.Internal(err)
	}
	e.PhotoPath = imageKey
	e.ThumbPath = thumbKey
	if err := s.repo.Update(ctx, tenantID, e); err != nil {
		return nil, err
	}
	s.auditor.Record(audit.Log{TenantID: tenantID, UserID: userID, Action: "employee.photo_updated",
		EntityType: "employee", EntityID: id})
	return s.viewByID(ctx, tenantID, id)
}

// ---- schedules ----

func (s *EmployeeService) GetSchedule(ctx context.Context, tenantID, employeeID string) ([]employee.ScheduleDay, error) {
	if _, err := s.repo.GetByID(ctx, tenantID, employeeID); err != nil {
		return nil, err
	}
	return s.repo.ListSchedule(ctx, tenantID, employeeID)
}

func (s *EmployeeService) SaveSchedule(ctx context.Context, tenantID, userID, employeeID string, days []employee.ScheduleDay) ([]employee.ScheduleDay, error) {
	if _, err := s.repo.GetByID(ctx, tenantID, employeeID); err != nil {
		return nil, err
	}
	seen := map[int]bool{}
	for _, d := range days {
		if d.DayOfWeek < 0 || d.DayOfWeek > 6 {
			return nil, apperror.Validation("day_of_week must be between 0 (Sunday) and 6 (Saturday)")
		}
		if seen[d.DayOfWeek] {
			return nil, apperror.Validation("each day can appear only once in a schedule")
		}
		seen[d.DayOfWeek] = true
		start, errS := time.Parse("15:04", d.StartTime)
		end, errE := time.Parse("15:04", d.EndTime)
		if errS != nil || errE != nil {
			return nil, apperror.Validation("times must use the HH:MM 24-hour format")
		}
		if !start.Before(end) {
			return nil, apperror.Validation("shift start must be before shift end")
		}
		if d.GraceMinutes < 0 || d.GraceMinutes > 240 {
			return nil, apperror.Validation("grace minutes must be between 0 and 240")
		}
	}
	if err := s.repo.ReplaceSchedule(ctx, tenantID, employeeID, days); err != nil {
		return nil, apperror.Internal(err)
	}
	s.auditor.Record(audit.Log{TenantID: tenantID, UserID: userID, Action: "employee.schedule_updated",
		EntityType: "employee", EntityID: employeeID, After: map[string]any{"days": len(days)}})
	return s.repo.ListSchedule(ctx, tenantID, employeeID)
}

// ---- attendance ----

// tenantNow returns the current server time in the tenant's timezone.
func (s *EmployeeService) tenantNow(ctx context.Context, tenantID string) (time.Time, error) {
	t, err := s.tenants.GetByID(ctx, tenantID)
	if err != nil {
		return time.Time{}, err
	}
	loc, err := time.LoadLocation(t.Timezone)
	if err != nil {
		loc = time.UTC
	}
	return time.Now().In(loc), nil
}

// selfEmployee resolves the calling user's employee profile.
func (s *EmployeeService) selfEmployee(ctx context.Context, tenantID, userID string) (*employee.Employee, error) {
	e, err := s.repo.GetByUserID(ctx, tenantID, userID)
	if err != nil {
		return nil, apperror.Validation("no employee profile is linked to your account — ask a manager to link one")
	}
	if !e.IsActive {
		return nil, apperror.Validation("your employee profile is inactive")
	}
	return e, nil
}

// scheduleWindow snapshots today's scheduled start/end for the clock-in
// moment; ok is false on a day off.
func scheduleWindow(days []employee.ScheduleDay, now time.Time) (start, end time.Time, grace int, ok bool) {
	for _, d := range days {
		if d.DayOfWeek != int(now.Weekday()) {
			continue
		}
		st, errS := time.Parse("15:04", d.StartTime)
		en, errE := time.Parse("15:04", d.EndTime)
		if errS != nil || errE != nil {
			return time.Time{}, time.Time{}, 0, false
		}
		start = time.Date(now.Year(), now.Month(), now.Day(), st.Hour(), st.Minute(), 0, 0, now.Location())
		end = time.Date(now.Year(), now.Month(), now.Day(), en.Hour(), en.Minute(), 0, 0, now.Location())
		return start, end, d.GraceMinutes, true
	}
	return time.Time{}, time.Time{}, 0, false
}

// ClockStatus is what the self-service clock page renders.
type ClockStatus struct {
	Employee      *EmployeeView         `json:"employee"`
	TodaySchedule *employee.ScheduleDay `json:"today_schedule"`
	Open          *employee.Attendance  `json:"open"`
	ServerTime    time.Time             `json:"server_time"`
}

func (s *EmployeeService) MyStatus(ctx context.Context, tenantID, userID string) (*ClockStatus, error) {
	e, err := s.selfEmployee(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	now, err := s.tenantNow(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	days, err := s.repo.ListSchedule(ctx, tenantID, e.ID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	var today *employee.ScheduleDay
	for i := range days {
		if days[i].DayOfWeek == int(now.Weekday()) {
			today = &days[i]
			break
		}
	}
	open, err := s.repo.GetOpenAttendance(ctx, tenantID, e.ID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	view := s.view(*e)
	return &ClockStatus{Employee: &view, TodaySchedule: today, Open: open, ServerTime: now}, nil
}

// ClockIn opens a shift at server time, snapshotting today's schedule and
// computing lateness beyond the grace period.
func (s *EmployeeService) ClockIn(ctx context.Context, tenantID, userID string) (*employee.Attendance, error) {
	e, err := s.selfEmployee(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	open, err := s.repo.GetOpenAttendance(ctx, tenantID, e.ID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if open != nil {
		return nil, apperror.Conflict("you are already clocked in")
	}
	now, err := s.tenantNow(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	a := &employee.Attendance{EmployeeID: e.ID, ClockIn: now, Status: employee.AttendancePending}
	days, err := s.repo.ListSchedule(ctx, tenantID, e.ID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if start, end, grace, ok := scheduleWindow(days, now); ok {
		a.ScheduledStart = &start
		a.ScheduledEnd = &end
		if late := now.Sub(start.Add(time.Duration(grace) * time.Minute)); late > 0 {
			a.LateMinutes = int(late.Minutes())
		}
	}
	if err := s.repo.CreateAttendance(ctx, tenantID, a); err != nil {
		return nil, err
	}
	s.auditor.Record(audit.Log{TenantID: tenantID, UserID: userID, Action: "attendance.clock_in",
		EntityType: "attendance", EntityID: a.ID, After: map[string]any{"late_minutes": a.LateMinutes}})

	// Late arrivals ping the managers, off the hot path.
	if a.LateMinutes > 0 && s.jobs != nil {
		if err := s.jobs.EnqueueAttendanceAlert(queue.AttendanceAlertPayload{
			TenantID: tenantID, EmployeeName: e.FullName, LateMinutes: a.LateMinutes,
			ClockInLocal: now.Format("Jan 2 3:04 PM"),
		}); err != nil {
			s.logger.Warn("failed to enqueue attendance alert", "employee", e.FullName, "error", err)
		}
	}
	return a, nil
}

// ClockOut closes the open shift, ending any running break and computing
// early-out and overtime against the schedule snapshot.
func (s *EmployeeService) ClockOut(ctx context.Context, tenantID, userID string) (*employee.Attendance, error) {
	e, err := s.selfEmployee(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	a, err := s.repo.GetOpenAttendance(ctx, tenantID, e.ID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if a == nil {
		return nil, apperror.Validation("you are not clocked in")
	}
	now, err := s.tenantNow(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	if a.BreakStart != nil {
		a.BreakMinutes += int(now.Sub(*a.BreakStart).Minutes())
		a.BreakStart = nil
	}
	a.ClockOut = &now
	if a.ScheduledEnd != nil {
		if early := a.ScheduledEnd.Sub(now); early > 0 {
			a.EarlyOutMinutes = int(early.Minutes())
		}
		if over := now.Sub(*a.ScheduledEnd); over > 0 {
			a.OvertimeMinutes = int(over.Minutes())
		}
	}
	if err := s.repo.UpdateAttendance(ctx, tenantID, a); err != nil {
		return nil, err
	}
	s.auditor.Record(audit.Log{TenantID: tenantID, UserID: userID, Action: "attendance.clock_out",
		EntityType: "attendance", EntityID: a.ID,
		After: map[string]any{"early_out_minutes": a.EarlyOutMinutes, "overtime_minutes": a.OvertimeMinutes}})
	return a, nil
}

// StartBreak begins an unpaid break on the open shift.
func (s *EmployeeService) StartBreak(ctx context.Context, tenantID, userID string) (*employee.Attendance, error) {
	e, err := s.selfEmployee(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	a, err := s.repo.GetOpenAttendance(ctx, tenantID, e.ID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if a == nil {
		return nil, apperror.Validation("you are not clocked in")
	}
	if a.BreakStart != nil {
		return nil, apperror.Conflict("a break is already running")
	}
	now, err := s.tenantNow(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	a.BreakStart = &now
	if err := s.repo.UpdateAttendance(ctx, tenantID, a); err != nil {
		return nil, err
	}
	return a, nil
}

// EndBreak stops the running break and accumulates its minutes.
func (s *EmployeeService) EndBreak(ctx context.Context, tenantID, userID string) (*employee.Attendance, error) {
	e, err := s.selfEmployee(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	a, err := s.repo.GetOpenAttendance(ctx, tenantID, e.ID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if a == nil || a.BreakStart == nil {
		return nil, apperror.Validation("no break is running")
	}
	now, err := s.tenantNow(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	a.BreakMinutes += int(now.Sub(*a.BreakStart).Minutes())
	a.BreakStart = nil
	if err := s.repo.UpdateAttendance(ctx, tenantID, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *EmployeeService) ListAttendance(ctx context.Context, tenantID string, f employee.AttendanceFilter) ([]employee.Attendance, error) {
	return s.repo.ListAttendance(ctx, tenantID, f)
}

// ApproveAttendance marks a completed record approved (manager+).
func (s *EmployeeService) ApproveAttendance(ctx context.Context, tenantID, userID, id string) (*employee.Attendance, error) {
	if err := s.repo.Approve(ctx, tenantID, id, userID); err != nil {
		return nil, err
	}
	s.auditor.Record(audit.Log{TenantID: tenantID, UserID: userID, Action: "attendance.approved",
		EntityType: "attendance", EntityID: id})
	return s.repo.GetAttendance(ctx, tenantID, id)
}
