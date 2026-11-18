package dashboard

import (
	"encoding/json"
	"net/http"
	"strings"

	"9router/proxy/internal/db"
	"9router/proxy/internal/handlerutil"
	"9router/proxy/internal/providers"
)

// LimitsTargetOption represents a selectable model or combo target in the limits view.
type LimitsTargetOption struct {
	Type        string `json:"type"`        // "model" | "combo"
	ID          string `json:"id"`          // "gpt-4o" or combo ID
	Name        string `json:"name"`        // Display name
	ProviderID  string `json:"providerId"`  // Provider ID if model
}

// LimitsResponse encapsulates the limits list and selectable target options.
type LimitsResponse struct {
	Limits  []db.ModelLimit      `json:"limits"`
	Targets []LimitsTargetOption `json:"targets"`
}

// HandleGetLimits handles GET /api/limits.
func (h *DashboardHandler) HandleGetLimits(w http.ResponseWriter, r *http.Request) {
	limits, err := h.Repo.ListModelLimits()
	if err != nil {
		handlerutil.WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var targets []LimitsTargetOption

	// Add combos
	combos, err := h.Repo.GetCombos()
	if err == nil {
		for _, c := range combos {
			if c != nil {
				targets = append(targets, LimitsTargetOption{
					Type: "combo",
					ID:   c.ID,
					Name: "Combo: " + c.Name,
				})
			}
		}
	}

	// Add connected provider models
	conns, err := h.Repo.GetProviderConnections("", false)
	if err == nil {
		activeProvMap := make(map[string]bool)
		for _, c := range conns {
			if c != nil && c.IsActive == 1 {
				activeProvMap[c.Provider] = true
			}
		}

		for provID := range activeProvMap {
			provMeta, ok := providers.ProviderRegistry[provID]
			provName := provID
			if ok && provMeta.Name != "" {
				provName = provMeta.Name
			}

			// Add registered models
			for _, m := range providers.GetProviderModels(provID) {
				targets = append(targets, LimitsTargetOption{
					Type:       "model",
					ID:         m,
					Name:       provName + " / " + m,
					ProviderID: provID,
				})
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(LimitsResponse{
		Limits:  limits,
		Targets: targets,
	})
}

// HandlePostLimits handles POST /api/limits (upsert).
func (h *DashboardHandler) HandlePostLimits(w http.ResponseWriter, r *http.Request) {
	var req db.ModelLimit
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handlerutil.WriteJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.TargetType = strings.TrimSpace(req.TargetType)
	req.TargetID = strings.TrimSpace(req.TargetID)
	if req.TargetType == "" || req.TargetID == "" {
		handlerutil.WriteJSONError(w, http.StatusBadRequest, "targetType and targetId are required")
		return
	}

	if req.ResetWindow == "" {
		req.ResetWindow = "lifetime"
	}
	if req.OnExhaustion == "" {
		req.OnExhaustion = "error"
	}

	if err := h.Repo.UpsertModelLimit(req); err != nil {
		handlerutil.WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "limit": req})
}

// HandleRefillLimits handles POST /api/limits/refill.
func (h *DashboardHandler) HandleRefillLimits(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID          string `json:"id"`
		TargetType  string `json:"targetType"`
		TargetID    string `json:"targetId"`
		TokensToAdd int64  `json:"tokensToAdd"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handlerutil.WriteJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	limitID := req.ID
	if limitID == "" && req.TargetType != "" && req.TargetID != "" {
		lim, err := h.Repo.GetModelLimit(req.TargetType, req.TargetID)
		if err == nil && lim != nil {
			limitID = lim.ID
		}
	}

	if limitID == "" || req.TokensToAdd <= 0 {
		handlerutil.WriteJSONError(w, http.StatusBadRequest, "valid limit id and positive tokensToAdd required")
		return
	}

	if err := h.Repo.RefillModelLimitTokens(limitID, req.TokensToAdd); err != nil {
		handlerutil.WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

// HandleToggleLimits handles POST /api/limits/toggle.
func (h *DashboardHandler) HandleToggleLimits(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID       string `json:"id"`
		IsActive bool   `json:"isActive"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handlerutil.WriteJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ID == "" {
		handlerutil.WriteJSONError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := h.Repo.ToggleModelLimit(req.ID, req.IsActive); err != nil {
		handlerutil.WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

// HandleDeleteLimits handles DELETE /api/limits.
func (h *DashboardHandler) HandleDeleteLimits(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	targetType := r.URL.Query().Get("targetType")
	targetID := r.URL.Query().Get("targetId")

	if id == "" && targetType != "" && targetID != "" {
		lim, err := h.Repo.GetModelLimit(targetType, targetID)
		if err == nil && lim != nil {
			id = lim.ID
		}
	}

	if id == "" {
		handlerutil.WriteJSONError(w, http.StatusBadRequest, "limit id or targetType+targetId required")
		return
	}

	if err := h.Repo.DeleteModelLimit(id); err != nil {
		handlerutil.WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}
