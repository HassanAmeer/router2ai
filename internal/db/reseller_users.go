package db

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ResellerUser represents a website/customer/tenant in the platform.
type ResellerUser struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	PlanTier      string `json:"planTier"`
	TokenBudget   int64  `json:"tokenBudget"`
	TokensUsed    int64  `json:"tokensUsed"`
	RPMLimit      int    `json:"rpmLimit"`
	IsBanned      bool   `json:"isBanned"`
	Subdomain     string `json:"subdomain"`
	AllowedModels string `json:"allowedModels"`
	APIKey        string `json:"apiKey"`
	KeysCount     int    `json:"keysCount,omitempty"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
}

// ListResellerUsers retrieves all website/reseller entries along with active key count.
func (r *Repo) ListResellerUsers() ([]ResellerUser, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("reseller_users: nil repo")
	}

	rows, err := r.db.Query(
		`SELECT u.id, u.email, u.name, u.plan_tier, u.token_budget, u.tokens_used, u.rpm_limit, u.is_banned,
		        COALESCE(u.subdomain, ''), COALESCE(u.allowed_models, ''), COALESCE(u.api_key, ''),
		        u.created_at, u.updated_at,
		        (SELECT COUNT(1) FROM apiKeys k WHERE k.userId = u.id) AS keys_count
		 FROM reseller_users u
		 ORDER BY u.created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list reseller_users: %w", err)
	}
	defer rows.Close()

	var list []ResellerUser
	for rows.Next() {
		var u ResellerUser
		var bannedInt int
		if err := rows.Scan(
			&u.ID, &u.Email, &u.Name, &u.PlanTier, &u.TokenBudget, &u.TokensUsed, &u.RPMLimit, &bannedInt,
			&u.Subdomain, &u.AllowedModels, &u.APIKey,
			&u.CreatedAt, &u.UpdatedAt, &u.KeysCount,
		); err != nil {
			return nil, err
		}
		u.IsBanned = bannedInt == 1
		list = append(list, u)
	}
	return list, rows.Err()
}

// GetResellerUser retrieves a user/website by ID.
func (r *Repo) GetResellerUser(id string) (*ResellerUser, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("reseller_users: nil repo")
	}

	var u ResellerUser
	var bannedInt int
	err := r.db.QueryRow(
		`SELECT u.id, u.email, u.name, u.plan_tier, u.token_budget, u.tokens_used, u.rpm_limit, u.is_banned,
		        COALESCE(u.subdomain, ''), COALESCE(u.allowed_models, ''), COALESCE(u.api_key, ''),
		        u.created_at, u.updated_at,
		        (SELECT COUNT(1) FROM apiKeys k WHERE k.userId = u.id) AS keys_count
		 FROM reseller_users u WHERE u.id = ? LIMIT 1`,
		id,
	).Scan(
		&u.ID, &u.Email, &u.Name, &u.PlanTier, &u.TokenBudget, &u.TokensUsed, &u.RPMLimit, &bannedInt,
		&u.Subdomain, &u.AllowedModels, &u.APIKey,
		&u.CreatedAt, &u.UpdatedAt, &u.KeysCount,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.IsBanned = bannedInt == 1
	return &u, nil
}

