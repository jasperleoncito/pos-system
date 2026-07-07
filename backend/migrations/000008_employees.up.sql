-- Employees, weekly schedules, attendance (clock in/out) records.

CREATE TABLE employees (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants (id),
    user_id     UUID REFERENCES users (id), -- optional login link for self-service clock
    full_name   TEXT NOT NULL,
    position    TEXT NOT NULL DEFAULT '',
    phone       TEXT NOT NULL DEFAULT '',
    email       TEXT NOT NULL DEFAULT '',
    address     TEXT NOT NULL DEFAULT '',
    salary_type TEXT NOT NULL DEFAULT 'daily', -- hourly | daily | monthly
    salary_rate BIGINT NOT NULL DEFAULT 0, -- centavos per salary_type period
    hire_date   DATE,
    photo_path  TEXT NOT NULL DEFAULT '',
    thumb_path  TEXT NOT NULL DEFAULT '',
    notes       TEXT NOT NULL DEFAULT '',
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX idx_employees_tenant ON employees (tenant_id, full_name) WHERE deleted_at IS NULL;
-- A login account maps to at most one employee profile per tenant.
CREATE UNIQUE INDEX idx_employees_tenant_user ON employees (tenant_id, user_id)
    WHERE user_id IS NOT NULL AND deleted_at IS NULL;

CREATE TABLE employee_schedules (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants (id),
    employee_id   UUID NOT NULL REFERENCES employees (id),
    day_of_week   SMALLINT NOT NULL CHECK (day_of_week BETWEEN 0 AND 6), -- 0 = Sunday
    start_time    TIME NOT NULL,
    end_time      TIME NOT NULL,
    grace_minutes INT NOT NULL DEFAULT 10 CHECK (grace_minutes BETWEEN 0 AND 240),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (start_time < end_time)
);

CREATE UNIQUE INDEX idx_schedules_employee_day ON employee_schedules (tenant_id, employee_id, day_of_week);

CREATE TABLE attendance_records (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenants (id),
    employee_id       UUID NOT NULL REFERENCES employees (id),
    clock_in          TIMESTAMPTZ NOT NULL,
    clock_out         TIMESTAMPTZ,
    -- Schedule snapshot at clock-in so later edits never rewrite history.
    scheduled_start   TIMESTAMPTZ,
    scheduled_end     TIMESTAMPTZ,
    break_start       TIMESTAMPTZ, -- set while a break is running
    break_minutes     INT NOT NULL DEFAULT 0,
    late_minutes      INT NOT NULL DEFAULT 0,
    early_out_minutes INT NOT NULL DEFAULT 0,
    overtime_minutes  INT NOT NULL DEFAULT 0,
    status            TEXT NOT NULL DEFAULT 'pending', -- pending | approved
    approved_by       UUID,
    approved_at       TIMESTAMPTZ,
    notes             TEXT NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- One open shift per employee.
CREATE UNIQUE INDEX idx_attendance_open ON attendance_records (tenant_id, employee_id) WHERE clock_out IS NULL;
CREATE INDEX idx_attendance_tenant_time ON attendance_records (tenant_id, clock_in DESC);
CREATE INDEX idx_attendance_employee_time ON attendance_records (tenant_id, employee_id, clock_in DESC);
