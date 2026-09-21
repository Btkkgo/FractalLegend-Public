package ledger

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"
)

func TestProductionLedgerExportsNoDirectFundingSurface(t *testing.T) {
	t.Helper()
	fset := token.NewFileSet()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	forbidden := map[string]struct{}{
		"ExternalCreditService":    {},
		"ExternalCreditRequest":    {},
		"ExternalCreditAuthorizer": {},
		"MintFB":                   {},
		"GrantFB":                  {},
		"SetBalance":               {},
		"AddBalance":               {},
		"AdminSetBalance":          {},
	}
	for _, path := range files {
		if filepath.Ext(path) != ".go" || filepath.Base(path) == "production_surface_test.go" || len(path) >= 8 && path[len(path)-8:] == "_test.go" {
			continue
		}
		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			switch value := node.(type) {
			case *ast.TypeSpec:
				if _, found := forbidden[value.Name.Name]; found {
					t.Errorf("production funding type exists: %s", value.Name.Name)
				}
			case *ast.FuncDecl:
				if _, found := forbidden[value.Name.Name]; found {
					t.Errorf("production funding function exists: %s", value.Name.Name)
				}
			}
			return true
		})
	}
}

func TestPostDraftRejectsOneSidedBalanceCreation(t *testing.T) {
	draft := PostDraft{
		ID:        "tx-one-sided",
		Type:      TransactionType("EXTERNAL_CREDIT"),
		Reference: LedgerReference{Type: "TEST", ID: "one-sided"},
		Entries:   []EntryDraft{{ID: "entry-one-sided", AccountID: "fb-a", Amount: 1}},
	}
	if err := ValidatePostDraft(draft); !errors.Is(err, ErrInvalidTransaction) {
		t.Fatalf("one-sided balance creation was accepted: %v", err)
	}
}
