package dashboard

import (
	"encoding/json"
	"net/http"
	"strings"

	"9router/proxy/internal/db"
	"9router/proxy/internal/handlerutil"
)

// UsersSummaryStats holds aggregation metrics for reseller users.
type UsersSummaryStats struct {
	TotalUsers     int   `json:"totalUsers"`
	ActiveUsers    int   `json:"activeUsers"`
	BannedUsers    int   `json:"bannedUsers"`
	TotalBudget    int64 `json:"totalBudget"`
	TotalTokensUsed int64 `json:"totalTokensUsed"`
}

// UsersResponse encapsulates the users list and summary stats.
type UsersResponse struct {
	Users []db.ResellerUser `json:"users"`
	Stats UsersSummaryStats `json:"stats"`
}

// HandleGetUsers handles GET /api/users.
func (h *DashboardHandler) HandleGetUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.Repo.ListResellerUsers()
	if err != nil {
		handlerutil.WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	stats := UsersSummaryStats{
		TotalUsers: len(users),
	}
	for _, u := range users {
		if u.IsBanned {
			stats.BannedUsers++
		} else {
			stats.ActiveUsers++
		}
		stats.TotalBudget += u.TokenBudget
		stats.TotalTokensUsed += u.TokensUsed
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(UsersResponse{
		Users: users,
		Stats: stats,
	})
}

// HandleCreateUser handles POST /api/users.
func (h *DashboardHandler) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req db.ResellerUser
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handlerutil.WriteJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.Name = strings.TrimSpace(req.Name)
	if req.Email == "" {
		handlerutil.WriteJSONError(w, http.StatusBadRequest, "email is required")
		return
	}
	if req.PlanTier == "" {
		req.PlanTier = "free"
	}

	user, err := h.Repo.CreateResellerUser(req)
	if err != nil {
		handlerutil.WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "user": user})
}

// HandleUpdateUser handles PUT/PATCH /api/users/{id} or POST /api/users/update.
func (h *DashboardHandler) HandleUpdateUser(w http.ResponseWriter, r *http.Request) {
	var req db.ResellerUser
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handlerutil.WriteJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ID == "" {
		// Try query param
		req.ID = r.URL.Query().Get("id")
	}
	if req.ID == "" {
		handlerutil.WriteJSONError(w, http.StatusBadRequest, "user id is required")
		return
	}

	if err := h.Repo.UpdateResellerUser(req); err != nil {
		handlerutil.WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

// HandleDeleteUser handles DELETE /api/users.
func (h *DashboardHandler) HandleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		handlerutil.WriteJSONError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := h.Repo.DeleteResellerUser(id); err != nil {
		handlerutil.WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}
