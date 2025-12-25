CREATE TABLE "bills" (
  "id" uuid PRIMARY KEY,
  "user_id" uuid,
  "balance" decimal(15,2) NOT NULL DEFAULT 0,
  "currency" varchar(3) NOT NULL DEFAULT 'RUB',
  "status" bill_status NOT NULL DEFAULT 'active',
  "created_at" timestamp DEFAULT (now()),
  "updated_at" timestamp,
  "closed_at" timestamp
);