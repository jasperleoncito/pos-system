-- Subscription plans for the platform admin view.
ALTER TABLE tenants ADD COLUMN plan TEXT NOT NULL DEFAULT 'free'; -- free | standard | premium
