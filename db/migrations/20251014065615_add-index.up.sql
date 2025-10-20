-- Create index "ix_account_transactions_signer" to table: "account_transactions"
CREATE INDEX "ix_account_transactions_signer" ON "public"."account_transactions" ("account_id", "is_signer", "block_height" DESC);
-- Create index "ix_transactions_move_ops" to table: "transactions"
CREATE INDEX "ix_transactions_move_ops" ON "public"."transactions" ("is_move_publish", "is_move_upgrade", "is_move_execute", "is_move_script");
-- Create index "ix_transactions_opinit" to table: "transactions"
CREATE INDEX "ix_transactions_opinit" ON "public"."transactions" ("is_opinit");
-- Create index "ix_transactions_send_ibc" to table: "transactions"
CREATE INDEX "ix_transactions_send_ibc" ON "public"."transactions" ("is_send", "is_ibc");
