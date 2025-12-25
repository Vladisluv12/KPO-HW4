CREATE TABLE "outbox_messages" (
  "id" uuid PRIMARY KEY,
  "message_id" varchar(100) UNIQUE NOT NULL,
  "exchange" varchar(100) NOT NULL,
  "routing_key" varchar(100) NOT NULL,
  "payload" jsonb NOT NULL,
  "headers" jsonb,
  "status" outbox_status NOT NULL DEFAULT 'pending',
  "created_at" timestamp DEFAULT (now()),
  "sent_at" timestamp,
  "error" text,
  "retry_count" integer DEFAULT 0
);

CREATE UNIQUE INDEX ON "outbox_messages" ("message_id");

CREATE INDEX ON "outbox_messages" ("status", "created_at");

CREATE INDEX ON "outbox_messages" ("exchange");

CREATE INDEX ON "outbox_messages" ("created_at");
