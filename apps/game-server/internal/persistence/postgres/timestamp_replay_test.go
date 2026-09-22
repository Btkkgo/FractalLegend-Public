package postgres

import (
	"context"
	"os"
	"reflect"
	"testing"
	"time"

	"fractallegend/game-server/internal/contribution"
	"fractallegend/game-server/internal/systemspend"
)

func requireExactReplay[T any](t *testing.T, name string, first, replay T, original, persisted time.Time) {
	t.Helper()
	if reflect.DeepEqual(first, replay) {
		return
	}
	t.Errorf("%s exact mismatch:\nfirst=%#v\nreplay=%#v\noriginal=%s (unix=%d ns=%d location=%s)\npersisted/reloaded=%s (unix=%d ns=%d location=%s)\nprecision difference=%s", name, first, replay,
		original.Format(time.RFC3339Nano), original.Unix(), original.Nanosecond(), original.Location(),
		persisted.Format(time.RFC3339Nano), persisted.Unix(), persisted.Nanosecond(), persisted.Location(),
		persisted.Sub(original))
}

func requireCanonicalTimestamp(t *testing.T, name string, value time.Time) {
	t.Helper()
	if value.Location() != time.UTC || value.Nanosecond()%1000 != 0 {
		t.Errorf("%s is not UTC microsecond canonical: %s (unix=%d ns=%d location=%s)", name, value.Format(time.RFC3339Nano), value.Unix(), value.Nanosecond(), value.Location())
	}
}

func TestG13ExactPostgresAccountTimestamp(t *testing.T) {
	_, service, _ := g13Fixture(t)
	ctx := context.Background()
	first, err := service.CreateAccount(ctx, "pg-player-c")
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := service.Account(ctx, "pg-player-c")
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reopened.Close)
	afterRestart, err := contribution.NewService(reopened).Account(ctx, "pg-player-c")
	if err != nil {
		t.Fatal(err)
	}
	requireExactReplay(t, "G13 account reload", first, loaded, first.CreatedAt, loaded.CreatedAt)
	requireExactReplay(t, "G13 account restart", first, afterRestart, first.CreatedAt, afterRestart.CreatedAt)
	requireCanonicalTimestamp(t, "G13 account created", first.CreatedAt)
	requireCanonicalTimestamp(t, "G13 account updated", first.UpdatedAt)
}

