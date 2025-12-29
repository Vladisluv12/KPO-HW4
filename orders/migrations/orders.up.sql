CREATE TABLE IF NOT EXISTS "orders" (
  "id" uuid PRIMARY KEY,
  "user_id" uuid NOT NULL,
  "price" decimal(10,2) NOT NULL,
  "description" text NOT NULL,
  "status" VARCHAR(20) NOT NULL DEFAULT 'new',
  "created_at" timestamp DEFAULT (now()),
  "updated_at" timestamp DEFAULT (now())
);

CREATE INDEX ON "orders" ("user_id");
CREATE INDEX ON "orders" ("status");
CREATE INDEX ON "orders" ("user_id", "status");
