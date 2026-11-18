//go:build integration

package integration

import (
	"net/http"
	"slices"
	"testing"
)

// TestComboReorderAPI pins the drag-to-reorder contract the dashboard posts
// (cheaprouter POST /api/combos/reorder): the ids name the full combo set in
// the new order, the answer carries the reordered combos, and a refusal
// leaves the stored order untouched.
func TestComboReorderAPI(t *testing.T) {
	env := newEnv(t)

	env.AddCombo(t, "combo-a", "reorder-a", []string{"deepseek/deepseek-chat"})
	env.AddCombo(t, "combo-b", "reorder-b", []string{"deepseek/deepseek-chat"})
	env.AddCombo(t, "combo-c", "reorder-c", []string{"deepseek/deepseek-chat"})

	names := func(t *testing.T, res *Result) []string {
		t.Helper()
		var out []struct {
			Name string `json:"name"`
		}
		res.Decode(t, &out)
		names := make([]string, 0, len(out))
		for _, combo := range out {
			names = append(names, combo.Name)
		}
		return names
	}

	// The full set, rearranged: c, a, b.
	res := env.Post(t, "/api/combos/reorder", map[string]any{
		"comboIds": []string{"combo-c", "combo-a", "combo-b"},
	})
	if res.Status != http.StatusOK {
		t.Fatalf("POST /api/combos/reorder = %d, want 200 (body: %s)", res.Status, truncate(res.Body))
	}
	var body struct {
		Combos []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"combos"`
	}
	res.Decode(t, &body)
	got := make([]string, 0, len(body.Combos))
	for _, combo := range body.Combos {
		got = append(got, combo.Name)
	}
	if want := []string{"reorder-c", "reorder-a", "reorder-b"}; !slices.Equal(got, want) {
		t.Fatalf("reordered combos = %v, want %v", got, want)
	}

	// The list route agrees with the write route.
	list := env.Get(t, "/api/combos")
	if list.Status != http.StatusOK {
		t.Fatalf("GET /api/combos = %d, want 200 (body: %s)", list.Status, truncate(list.Body))
	}
	if got := names(t, list); !slices.Equal(got, []string{"reorder-c", "reorder-a", "reorder-b"}) {
		t.Fatalf("GET /api/combos order = %v, want [reorder-c reorder-a reorder-b]", got)
	}

	// A stale client that misses a combo is refused, and nothing moves.
	bad := env.Post(t, "/api/combos/reorder", map[string]any{
		"comboIds": []string{"combo-a"},
	})
	if bad.Status != http.StatusBadRequest {
		t.Fatalf("POST /api/combos/reorder with a subset = %d, want 400 (body: %s)", bad.Status, truncate(bad.Body))
	}
	after := env.Get(t, "/api/combos")
	if got := names(t, after); !slices.Equal(got, []string{"reorder-c", "reorder-a", "reorder-b"}) {
		t.Fatalf("order after refused reorder = %v, want the stored order untouched", got)
	}
}
