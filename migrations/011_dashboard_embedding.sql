-- Per-dashboard iframe publication is opt-in and can only be configured by administrators.
ALTER TABLE dashboards ADD COLUMN public_embed_enabled INTEGER NOT NULL DEFAULT 0;
