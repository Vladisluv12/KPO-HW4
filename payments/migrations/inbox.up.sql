CREATE TABLE IF NOT EXISTS "inbox_messages" (
  "id" uuid PRIMARY KEY,
  "message_id" varchar(100) UNIQUE NOT NULL,
  "queue" varchar(100) NOT NULL,
  "payload" jsonb NOT NULL,
  "processed" boolean NOT NULL DEFAULT false,
  "processed_at" timestamp,
  "created_at" timestamp DEFAULT (now())
);

CREATE UNIQUE INDEX ON "inbox_messages" ("message_id");

CREATE INDEX ON "inbox_messages" ("processed", "created_at");

CREATE INDEX ON "inbox_messages" ("queue");

CREATE INDEX ON "inbox_messages" ("created_at");