func TestG13ExactPostgresTimestampReplay(t *testing.T) {
	store, service, _ := g13Fixture(t)
	ctx := context.Background()
	request := g13Request("pg-player-a", "pg-fb-a", "g13-exact-timestamp", 100)
	first, err := service.PostSystemSpend(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	same, err := service.PostSystemSpend(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reopened.Close)
	restarted := contribution.NewService(reopened)
	afterRestart, err := restarted.PostSystemSpend(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := restarted.Entries(ctx, request.PlayerID)
	if err != nil || len(entries) != 1 {
		t.Fatalf("entries=%+v err=%v", entries, err)
	}
	audit, err := restarted.AuditEvents(ctx, request.PlayerID)
	if err != nil || len(audit) != 1 {
		t.Fatalf("audit=%+v err=%v", audit, err)
	}
	ledgerSnapshot, err := store.LoadLedgerTransaction(ctx, first.FBTransaction.ID)
	if err != nil {
		t.Fatal(err)
	}
	ledgerAudit, err := store.LedgerAuditEvents(ctx, first.FBTransaction.ID)
	if err != nil || len(ledgerAudit) != 1 {
		t.Fatalf("ledger audit=%+v err=%v", ledgerAudit, err)
	}
	requireExactReplay(t, "G13 same-process posting", first, same, first.Entry.CreatedAt, same.Entry.CreatedAt)
	requireExactReplay(t, "G13 restart posting", first, afterRestart, first.Entry.CreatedAt, afterRestart.Entry.CreatedAt)
	requireExactReplay(t, "G13 entry snapshot", first.Entry, entries[0], first.Entry.CreatedAt, entries[0].CreatedAt)
	requireExactReplay(t, "G13 audit timestamp", first.Entry.CreatedAt, audit[0].CreatedAt, first.Entry.CreatedAt, audit[0].CreatedAt)
	requireExactReplay(t, "G13 ledger snapshot", first.FBTransaction, ledgerSnapshot, first.FBTransaction.CreatedAt, ledgerSnapshot.CreatedAt)
	requireExactReplay(t, "G13 ledger audit timestamp", first.FBTransaction.CreatedAt, ledgerAudit[0].OccurredAt, first.FBTransaction.CreatedAt, ledgerAudit[0].OccurredAt)
	requireCanonicalTimestamp(t, "G13 posting", first.Entry.CreatedAt)
}

func TestG14ExactPostgresTimestampReplay(t *testing.T) {
	for _, reversal := range []bool{false, true} {
		name := "refund"
		if reversal {
			name = "reversal"
		}
		t.Run(name, func(t *testing.T) {
			_, service, _ := g13Fixture(t)
			ctx := context.Background()
			original, err := service.PostSystemSpend(ctx, g13Request("pg-player-a", "pg-fb-a", "g14-exact-original", 100))
			if err != nil {
				t.Fatal(err)
			}
			request := g14Refund(original, "g14-exact-"+name, 40)
			request.Reversal = reversal
			first, err := service.RefundSystemSpend(ctx, request)
			if err != nil {
				t.Fatal(err)
			}
			same, err := service.RefundSystemSpend(ctx, request)
			if err != nil {
				t.Fatal(err)
			}
			reopened, err := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL"))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(reopened.Close)
			restarted := contribution.NewService(reopened)
			afterRestart, err := restarted.RefundSystemSpend(ctx, request)
			if err != nil {
				t.Fatal(err)
			}
			compensations, err := restarted.Compensations(ctx, request.PlayerID)
			if err != nil || len(compensations) != 1 {
				t.Fatalf("compensations=%+v err=%v", compensations, err)
			}
			ledgerSnapshot, err := reopened.LoadLedgerTransaction(ctx, first.FBTransaction.ID)
			if err != nil {
				t.Fatal(err)
			}
			ledgerAudit, err := reopened.LedgerAuditEvents(ctx, first.FBTransaction.ID)
			if err != nil || len(ledgerAudit) != 1 {
				t.Fatalf("ledger audit=%+v err=%v", ledgerAudit, err)
			}
			requireExactReplay(t, "G14 same-process "+name, first, same, first.Compensation.CreatedAt, same.Compensation.CreatedAt)
			requireExactReplay(t, "G14 restart "+name, first, afterRestart, first.Compensation.CreatedAt, afterRestart.Compensation.CreatedAt)
			requireExactReplay(t, "G14 compensation snapshot "+name, first.Compensation, compensations[0], first.Compensation.CreatedAt, compensations[0].CreatedAt)
			requireExactReplay(t, "G14 ledger snapshot "+name, first.FBTransaction, ledgerSnapshot, first.FBTransaction.CreatedAt, ledgerSnapshot.CreatedAt)
			requireExactReplay(t, "G14 ledger audit timestamp "+name, first.FBTransaction.CreatedAt, ledgerAudit[0].OccurredAt, first.FBTransaction.CreatedAt, ledgerAudit[0].OccurredAt)
			requireCanonicalTimestamp(t, "G14 "+name, first.Compensation.CreatedAt)
		})
	}
}

func TestG15ExactPostgresTimestampReplay(t *testing.T) {
	for _, eligible := range []bool{false, true} {
		name := "noneligible"
		if eligible {
			name = "eligible"
		}
		t.Run(name, func(t *testing.T) {
			store, _, _ := g13Fixture(t)
			ctx := context.Background()
			service := systemspend.NewService(store, g15Registry(t))
			intent := g15Intent("g15-exact-"+name, 50)
			if !eligible {
				intent.ProducerType = systemspend.ProducerInternalTestNonEligible
			}
			first, err := service.Post(ctx, intent)
			if err != nil {
				t.Fatal(err)
			}
			same, err := service.Post(ctx, intent)
			if err != nil {
				t.Fatal(err)
			}
			reopened, err := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL"))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(reopened.Close)
			restarted := systemspend.NewService(reopened, g15Registry(t))
			afterRestart, err := restarted.Post(ctx, intent)
			if err != nil {
				t.Fatal(err)
			}
			loaded, err := restarted.Load(ctx, intent.OperationID)
			if err != nil {
				t.Fatal(err)
			}
			requireExactReplay(t, "G15 same-process "+name, first, same, first.CreatedAt, same.CreatedAt)
			requireExactReplay(t, "G15 restart "+name, first, afterRestart, first.CreatedAt, afterRestart.CreatedAt)
			requireExactReplay(t, "G15 snapshot "+name, first, loaded, first.CreatedAt, loaded.CreatedAt)
			requireCanonicalTimestamp(t, "G15 created "+name, first.CreatedAt)
			requireCanonicalTimestamp(t, "G15 completed "+name, first.CompletedAt)
		})
	}
}
