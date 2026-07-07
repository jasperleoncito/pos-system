package v1

import (
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/employee"
	"github.com/jasperleoncito/pos-system/backend/internal/dto"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/imageproc"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/response"
	"github.com/jasperleoncito/pos-system/backend/internal/service"
)

// EmployeeHandler exposes staff profiles, schedules, and attendance.
type EmployeeHandler struct {
	employees *service.EmployeeService
}

func NewEmployeeHandler(e *service.EmployeeService) *EmployeeHandler {
	return &EmployeeHandler{employees: e}
}

func employeeInputFromRequest(req dto.EmployeeRequest) service.EmployeeInput {
	in := service.EmployeeInput{
		FullName: req.FullName, Position: req.Position, Phone: req.Phone,
		Email: req.Email, Address: req.Address, SalaryType: req.SalaryType,
		SalaryRate: req.SalaryRate, Notes: req.Notes,
		IsActive: boolOrDefault(req.IsActive, true), UserEmail: req.UserEmail,
	}
	if req.HireDate != "" {
		if d, err := time.Parse("2006-01-02", req.HireDate); err == nil {
			in.HireDate = &d
		}
	}
	return in
}

// ---- employees ----

// ListEmployees godoc
//
//	@Summary	List employees
//	@Tags		employees
//	@Security	BearerAuth
//	@Produce	json
//	@Param		search	query		string	false	"Name or position filter"
//	@Success	200		{object}	response.Envelope
//	@Router		/employees [get]
func (h *EmployeeHandler) ListEmployees(c *gin.Context) {
	tenantID, _ := tenantUser(c)
	employees, err := h.employees.List(c.Request.Context(), tenantID, c.Query("search"))
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", employees)
}

// GetEmployee godoc
//
//	@Summary	Get one employee
//	@Tags		employees
//	@Security	BearerAuth
//	@Produce	json
//	@Param		id	path		string	true	"Employee ID"
//	@Success	200	{object}	response.Envelope
//	@Router		/employees/{id} [get]
func (h *EmployeeHandler) GetEmployee(c *gin.Context) {
	tenantID, _ := tenantUser(c)
	e, err := h.employees.Get(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", e)
}

// CreateEmployee godoc
//
//	@Summary	Create an employee
//	@Tags		employees
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.EmployeeRequest	true	"Employee"
//	@Success	201		{object}	response.Envelope
//	@Router		/employees [post]
func (h *EmployeeHandler) CreateEmployee(c *gin.Context) {
	var req dto.EmployeeRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	created, err := h.employees.Create(c.Request.Context(), tenantID, userID, employeeInputFromRequest(req))
	if err != nil {
		respondError(c, err)
		return
	}
	response.Created(c, "employee created", created)
}

// UpdateEmployee godoc
//
//	@Summary	Update an employee
//	@Tags		employees
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string				true	"Employee ID"
//	@Param		payload	body		dto.EmployeeRequest	true	"Employee"
//	@Success	200		{object}	response.Envelope
//	@Router		/employees/{id} [put]
func (h *EmployeeHandler) UpdateEmployee(c *gin.Context) {
	var req dto.EmployeeRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	updated, err := h.employees.Update(c.Request.Context(), tenantID, userID, c.Param("id"), employeeInputFromRequest(req))
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "employee updated", updated)
}

// DeleteEmployee godoc
//
//	@Summary	Delete an employee
//	@Tags		employees
//	@Security	BearerAuth
//	@Produce	json
//	@Param		id	path		string	true	"Employee ID"
//	@Success	200	{object}	response.Envelope
//	@Router		/employees/{id} [delete]
func (h *EmployeeHandler) DeleteEmployee(c *gin.Context) {
	tenantID, userID := tenantUser(c)
	if err := h.employees.Delete(c.Request.Context(), tenantID, userID, c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "employee deleted", nil)
}

// UploadEmployeePhoto godoc
//
//	@Summary	Upload an employee photo (optimized to WebP automatically)
//	@Tags		employees
//	@Security	BearerAuth
//	@Accept		multipart/form-data
//	@Produce	json
//	@Param		id		path		string	true	"Employee ID"
//	@Param		image	formData	file	true	"PNG/JPG/WEBP, max 10MB"
//	@Success	200		{object}	response.Envelope
//	@Router		/employees/{id}/photo [post]
func (h *EmployeeHandler) UploadEmployeePhoto(c *gin.Context) {
	file, _, err := c.Request.FormFile("image")
	if err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "attach an image file in the 'image' field")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, imageproc.MaxUploadBytes+1))
	if err != nil {
		respondError(c, err)
		return
	}

	tenantID, userID := tenantUser(c)
	updated, err := h.employees.UploadPhoto(c.Request.Context(), tenantID, userID, c.Param("id"), data)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "employee photo updated", updated)
}

// ---- schedules ----

