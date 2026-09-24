-- One row per /api/greeting hit; /api/stats reports the count.
CREATE TABLE greeting_log (
    id bigserial PRIMARY KEY,
    greeting text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