// CreateResellerUser creates a new website/reseller entry and auto-provisions a scoped API key.
func (r *Repo) CreateResellerUser(u ResellerUser) (*ResellerUser, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("reseller_users: nil repo")
	}

	if u.ID == "" {
		u.ID = "site-" + uuid.NewString()
	}
	now := time.Now().UTC().Format(time.RFC3339)
	u.CreatedAt = now
	u.UpdatedAt = now
	if u.PlanTier == "" {
		u.PlanTier = "standard"
	}

	// Auto-generate scoped API Key if not specified
	if u.APIKey == "" {
		rawHex := strings.ReplaceAll(uuid.NewString(), "-", "")[:24]
		u.APIKey = "sk-site-" + rawHex
	}

	bannedInt := 0
	if u.IsBanned {
		bannedInt = 1
	}

	_, err := r.db.Exec(
		`INSERT INTO reseller_users (id, email, name, plan_tier, token_budget, tokens_used, rpm_limit, is_banned, subdomain, allowed_models, api_key, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		u.ID, u.Email, u.Name, u.PlanTier, u.TokenBudget, u.TokensUsed, u.RPMLimit, bannedInt, u.Subdomain, u.AllowedModels, u.APIKey, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("create reseller_user: %w", err)
	}

	// Provision scoped API key entry in apiKeys table
	keyID := "key-" + uuid.NewString()
	sum := sha256.Sum256([]byte(u.APIKey))
	lookupHash := hex.EncodeToString(sum[:])

	_, _ = r.db.Exec(
		`INSERT INTO apiKeys (id, key, name, userId, lookupHash, isActive, rateLimitRPM, limitTokens, createdAt)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		keyID, u.APIKey, u.Name+" API Key", u.ID, lookupHash, loTernary(u.IsBanned, 0, 1), u.RPMLimit, u.TokenBudget, now,
	)

	// Attach model allowlist if defined
	r.syncKeyModelAccess(keyID, u.AllowedModels)

	u.KeysCount = 1
	return &u, nil
}

// UpdateResellerUser updates a website/reseller entry and syncs its linked API keys.
func (r *Repo) UpdateResellerUser(u ResellerUser) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("reseller_users: nil repo")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	bannedInt := 0
	if u.IsBanned {
		bannedInt = 1
	}

	_, err := r.db.Exec(
		`UPDATE reseller_users
		 SET email = ?, name = ?, plan_tier = ?, token_budget = ?, rpm_limit = ?, is_banned = ?, subdomain = ?, allowed_models = ?, api_key = ?, updated_at = ?
		 WHERE id = ?`,
		u.Email, u.Name, u.PlanTier, u.TokenBudget, u.RPMLimit, bannedInt, u.Subdomain, u.AllowedModels, u.APIKey, now, u.ID,
	)
	if err != nil {
		return err
	}

	// Sync linked apiKeys (active state, budget, RPM)
	isActiveVal := 1
	if u.IsBanned {
		isActiveVal = 0
	}
	_, _ = r.db.Exec(
		`UPDATE apiKeys
		 SET isActive = ?, rateLimitRPM = ?, limitTokens = ?
		 WHERE userId = ?`,
		isActiveVal, u.RPMLimit, u.TokenBudget, u.ID,
	)

	// Sync model access for linked keys
	rows, err := r.db.Query(`SELECT id FROM apiKeys WHERE userId = ?`, u.ID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var kID string
			if scanErr := rows.Scan(&kID); scanErr == nil {
				r.syncKeyModelAccess(kID, u.AllowedModels)
			}
		}
	}

	return nil
}

// syncKeyModelAccess updates allowed models for an API key.
func (r *Repo) syncKeyModelAccess(keyID, allowedModels string) {
	if keyID == "" {
		return
	}
	_, _ = r.db.Exec(`DELETE FROM api_key_model_access WHERE api_key_id = ?`, keyID)
	patterns := strings.Split(allowedModels, ",")
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p != "" {
			_, _ = r.db.Exec(
				`INSERT INTO api_key_model_access (api_key_id, model, created_at) VALUES (?, ?, CURRENT_TIMESTAMP)`,
				keyID, p,
			)
		}
	}
}

// DeleteResellerUser deletes a website/user and its linked keys and model access rules.
func (r *Repo) DeleteResellerUser(id string) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("reseller_users: nil repo")
	}

	// Clean up model access and keys
	_, _ = r.db.Exec(`DELETE FROM api_key_model_access WHERE api_key_id IN (SELECT id FROM apiKeys WHERE userId = ?)`, id)
	_, _ = r.db.Exec(`DELETE FROM apiKeys WHERE userId = ?`, id)
	_, err := r.db.Exec(`DELETE FROM reseller_users WHERE id = ?`, id)
	return err
}

func loTernary[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}
