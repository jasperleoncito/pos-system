package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/employee"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
)

type EmployeeRepo struct {
	db *pgxpool.Pool
}

func NewEmployeeRepo(db *pgxpool.Pool) *EmployeeRepo { return &EmployeeRepo{db: db} }

// ---- employees ----

const employeeColumns = `
	e.id, e.user_id, COALESCE(u.email, ''), e.full_name, e.position, e.phone, e.email, e.address,
	e.salary_type, e.salary_rate, e.hire_date, e.photo_path, e.thumb_path, e.notes, e.is_active, e.created_at`

func scanEmployee(row pgx.Row) (*employee.Employee, error) {
	var e employee.Employee
	err := row.Scan(&e.ID, &e.UserID, &e.UserEmail, &e.FullName, &e.Position, &e.Phone, &e.Email, &e.Address,
		&e.SalaryType, &e.SalaryRate, &e.HireDate, &e.PhotoPath, &e.ThumbPath, &e.Notes, &e.IsActive, &e.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.NotFound("employee")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan employee: %w", err)
	}
	return &e, nil
}

func (r *EmployeeRepo) Create(ctx context.Context, tenantID string, e *employee.Employee) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO employees (tenant_id, user_id, full_name, position, phone, email, address,
			salary_type, salary_rate, hire_date, notes, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING id, created_at`,
		tenantID, e.UserID, e.FullName, e.Position, e.Phone, e.Email, e.Address,
		e.SalaryType, e.SalaryRate, e.HireDate, e.Notes, e.IsActive,
	).Scan(&e.ID, &e.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return apperror.Conflict("that user account is already linked to another employee")
		}
		return fmt.Errorf("failed to create employee: %w", err)
	}
	return nil
}

func (r *EmployeeRepo) GetByID(ctx context.Context, tenantID, id string) (*employee.Employee, error) {
	return scanEmployee(r.db.QueryRow(ctx, `
		SELECT `+employeeColumns+` FROM employees e
		LEFT JOIN users u ON u.id = e.user_id
		WHERE e.tenant_id=$1 AND e.id=$2 AND e.deleted_at IS NULL`, tenantID, id))
}

func (r *EmployeeRepo) GetByUserID(ctx context.Context, tenantID, userID string) (*employee.Employee, error) {
	return scanEmployee(r.db.QueryRow(ctx, `
		SELECT `+employeeColumns+` FROM employees e
		LEFT JOIN users u ON u.id = e.user_id
		WHERE e.tenant_id=$1 AND e.user_id=$2 AND e.deleted_at IS NULL`, tenantID, userID))
}

func (r *EmployeeRepo) List(ctx context.Context, tenantID, search string) ([]employee.Employee, error) {
	query := `
		SELECT ` + employeeColumns + ` FROM employees e
		LEFT JOIN users u ON u.id = e.user_id
		WHERE e.tenant_id=$1 AND e.deleted_at IS NULL`
	args := []any{tenantID}
	if search != "" {
		query += ` AND (e.full_name ILIKE $2 OR e.position ILIKE $2)`
		args = append(args, "%"+search+"%")
	}
	query += ` ORDER BY e.full_name`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list employees: %w", err)
	}
	defer rows.Close()
	var employees []employee.Employee
	for rows.Next() {
		e, err := scanEmployee(rows)
		if err != nil {
			return nil, err
		}
		employees = append(employees, *e)
	}
	return employees, rows.Err()
}

func (r *EmployeeRepo) Update(ctx context.Context, tenantID string, e *employee.Employee) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE employees SET user_id=$3, full_name=$4, position=$5, phone=$6, email=$7, address=$8,
			salary_type=$9, salary_rate=$10, hire_date=$11, photo_path=$12, thumb_path=$13,
			notes=$14, is_active=$15, updated_at=now()
		WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL`,
		tenantID, e.ID, e.UserID, e.FullName, e.Position, e.Phone, e.Email, e.Address,
		e.SalaryType, e.SalaryRate, e.HireDate, e.PhotoPath, e.ThumbPath, e.Notes, e.IsActive)
	if err != nil {
		if isUniqueViolation(err) {
			return apperror.Conflict("that user account is already linked to another employee")
		}
		return fmt.Errorf("failed to update employee: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("employee")
	}
	return nil
}

func (r *EmployeeRepo) SoftDelete(ctx context.Context, tenantID, id string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE employees SET deleted_at=now(), updated_at=now()
		WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL`, tenantID, id)
	if err != nil {
		return fmt.Errorf("failed to delete employee: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("employee")
	}
	return nil
}

// ---- schedules ----

func (r *EmployeeRepo) ListSchedule(ctx context.Context, tenantID, employeeID string) ([]employee.ScheduleDay, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, employee_id, day_of_week, to_char(start_time, 'HH24:MI'), to_char(end_time, 'HH24:MI'), grace_minutes
		FROM employee_schedules WHERE tenant_id=$1 AND employee_id=$2 ORDER BY day_of_week`, tenantID, employeeID)
	if err != nil {
		return nil, fmt.Errorf("failed to list schedule: %w", err)
	}
	defer rows.Close()
	var days []employee.ScheduleDay
	for rows.Next() {
		var d employee.ScheduleDay
		if err := rows.Scan(&d.ID, &d.EmployeeID, &d.DayOfWeek, &d.StartTime, &d.EndTime, &d.GraceMinutes); err != nil {
			return nil, fmt.Errorf("failed to scan schedule day: %w", err)
		}
		days = append(days, d)
	}
	return days, rows.Err()
}

