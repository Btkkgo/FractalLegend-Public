package postgres

import (
	"os"
	"strings"
	"testing"
)

func TestFBLedgerMigrationHasNoProductionFundingTransactionType(t *testing.T) {
	data, err := os.ReadFile("migrations/0003_fb_ledger.sql")
	if err != nil {
		t.Fatal(err)
	}
	upper := strings.ToUpper(string(data))
	for _, forbidden := range []string{"EXTERNAL_CREDIT", "ADMIN_ADJUSTMENT"} {
		if strings.Contains(upper, forbidden) {
			t.Errorf("production migration exposes forbidden funding type %s", forbidden)
		}
	}
	if !strings.Contains(upper, "CURRENCY TEXT NOT NULL DEFAULT 'FB' CHECK (CURRENCY = 'FB')") {
		t.Error("production migration does not constrain account currency to FB")
	}
	if !strings.Contains(upper, "CONSTRAINT TRIGGER FB_LEDGER_TRANSACTION_BALANCED") || !strings.Contains(upper, "DEFERRABLE INITIALLY DEFERRED") {
		t.Error("production migration does not defer a database conservation check until transaction commit")
	}
}
