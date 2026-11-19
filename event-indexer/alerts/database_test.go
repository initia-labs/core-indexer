package alerts

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	coredb "github.com/initia-labs/core-indexer/pkg/db"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Use a disposable PostgreSQL database; each test creates and drops its own schema.
func alertDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("ALERT_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("requires disposable PostgreSQL: ALERT_TEST_DATABASE_URL")
	}
	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("alert_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE SCHEMA " + schema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Exec("DROP SCHEMA " + schema + " CASCADE"); admin.Close() })
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		u, err := url.Parse(dsn)
		if err != nil {
			t.Fatal(err)
		}
		q := u.Query()
		q.Set("search_path", schema)
		u.RawQuery = q.Encode()
		dsn = u.String()
	} else {
		dsn += " search_path=" + schema
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(6)
	t.Cleanup(func() { db.Close() })
	for _, name := range []string{"20261118000000_transfer_alerts", "20261118010000_transfer_alert_batches", "20261118020000_transfer_alert_failures", "20261119000000_transfer_alert_rpc_cursors"} {
		data, err := os.ReadFile("../../db/migrations/" + name + ".up.sql")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(string(data)); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func putBatch(t *testing.T, db *sql.DB, height int, hash string, events ...abci.Event) {
	t.Helper()
	data, err := json.Marshal(events)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO transfer_alert_batches (chain_id, block_height, transaction_hash, events)
		VALUES ('test-1', $1, $2, $3) ON CONFLICT DO NOTHING`, height, hash, string(data)); err != nil {
		t.Fatal(err)
	}
}

func TestDBBatchAtomicityLateBlocksAndReplay(t *testing.T) {
	db := alertDB(t)
	ctx := context.Background()
	c := testConfig(t)
	c.BatchSize = 2
	e := packet("channel-2", "transfer/channel-2/uusdc", "60000000000")
	putBatch(t, db, 42, "abc", e, e)
	putBatch(t, db, 43, "def", e) // Below threshold still completes evaluation.
	putBatch(t, db, 44, "ghi", e, e)
	if n, err := scanBatch(ctx, db, c); err != nil || n != 2 {
		t.Fatalf("first batch: %d %v", n, err)
	}
	putBatch(t, db, 41, "late", e, e) // Arrives after a higher block was processed.
	if n, err := scanBatch(ctx, db, c); err != nil || n != 2 {
		t.Fatalf("late batch: %d %v", n, err)
	}
	putBatch(t, db, 42, "abc", e, e)
	if n, err := scanBatch(ctx, db, c); err != nil || n != 0 {
		t.Fatalf("replay requeued work: %d %v", n, err)
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM transfer_alerts WHERE message LIKE '%Amount: 120000 USDC%'`).Scan(&count); err != nil || count != 3 {
		t.Fatalf("aggregated alerts: %d %v", count, err)
	}
	// A malformed group is recorded without blocking its valid neighbor.
	putBatch(t, db, 45, "good", e, e)
	bad := event("send_packet", "packet_src_port", "transfer", "packet_src_channel", "channel-2", "packet_data", "invalid JSON")
	putBatch(t, db, 46, "bad", bad)
	if n, err := scanBatch(ctx, db, c); err != nil || n != 2 {
		t.Fatalf("failure isolation: %d %v", n, err)
	}
	var failure string
	if err := db.QueryRow(`SELECT evaluation_error FROM transfer_alert_batches WHERE block_height = 46 AND evaluated_at IS NULL`).Scan(&failure); err != nil || failure == "" {
		t.Fatalf("evaluation failure missing: %q %v", failure, err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM transfer_alerts`).Scan(&count); err != nil || count != 4 {
		t.Fatalf("valid neighbor lost: %d %v", count, err)
	}
	if n, err := scanBatch(ctx, db, c); err != nil || n != 0 {
		t.Fatalf("failed group was automatically retried: %d %v", n, err)
	}
	fixed, _ := json.Marshal([]abci.Event{e, e})
	if _, err := db.Exec(`UPDATE transfer_alert_batches SET events = $1, evaluation_error = NULL WHERE block_height = 46`, string(fixed)); err != nil {
		t.Fatal(err)
	}
	if n, err := scanBatch(ctx, db, c); err != nil || n != 1 {
		t.Fatalf("corrected group replay: %d %v", n, err)
	}
	// A persistence error still rolls back every group in the DB transaction.
	putBatch(t, db, 47, "good", e, e)
	putBatch(t, db, 48, "fail-db", e, e)
	if _, err := db.Exec(`ALTER TABLE transfer_alerts ADD CONSTRAINT reject_test_alert CHECK (id <> 'test-1/48/tx/fail-db')`); err != nil {
		t.Fatal(err)
	}
	if _, err := scanBatch(ctx, db, c); err == nil {
		t.Fatal("DB error was lost")
	}
	if err := db.QueryRow(`SELECT count(*) FROM transfer_alert_batches WHERE block_height >= 47 AND evaluated_at IS NULL`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("partial evaluation committed: %d %v", count, err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM transfer_alerts`).Scan(&count); err != nil || count != 5 {
		t.Fatalf("partial alert committed: %d %v", count, err)
	}
}

func TestDBWorkersSkipLockedBatches(t *testing.T) {
	db := alertDB(t)
	ctx := context.Background()
	c := testConfig(t)
	e := packet("channel-2", "transfer/channel-2/uusdc", "100000000001")
	putBatch(t, db, 42, "locked", e)
	putBatch(t, db, 43, "available", e)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`SELECT 1 FROM transfer_alert_batches WHERE block_height = 42 FOR UPDATE`); err != nil {
		t.Fatal(err)
	}
	bounded, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if n, err := scanBatch(bounded, db, c); err != nil || n != 1 {
		t.Fatalf("worker blocked or took locked job: %d %v", n, err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if n, err := scanBatch(ctx, db, c); err != nil || n != 1 {
		t.Fatalf("locked job was lost: %d %v", n, err)
	}
}

func TestDBIndexerPublishesOnlyCommittedGroups(t *testing.T) {
	db := alertDB(t)
	gormDB, err := gorm.Open(gormpostgres.New(gormpostgres.Config{Conn: db}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	b := block(packet("channel-2", "transfer/channel-2/uusdc", "100000000001"))
	tx := gormDB.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	if err := coredb.InsertTransferAlertBatches(ctx, tx, "test-1", &b); err != nil {
		t.Fatal(err)
	}
	if n, err := scanBatch(ctx, db, testConfig(t)); err != nil || n != 0 {
		t.Fatalf("uncommitted group visible: %d %v", n, err)
	}
	if err := tx.Rollback().Error; err != nil {
		t.Fatal(err)
	}
	if n, err := scanBatch(ctx, db, testConfig(t)); err != nil || n != 0 {
		t.Fatalf("rolled-back group visible: %d %v", n, err)
	}
	if err := gormDB.Transaction(func(tx *gorm.DB) error { return coredb.InsertTransferAlertBatches(ctx, tx, "test-1", &b) }); err != nil {
		t.Fatal(err)
	}
	if n, err := scanBatch(ctx, db, testConfig(t)); err != nil || n != 1 {
		t.Fatalf("committed group missing: %d %v", n, err)
	}
	if err := gormDB.Transaction(func(tx *gorm.DB) error { return coredb.InsertTransferAlertBatches(ctx, tx, "test-1", &b) }); err != nil {
		t.Fatal(err)
	}
	if n, err := scanBatch(ctx, db, testConfig(t)); err != nil || n != 0 {
		t.Fatalf("producer replay reset processed state: %d %v", n, err)
	}
}

func TestDBDeliveryRetryAndConcurrentSenders(t *testing.T) {
	db := alertDB(t)
	ctx := context.Background()
	for _, id := range []string{"test-1/a", "test-1/b", "other-chain/c"} {
		if _, err := db.Exec(`INSERT INTO transfer_alerts (id, chain_id, message) VALUES ($1, split_part($1, '/', 1), $1)`, id); err != nil {
			t.Fatal(err)
		}
	}
	started, release := make(chan struct{}), make(chan struct{})
	defer close(release)
	done := make(chan error, 1)
	go func() {
		_, err := deliverNext(ctx, db, "test-1", func(context.Context, string) error {
			close(started)
			<-release
			return errors.New("send failed")
		})
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("sender did not start")
	}
	bounded, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var sent []string
	send := func(_ context.Context, text string) error { sent = append(sent, text); return nil }
	if ok, err := deliverNext(bounded, db, "test-1", send); err != nil || !ok {
		t.Fatalf("second sender blocked: %v %v", ok, err)
	}
	release <- struct{}{}
	if err := <-done; err == nil {
		t.Fatal("failed send was lost")
	}
	if ok, err := deliverNext(ctx, db, "test-1", send); err != nil || !ok {
		t.Fatalf("retry failed: %v %v", ok, err)
	}
	if len(sent) != 2 || sent[0] != "test-1/b" || sent[1] != "test-1/a" {
		t.Fatalf("wrong deliveries: %v", sent)
	}
	if ok, err := deliverNext(ctx, db, "test-1", send); err != nil || ok {
		t.Fatalf("repeated delivery or crossed chains: %v %v", ok, err)
	}
}

func TestDBDeliveryFailureDoesNotCancelEvaluation(t *testing.T) {
	db := alertDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := testConfig(t)
	putBatch(t, db, 42, "abc", packet("channel-2", "transfer/channel-2/uusdc", "120000000000"))
	if _, err := scanBatch(ctx, db, c); err != nil {
		t.Fatal(err)
	}
	putBatch(t, db, 43, "next", packet("channel-2", "transfer/channel-2/uusdc", "120000000000"))
	attempts := 0
	var delays []time.Duration
	err := runDelivery(ctx, db, c.Chain, func(context.Context, string) error {
		attempts++
		if attempts < 3 {
			return errors.New("Telegram unavailable")
		}
		return nil
	}, func(ctx context.Context, delay time.Duration) error {
		delays = append(delays, delay)
		if attempts == 1 {
			if n, err := scanBatch(ctx, db, c); err != nil || n != 1 {
				t.Fatalf("evaluation stopped during outage: %d %v", n, err)
			}
		}
		if attempts == 3 {
			cancel()
			return ctx.Err()
		}
		return nil
	})
	if !errors.Is(err, context.Canceled) || attempts != 3 || len(delays) != 3 || delays[0] != 3*time.Second || delays[1] != 6*time.Second || delays[2] != 3*time.Second {
		t.Fatalf("retry behavior: attempts=%d delays=%v err=%v", attempts, delays, err)
	}
}

func TestDBChainMigrationAndStoredPreview(t *testing.T) {
	db := alertDB(t)
	// Exercise migration of existing replay IDs, including a slash in the chain.
	for _, suffix := range []string{"down", "up"} {
		data, err := os.ReadFile("../../db/migrations/20261118020000_transfer_alert_failures." + suffix + ".sql")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(string(data)); err != nil {
			t.Fatal(err)
		}
		if suffix == "down" {
			for _, id := range []string{"rollup/dev-1/42/tx/hash", "rollup/42/tx/hash", "test-1/42/hash/0/IBC outbound/USDC"} {
				if _, err := db.Exec(`INSERT INTO transfer_alerts (id, message) VALUES ($1, $1)`, id); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	var chain string
	if err := db.QueryRow(`SELECT chain_id FROM transfer_alerts WHERE id = 'rollup/dev-1/42/tx/hash'`).Scan(&chain); err != nil || chain != "rollup/dev-1" {
		t.Fatalf("chain backfill: %q %v", chain, err)
	}
	if _, err := db.Exec(`UPDATE transfer_alerts SET telegram_message = 'canonical preview' WHERE chain_id = 'rollup/dev-1'`); err != nil {
		t.Fatal(err)
	}
	var message string
	ok, err := deliverNext(context.Background(), db, "rollup/dev-1", func(_ context.Context, text string) error { message = text; return nil })
	if err != nil || !ok || message != "canonical preview" {
		t.Fatalf("stored preview: %v %q %v", ok, message, err)
	}
	if ok, err := deliverNext(context.Background(), db, "rollup/dev-1", func(context.Context, string) error { t.Fatal("crossed chain boundary"); return nil }); err != nil || ok {
		t.Fatalf("chain filtering: %v %v", ok, err)
	}
}

func TestDBNullAttributesCanBeStored(t *testing.T) {
	db := alertDB(t)
	gormDB, err := gorm.Open(gormpostgres.New(gormpostgres.Config{Conn: db}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	b := block(packet("channel-2", "transfer/channel-2/uusdc", "120000000000"))
	b.FinalizeBlockEvents = []abci.Event{event("coin_spent", "spender", "escrow", "amount", "120000000000ibc/USDC", "note", "a\x00b")}
	if err := gormDB.Transaction(func(tx *gorm.DB) error {
		return coredb.InsertTransferAlertBatches(context.Background(), tx, "test-1", &b)
	}); err != nil {
		t.Fatal(err)
	}
	if n, err := scanBatch(context.Background(), db, testConfig(t)); err != nil || n != 2 {
		t.Fatalf("NUL batch lost: %d %v", n, err)
	}
}