func (r *EmployeeRepo) ReplaceSchedule(ctx context.Context, tenantID, employeeID string, days []employee.ScheduleDay) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		DELETE FROM employee_schedules WHERE tenant_id=$1 AND employee_id=$2`, tenantID, employeeID); err != nil {
		return fmt.Errorf("failed to clear schedule: %w", err)
	}
	for _, d := range days {
		if _, err := tx.Exec(ctx, `
			INSERT INTO employee_schedules (tenant_id, employee_id, day_of_week, start_time, end_time, grace_minutes)
			VALUES ($1, $2, $3, $4::time, $5::time, $6)`,
			tenantID, employeeID, d.DayOfWeek, d.StartTime, d.EndTime, d.GraceMinutes); err != nil {
			return fmt.Errorf("failed to insert schedule day: %w", err)
		}
	}
	return tx.Commit(ctx)
}

// ---- attendance ----

const attendanceColumns = `
	a.id, a.employee_id, e.full_name, a.clock_in, a.clock_out, a.scheduled_start, a.scheduled_end,
	a.break_start, a.break_minutes, a.late_minutes, a.early_out_minutes, a.overtime_minutes,
	a.status, a.approved_by, a.approved_at, a.notes, a.created_at`

func scanAttendance(row pgx.Row) (*employee.Attendance, error) {
	var a employee.Attendance
	err := row.Scan(&a.ID, &a.EmployeeID, &a.EmployeeName, &a.ClockIn, &a.ClockOut, &a.ScheduledStart, &a.ScheduledEnd,
		&a.BreakStart, &a.BreakMinutes, &a.LateMinutes, &a.EarlyOutMinutes, &a.OvertimeMinutes,
		&a.Status, &a.ApprovedBy, &a.ApprovedAt, &a.Notes, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *EmployeeRepo) CreateAttendance(ctx context.Context, tenantID string, a *employee.Attendance) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO attendance_records (tenant_id, employee_id, clock_in, scheduled_start, scheduled_end, late_minutes, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, created_at`,
		tenantID, a.EmployeeID, a.ClockIn, a.ScheduledStart, a.ScheduledEnd, a.LateMinutes, a.Notes,
	).Scan(&a.ID, &a.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return apperror.Conflict("this employee is already clocked in")
		}
		return fmt.Errorf("failed to create attendance: %w", err)
	}
	return nil
}

func (r *EmployeeRepo) GetAttendance(ctx context.Context, tenantID, id string) (*employee.Attendance, error) {
	a, err := scanAttendance(r.db.QueryRow(ctx, `
		SELECT `+attendanceColumns+` FROM attendance_records a
		JOIN employees e ON e.id = a.employee_id
		WHERE a.tenant_id=$1 AND a.id=$2`, tenantID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.NotFound("attendance record")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get attendance: %w", err)
	}
	return a, nil
}

// GetOpenAttendance returns the employee's running shift, or (nil, nil)
// when they are not clocked in.
func (r *EmployeeRepo) GetOpenAttendance(ctx context.Context, tenantID, employeeID string) (*employee.Attendance, error) {
	a, err := scanAttendance(r.db.QueryRow(ctx, `
		SELECT `+attendanceColumns+` FROM attendance_records a
		JOIN employees e ON e.id = a.employee_id
		WHERE a.tenant_id=$1 AND a.employee_id=$2 AND a.clock_out IS NULL`, tenantID, employeeID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get open attendance: %w", err)
	}
	return a, nil
}

func (r *EmployeeRepo) UpdateAttendance(ctx context.Context, tenantID string, a *employee.Attendance) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE attendance_records SET clock_out=$3, break_start=$4, break_minutes=$5,
			early_out_minutes=$6, overtime_minutes=$7, notes=$8, updated_at=now()
		WHERE tenant_id=$1 AND id=$2`,
		tenantID, a.ID, a.ClockOut, a.BreakStart, a.BreakMinutes,
		a.EarlyOutMinutes, a.OvertimeMinutes, a.Notes)
	if err != nil {
		return fmt.Errorf("failed to update attendance: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("attendance record")
	}
	return nil
}

func (r *EmployeeRepo) ListAttendance(ctx context.Context, tenantID string, f employee.AttendanceFilter) ([]employee.Attendance, error) {
	query := `
		SELECT ` + attendanceColumns + ` FROM attendance_records a
		JOIN employees e ON e.id = a.employee_id
		WHERE a.tenant_id=$1`
	args := []any{tenantID}
	if f.EmployeeID != "" {
		args = append(args, f.EmployeeID)
		query += fmt.Sprintf(" AND a.employee_id=$%d", len(args))
	}
	if f.From != nil {
		args = append(args, *f.From)
		query += fmt.Sprintf(" AND a.clock_in >= $%d", len(args))
	}
	if f.To != nil {
		args = append(args, *f.To)
		query += fmt.Sprintf(" AND a.clock_in < $%d", len(args))
	}
	limit := f.Limit
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	args = append(args, limit)
	query += fmt.Sprintf(" ORDER BY a.clock_in DESC LIMIT $%d", len(args))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list attendance: %w", err)
	}
	defer rows.Close()
	var records []employee.Attendance
	for rows.Next() {
		a, err := scanAttendance(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan attendance: %w", err)
		}
		records = append(records, *a)
	}
	return records, rows.Err()
}

func (r *EmployeeRepo) Approve(ctx context.Context, tenantID, id, approverUserID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE attendance_records SET status='approved', approved_by=$3, approved_at=now(), updated_at=now()
		WHERE tenant_id=$1 AND id=$2 AND status='pending' AND clock_out IS NOT NULL`,
		tenantID, id, approverUserID)
	if err != nil {
		return fmt.Errorf("failed to approve attendance: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.Validation("only completed, pending records can be approved")
	}
	return nil
}
