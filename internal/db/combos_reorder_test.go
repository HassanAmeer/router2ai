package db

import (
	"errors"
	"testing"

	"9router/proxy/internal/models"
)

func seedThreeCombos(t *testing.T) *Repo {
	t.Helper()
	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)
	_, err := db.Exec(`INSERT INTO combos (id, name, kind, models, createdAt, updatedAt) VALUES
		('c1', 'combo-1', 'chat', '[]', '2026-07-18T00:00:01Z', '2026-07-18T00:00:01Z'),
		('c2', 'combo-2', 'chat', '[]', '2026-07-18T00:00:02Z', '2026-07-18T00:00:02Z'),
		('c3', 'combo-3', 'chat', '[]', '2026-07-18T00:00:03Z', '2026-07-18T00:00:03Z');`)
	if err != nil {
		t.Fatalf("failed to seed combos: %v", err)
	}
	return NewRepo(db)
}

func comboNameList(combos []*models.Combo) []string {
	names := make([]string, 0, len(combos))
	for _, combo := range combos {
		names = append(names, combo.Name)
	}
	return names
}

func assertOrder(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

// TestReorderCombos pins the dashboard drag-to-reorder contract
// (POST /api/combos/reorder, cheaprouter parity): the ids must be the full
// current set, the positions are rewritten in one transaction, and a
// validation failure leaves the stored order untouched.
func TestReorderCombos(t *testing.T) {
	t.Run("reorders and persists the new order", func(t *testing.T) {
		repo := seedThreeCombos(t)

		got, err := repo.ReorderCombos([]string{"c3", "c1", "c2"})
		if err != nil {
			t.Fatalf("ReorderCombos failed: %v", err)
		}
		assertOrder(t, comboNameList(got), []string{"combo-3", "combo-1", "combo-2"})

		// The read path agrees with the write path.
		stored, err := repo.GetCombos()
		if err != nil {
			t.Fatalf("GetCombos failed: %v", err)
		}
		assertOrder(t, comboNameList(stored), []string{"combo-3", "combo-1", "combo-2"})
	})

	t.Run("a freshly created combo appends after the reordered set", func(t *testing.T) {
		repo := seedThreeCombos(t)

		if _, err := repo.ReorderCombos([]string{"c3", "c1", "c2"}); err != nil {
			t.Fatalf("ReorderCombos failed: %v", err)
		}
		if err := repo.CreateCombo("c4", "combo-4", "chat", `[]`, "fallback", ""); err != nil {
			t.Fatalf("CreateCombo failed: %v", err)
		}

		stored, err := repo.GetCombos()
		if err != nil {
			t.Fatalf("GetCombos failed: %v", err)
		}
		got := comboNameList(stored)
		last := got[len(got)-1]
		if last != "combo-4" {
			t.Errorf("last combo = %q, want the newly created combo-4 (full order: %v)", last, got)
		}
	})

	t.Run("validation failures leave the stored order untouched", func(t *testing.T) {
		invalid := []struct {
			name string
			ids  []string
		}{
			{"empty list", nil},
			{"unknown id", []string{"c1", "nope"}},
			{"duplicate id", []string{"c1", "c1", "c2", "c3"}},
			{"missing id", []string{"c1", "c2"}},
		}
		for _, tc := range invalid {
			t.Run(tc.name, func(t *testing.T) {
				repo := seedThreeCombos(t)

				if _, err := repo.ReorderCombos(tc.ids); !errors.Is(err, ErrComboReorderInvalid) {
					t.Fatalf("ReorderCombos error = %v, want ErrComboReorderInvalid", err)
				}
				stored, err := repo.GetCombos()
				if err != nil {
					t.Fatalf("GetCombos failed: %v", err)
				}
				assertOrder(t, comboNameList(stored), []string{"combo-1", "combo-2", "combo-3"})
			})
		}
	})
}
