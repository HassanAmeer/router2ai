package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ModelCustomization represents a per-model override ("Pipe Models" / Curation).
type ModelCustomization struct {
	ID              string  `json:"id"`
	ProviderID      string  `json:"providerId"`
	OriginalModelID string  `json:"originalModelId"`
	CustomName      string  `json:"customName"`
	CustomAlias     string  `json:"customAlias"`
	ContextWindow   int     `json:"contextWindow"`
	InputPrice      float64 `json:"inputPrice"`
	CachePrice      float64 `json:"cachePrice"`
	OutputPrice     float64 `json:"outputPrice"`
	IsActive        bool    `json:"isActive"`
	CreatedAt       string  `json:"createdAt"`
	UpdatedAt       string  `json:"updatedAt"`
}

// GetModelCustomizations returns all customizations for a given provider.
func (r *Repo) GetModelCustomizations(providerID string) ([]ModelCustomization, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("model_customizations: nil repo")
	}

	rows, err := r.db.Query(
		`SELECT id, provider_id, original_model_id, custom_name, custom_alias, context_window, is_active, created_at, updated_at,
		 COALESCE(input_price, 0), COALESCE(cache_price, 0), COALESCE(output_price, 0)
		 FROM model_customizations WHERE provider_id = ?`,
		providerID,
	)
	if err != nil {
		return nil, fmt.Errorf("query model_customizations for provider %s: %w", providerID, err)
	}
	defer rows.Close()

	var list []ModelCustomization
	for rows.Next() {
		var mc ModelCustomization
		var activeInt int
		if err := rows.Scan(
			&mc.ID, &mc.ProviderID, &mc.OriginalModelID, &mc.CustomName, &mc.CustomAlias,
			&mc.ContextWindow, &activeInt, &mc.CreatedAt, &mc.UpdatedAt,
			&mc.InputPrice, &mc.CachePrice, &mc.OutputPrice,
		); err != nil {
			return nil, err
		}
		mc.IsActive = activeInt == 1
		list = append(list, mc)
	}
	return list, rows.Err()
}

// GetAllModelCustomizations returns all customizations across all providers.
func (r *Repo) GetAllModelCustomizations() ([]ModelCustomization, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("model_customizations: nil repo")
	}

	rows, err := r.db.Query(
		`SELECT id, provider_id, original_model_id, custom_name, custom_alias, context_window, is_active, created_at, updated_at,
		 COALESCE(input_price, 0), COALESCE(cache_price, 0), COALESCE(output_price, 0)
		 FROM model_customizations`,
	)
	if err != nil {
		return nil, fmt.Errorf("query all model_customizations: %w", err)
	}
	defer rows.Close()

	var list []ModelCustomization
	for rows.Next() {
		var mc ModelCustomization
		var activeInt int
		if err := rows.Scan(
			&mc.ID, &mc.ProviderID, &mc.OriginalModelID, &mc.CustomName, &mc.CustomAlias,
			&mc.ContextWindow, &activeInt, &mc.CreatedAt, &mc.UpdatedAt,
			&mc.InputPrice, &mc.CachePrice, &mc.OutputPrice,
		); err != nil {
			return nil, err
		}
		mc.IsActive = activeInt == 1
		list = append(list, mc)
	}
	return list, rows.Err()
}

// UpsertModelCustomization saves or updates a model customization.
func (r *Repo) UpsertModelCustomization(mc ModelCustomization) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("model_customizations: nil repo")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if mc.ID == "" {
		mc.ID = "mc-" + uuid.NewString()
	}

	activeInt := 0
	if mc.IsActive {
		activeInt = 1
	}

	// Check if already exists for (provider_id, original_model_id)
	var existingID string
	err := r.db.QueryRow(
		`SELECT id FROM model_customizations WHERE provider_id = ? AND original_model_id = ? LIMIT 1`,
		mc.ProviderID, mc.OriginalModelID,
	).Scan(&existingID)

	if err == sql.ErrNoRows {
		_, err = r.db.Exec(
			`INSERT INTO model_customizations (id, provider_id, original_model_id, custom_name, custom_alias, context_window, is_active, created_at, updated_at, input_price, cache_price, output_price)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			mc.ID, mc.ProviderID, mc.OriginalModelID, mc.CustomName, mc.CustomAlias, mc.ContextWindow, activeInt, now, now, mc.InputPrice, mc.CachePrice, mc.OutputPrice,
		)
		return err
	} else if err != nil {
		return err
	}

	_, err = r.db.Exec(
		`UPDATE model_customizations
		 SET custom_name = ?, custom_alias = ?, context_window = ?, is_active = ?, updated_at = ?, input_price = ?, cache_price = ?, output_price = ?
		 WHERE id = ?`,
		mc.CustomName, mc.CustomAlias, mc.ContextWindow, activeInt, now, mc.InputPrice, mc.CachePrice, mc.OutputPrice, existingID,
	)
	return err
}

// SetModelCustomizationActive toggles active status for a model.
func (r *Repo) SetModelCustomizationActive(providerID, originalModelID string, active bool) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("model_customizations: nil repo")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	activeInt := 0
	if active {
		activeInt = 1
	}

	var existingID string
	err := r.db.QueryRow(
		`SELECT id FROM model_customizations WHERE provider_id = ? AND original_model_id = ? LIMIT 1`,
		providerID, originalModelID,
	).Scan(&existingID)

	if err == sql.ErrNoRows {
		id := "mc-" + uuid.NewString()
		_, err = r.db.Exec(
			`INSERT INTO model_customizations (id, provider_id, original_model_id, is_active, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			id, providerID, originalModelID, activeInt, now, now,
		)
		return err
	} else if err != nil {
		return err
	}

	_, err = r.db.Exec(
		`UPDATE model_customizations SET is_active = ?, updated_at = ? WHERE id = ?`,
		activeInt, now, existingID,
	)
	return err
}

// SetAllModelCustomizationsActive bulk toggles active status for a provider's models.
func (r *Repo) SetAllModelCustomizationsActive(providerID string, modelIDs []string, active bool) error {
	for _, m := range modelIDs {
		if err := r.SetModelCustomizationActive(providerID, m, active); err != nil {
			return err
		}
	}
	return nil
}

// DeleteModelCustomization removes a customization row.
func (r *Repo) DeleteModelCustomization(providerID, originalModelID string) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("model_customizations: nil repo")
	}
	_, err := r.db.Exec(
		`DELETE FROM model_customizations WHERE provider_id = ? AND original_model_id = ?`,
		providerID, originalModelID,
	)
	return err
}
