-- Expiry is UTC Unix milliseconds. NULL keeps legacy shares permanent and open.
ALTER TABLE dashboards ADD COLUMN public_expires_at BIGINT;
ALTER TABLE dashboards ADD COLUMN public_password_hash VARCHAR(255);