// GetSchedule godoc
//
//	@Summary	Get an employee's weekly schedule
//	@Tags		employees
//	@Security	BearerAuth
//	@Produce	json
//	@Param		id	path		string	true	"Employee ID"
//	@Success	200	{object}	response.Envelope
//	@Router		/employees/{id}/schedule [get]
func (h *EmployeeHandler) GetSchedule(c *gin.Context) {
	tenantID, _ := tenantUser(c)
	days, err := h.employees.GetSchedule(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", days)
}

// SaveSchedule godoc
//
//	@Summary	Replace an employee's weekly schedule
//	@Tags		employees
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string					true	"Employee ID"
//	@Param		payload	body		dto.SaveScheduleRequest	true	"Weekly schedule"
//	@Success	200		{object}	response.Envelope
//	@Router		/employees/{id}/schedule [put]
func (h *EmployeeHandler) SaveSchedule(c *gin.Context) {
	var req dto.SaveScheduleRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	days := make([]employee.ScheduleDay, len(req.Days))
	for i, d := range req.Days {
		days[i] = employee.ScheduleDay{
			DayOfWeek: d.DayOfWeek, StartTime: d.StartTime, EndTime: d.EndTime,
			GraceMinutes: d.GraceMinutes,
		}
	}
	saved, err := h.employees.SaveSchedule(c.Request.Context(), tenantID, userID, c.Param("id"), days)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "schedule saved", saved)
}

// ---- attendance (self-service clock) ----

// MyClockStatus godoc
//
//	@Summary	Current clock state for the signed-in employee
//	@Tags		attendance
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	response.Envelope
//	@Router		/attendance/me [get]
func (h *EmployeeHandler) MyClockStatus(c *gin.Context) {
	tenantID, userID := tenantUser(c)
	status, err := h.employees.MyStatus(c.Request.Context(), tenantID, userID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", status)
}

// ClockIn godoc
//
//	@Summary	Clock in (server time; computes lateness vs schedule)
//	@Tags		attendance
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	response.Envelope
//	@Router		/attendance/clock-in [post]
func (h *EmployeeHandler) ClockIn(c *gin.Context) {
	tenantID, userID := tenantUser(c)
	a, err := h.employees.ClockIn(c.Request.Context(), tenantID, userID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "clocked in", a)
}

// ClockOut godoc
//
//	@Summary	Clock out (computes early-out and overtime)
//	@Tags		attendance
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	response.Envelope
//	@Router		/attendance/clock-out [post]
func (h *EmployeeHandler) ClockOut(c *gin.Context) {
	tenantID, userID := tenantUser(c)
	a, err := h.employees.ClockOut(c.Request.Context(), tenantID, userID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "clocked out", a)
}

// StartBreak godoc
//
//	@Summary	Start a break on the open shift
//	@Tags		attendance
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	response.Envelope
//	@Router		/attendance/break/start [post]
func (h *EmployeeHandler) StartBreak(c *gin.Context) {
	tenantID, userID := tenantUser(c)
	a, err := h.employees.StartBreak(c.Request.Context(), tenantID, userID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "break started", a)
}

// EndBreak godoc
//
//	@Summary	End the running break
//	@Tags		attendance
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	response.Envelope
//	@Router		/attendance/break/end [post]
func (h *EmployeeHandler) EndBreak(c *gin.Context) {
	tenantID, userID := tenantUser(c)
	a, err := h.employees.EndBreak(c.Request.Context(), tenantID, userID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "break ended", a)
}

// ---- attendance (review & reports) ----

// ListAttendance godoc
//
//	@Summary	List attendance records
//	@Tags		attendance
//	@Security	BearerAuth
//	@Produce	json
//	@Param		employee_id	query		string	false	"Filter by employee"
//	@Param		from		query		string	false	"ISO date (inclusive)"
//	@Param		to			query		string	false	"ISO date (exclusive)"
//	@Success	200			{object}	response.Envelope
//	@Router		/attendance [get]
func (h *EmployeeHandler) ListAttendance(c *gin.Context) {
	tenantID, _ := tenantUser(c)
	filter := employee.AttendanceFilter{EmployeeID: c.Query("employee_id")}
	if from := c.Query("from"); from != "" {
		if t, err := time.Parse("2006-01-02", from); err == nil {
			filter.From = &t
		}
	}
	if to := c.Query("to"); to != "" {
		if t, err := time.Parse("2006-01-02", to); err == nil {
			end := t.AddDate(0, 0, 1) // exclusive upper bound: include the whole "to" day
			filter.To = &end
		}
	}
	records, err := h.employees.ListAttendance(c.Request.Context(), tenantID, filter)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", records)
}

// ApproveAttendance godoc
//
//	@Summary	Approve a completed attendance record (manager+)
//	@Tags		attendance
//	@Security	BearerAuth
//	@Produce	json
//	@Param		id	path		string	true	"Attendance ID"
//	@Success	200	{object}	response.Envelope
//	@Router		/attendance/{id}/approve [post]
func (h *EmployeeHandler) ApproveAttendance(c *gin.Context) {
	tenantID, userID := tenantUser(c)
	a, err := h.employees.ApproveAttendance(c.Request.Context(), tenantID, userID, c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "attendance approved", a)
}
