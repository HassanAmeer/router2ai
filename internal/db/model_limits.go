package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ModelLimit represents a token/rate budget on a model or combo ("Limits Bank").
type ModelLimit struct {
	ID                 string `json:"id"`
	TargetType         string `json:"targetType"` // "model" | "combo"
	TargetID           string `json:"targetId"`
	TargetName         string `json:"targetName"`
	ProviderID         string `json:"providerId"`
	TotalTokenPool     int64  `json:"totalTokenPool"`
	ConsumedTokens     int64  `json:"consumedTokens"`
	RPMLimit           int    `json:"rpmLimit"`
	TPMLimit           int    `json:"tpmLimit"`
	ResetWindow        string `json:"resetWindow"` // "lifetime" | "monthly" | "weekly" | "daily"
	OnExhaustion       string `json:"onExhaustion"` // "error" | "fallback"
	FallbackTarget     string `json:"fallbackTarget"`
	CustomErrorMessage string `json:"customErrorMessage"`
	IsActive           bool   `json:"isActive"`
	CreatedAt          string `json:"createdAt"`
	UpdatedAt          string `json:"updatedAt"`
}

// ListModelLimits retrieves all model limit policies.
func (r *Repo) ListModelLimits() ([]ModelLimit, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("model_limits: nil repo")
	}

	rows, err := r.db.Query(
		`SELECT id, target_type, target_id, target_name, provider_id, total_token_pool, consumed_tokens,
		        rpm_limit, tpm_limit, reset_window, on_exhaustion, fallback_target, custom_error_message,
		        is_active, created_at, updated_at
		 FROM model_limits ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list model_limits: %w", err)
	}
	defer rows.Close()

	var list []ModelLimit
	for rows.Next() {
		var ml ModelLimit
		var activeInt int
		if err := rows.Scan(
			&ml.ID, &ml.TargetType, &ml.TargetID, &ml.TargetName, &ml.ProviderID,
			&ml.TotalTokenPool, &ml.ConsumedTokens, &ml.RPMLimit, &ml.TPMLimit,
			&ml.ResetWindow, &ml.OnExhaustion, &ml.FallbackTarget, &ml.CustomErrorMessage,
			&activeInt, &ml.CreatedAt, &ml.UpdatedAt,
		); err != nil {
			return nil, err
		}
		ml.IsActive = activeInt == 1
		list = append(list, ml)
	}
	return list, rows.Err()
}

// GetModelLimit retrieves one limit policy by targetType and targetID.
func (r *Repo) GetModelLimit(targetType, targetID string) (*ModelLimit, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("model_limits: nil repo")
	}

	var ml ModelLimit
	var activeInt int
	err := r.db.QueryRow(
		`SELECT id, target_type, target_id, target_name, provider_id, total_token_pool, consumed_tokens,
		        rpm_limit, tpm_limit, reset_window, on_exhaustion, fallback_target, custom_error_message,
		        is_active, created_at, updated_at
		 FROM model_limits WHERE target_type = ? AND target_id = ? LIMIT 1`,
		targetType, targetID,
	).Scan(
		&ml.ID, &ml.TargetType, &ml.TargetID, &ml.TargetName, &ml.ProviderID,
		&ml.TotalTokenPool, &ml.ConsumedTokens, &ml.RPMLimit, &ml.TPMLimit,
		&ml.ResetWindow, &ml.OnExhaustion, &ml.FallbackTarget, &ml.CustomErrorMessage,
		&activeInt, &ml.CreatedAt, &ml.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	ml.IsActive = activeInt == 1
	return &ml, nil
}

// UpsertModelLimit creates or updates a model limit policy.
func (r *Repo) UpsertModelLimit(ml ModelLimit) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("model_limits: nil repo")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if ml.ID == "" {
		ml.ID = "lim-" + uuid.NewString()
	}

	activeInt := 0
	if ml.IsActive {
		activeInt = 1
	}

	if ml.ResetWindow == "" {
		ml.ResetWindow = "lifetime"
	}
	if ml.OnExhaustion == "" {
		ml.OnExhaustion = "error"
	}

	_, err := r.db.Exec(
		`INSERT INTO model_limits (
			id, target_type, target_id, target_name, provider_id, total_token_pool, consumed_tokens,
			rpm_limit, tpm_limit, reset_window, on_exhaustion, fallback_target, custom_error_message,
			is_active, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(target_type, target_id) DO UPDATE SET
			target_name = excluded.target_name,
			provider_id = excluded.provider_id,
			total_token_pool = excluded.total_token_pool,
			rpm_limit = excluded.rpm_limit,
			tpm_limit = excluded.tpm_limit,
			reset_window = excluded.reset_window,
			on_exhaustion = excluded.on_exhaustion,
			fallback_target = excluded.fallback_target,
			custom_error_message = excluded.custom_error_message,
			is_active = excluded.is_active,
			updated_at = excluded.updated_at`,
		ml.ID, ml.TargetType, ml.TargetID, ml.TargetName, ml.ProviderID,
		ml.TotalTokenPool, ml.ConsumedTokens, ml.RPMLimit, ml.TPMLimit,
		ml.ResetWindow, ml.OnExhaustion, ml.FallbackTarget, ml.CustomErrorMessage,
		activeInt, now, now,
	)
	return err
}

// RefillModelLimitTokens adds more tokens to the target's total pool.
func (r *Repo) RefillModelLimitTokens(id string, addTokens int64) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("model_limits: nil repo")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.Exec(
		`UPDATE model_limits SET total_token_pool = total_token_pool + ?, updated_at = ? WHERE id = ?`,
		addTokens, now, id,
	)
	return err
}

// ToggleModelLimit toggles the active/paused status of a limit.
func (r *Repo) ToggleModelLimit(id string, active bool) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("model_limits: nil repo")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	activeInt := 0
	if active {
		activeInt = 1
	}
	_, err := r.db.Exec(
		`UPDATE model_limits SET is_active = ?, updated_at = ? WHERE id = ?`,
		activeInt, now, id,
	)
	return err
}

// DeleteModelLimit deletes a limit policy.
func (r *Repo) DeleteModelLimit(id string) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("model_limits: nil repo")
	}
	_, err := r.db.Exec(`DELETE FROM model_limits WHERE id = ?`, id)
	return err
}

// RecordModelLimitUsage increments consumed tokens for a target.
func (r *Repo) RecordModelLimitUsage(targetType, targetID string, tokens int64) error {
	if r == nil || r.db == nil || tokens <= 0 {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.Exec(
		`UPDATE model_limits SET consumed_tokens = consumed_tokens + ?, updated_at = ?
		 WHERE target_type = ? AND target_id = ? AND is_active = 1`,
		tokens, now, targetType, targetID,
	)
	return err
}
