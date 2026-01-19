package repository

import (
	"encoding/json"
	"fmt"
	"time"

	"pdf-text-reader/internal/domain"
)

// UserPreferencesRepository implements the domain.UserPreferencesRepository interface
type UserPreferencesRepository struct {
	supabaseClient domain.SupabaseClient
	logger         domain.Logger
}

// NewSupabasePreferenceRepository creates a new Supabase preference repository

func NewUserPreferencesRepository(supabaseClient domain.SupabaseClient, logger domain.Logger) domain.UserPreferencesRepository {
	return &UserPreferencesRepository{
		supabaseClient: supabaseClient,
		logger:         logger,
	}
}

// GetPreferences retrieves user preferences from Supabase

func (r *UserPreferencesRepository) GetPreferences(userID string, token string) (*domain.UserPreferences, error) {
	// Use client with token for RLS policies
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to get client with token: %w", err)
	}
	if client == nil {
		return nil, fmt.Errorf("supabase client not initialized")
	}

	data, _, err := client.From("user_preferences").
		Select("*", "", false).
		Eq("user_id", userID).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get preferences: %w", err)
	}

	var prefsData []map[string]interface{}
	if err := json.Unmarshal(data, &prefsData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	var prefs *domain.UserPreferences
	if len(prefsData) == 0 {
		// Return default preferences if none exist
		prefs = &domain.UserPreferences{
			UserID:            userID,
			FontSize:          16,
			FontFamily:        "system-ui",
			Theme:             "light",
			SubscriptionPlan:  "free",
			StorageLimitBytes: 15 * 1024 * 1024,
			Tags:              []string{},
		}
	} else {
		prefs, err = r.mapToPreferences(prefsData[0])
		if err != nil {
			return nil, err
		}
	}

	// Fetch tags from user_tags table
	tagsData, _, err := client.From("user_tags").
		Select("name", "", false).
		Eq("user_id", userID).
		Execute()
	if err == nil {
		var tagsList []map[string]interface{}
		if err := json.Unmarshal(tagsData, &tagsList); err == nil {
			tags := make([]string, 0, len(tagsList))
			for _, tagData := range tagsList {
				if tagName := getString(tagData, "name"); tagName != "" {
					tags = append(tags, tagName)
				}
			}
			prefs.Tags = tags
		}
	}

	return prefs, nil
}

// UpdatePreferences updates or creates user preferences in Supabase
func (r *UserPreferencesRepository) UpdatePreferences(prefs *domain.UserPreferences, token string) error {
	// Use client with token for RLS policies
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return fmt.Errorf("failed to get client with token: %w", err)
	}
	if client == nil {
		return fmt.Errorf("supabase client not initialized")
	}

	// Update user_preferences (without tags - tags are in separate table)
	data := map[string]interface{}{
		"user_id":             prefs.UserID,
		"font_size":           prefs.FontSize,
		"font_family":         prefs.FontFamily,
		"theme":               prefs.Theme,
		"subscription_plan":   prefs.SubscriptionPlan,
		"storage_limit_bytes": prefs.StorageLimitBytes,
		// Don't send updated_at - the database trigger will handle it
	}

	// Use upsert to insert or update
	_, _, err = client.From("user_preferences").
		Upsert(data, "", "", "").
		Execute()
	if err != nil {
		return fmt.Errorf("failed to update preferences: %w", err)
	}

	// Update tags in user_tags table
	// First, get existing tags
	existingTagsData, _, err := client.From("user_tags").
		Select("id,name", "", false).
		Eq("user_id", prefs.UserID).
		Execute()
	if err != nil {
		return fmt.Errorf("failed to get existing tags: %w", err)
	}

	var existingTags []map[string]interface{}
	if err := json.Unmarshal(existingTagsData, &existingTags); err != nil {
		return fmt.Errorf("failed to unmarshal existing tags: %w", err)
	}

	// Create a map of existing tag names
	existingTagMap := make(map[string]string) // name -> id
	for _, tagData := range existingTags {
		tagName := getString(tagData, "name")
		tagID := getString(tagData, "id")
		if tagName != "" && tagID != "" {
			existingTagMap[tagName] = tagID
		}
	}

	// Create a set of new tag names
	newTagSet := make(map[string]bool)
	for _, tagName := range prefs.Tags {
		if tagName != "" {
			newTagSet[tagName] = true
		}
	}

	// Delete tags that are no longer in the list
	for tagName, tagID := range existingTagMap {
		if !newTagSet[tagName] {
			// First, delete all document_tag relationships for this tag
			// (CASCADE should handle this, but we'll do it explicitly to be safe)
			_, _, err := client.From("document_tags").
				Delete("", "").
				Eq("tag_id", tagID).
				Execute()
			if err != nil {
				r.logger.Warn("Failed to delete document_tag relationships", "error", err, "tag_id", tagID)
			}

			// Then delete the tag itself
			_, _, err = client.From("user_tags").
				Delete("", "").
				Eq("id", tagID).
				Execute()
			if err != nil {
				r.logger.Warn("Failed to delete tag", "error", err, "tag_id", tagID, "tag_name", tagName)
			}
		}
	}

	// Insert new tags that don't exist
	for _, tagName := range prefs.Tags {
		if tagName == "" {
			continue
		}
		if _, exists := existingTagMap[tagName]; !exists {
			tagData := map[string]interface{}{
				"user_id": prefs.UserID,
				"name":    tagName,
			}
			_, _, err := client.From("user_tags").
				Insert(tagData, false, "", "", "").
				Execute()
			if err != nil {
				r.logger.Warn("Failed to insert tag", "error", err, "tag_name", tagName)
				// Continue with other tags even if one fails
			}
		}
	}

	r.logger.Info("Preferences updated successfully", "user_id", prefs.UserID)
	return nil
}

// GetReadingPosition retrieves reading position for a document from Supabase
func (r *UserPreferencesRepository) GetReadingPosition(userID, documentID string, token string) (*domain.ReadingPosition, error) {
	// Use client with token for RLS policies
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to get client with token: %w", err)
	}
	if client == nil {
		return nil, fmt.Errorf("supabase client not initialized")
	}

	data, _, err := client.From("reading_positions").
		Select("*", "", false).
		Eq("user_id", userID).
		Eq("document_id", documentID).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get reading position: %w", err)
	}

	var positionsData []map[string]interface{}
	if err := json.Unmarshal(data, &positionsData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(positionsData) == 0 {
		// Return default position if none exists
		return &domain.ReadingPosition{
			UserID:     userID,
			DocumentID: documentID,
			Progress:   0.0,
			PageNumber: 1,
			UpdatedAt:  time.Now(),
		}, nil
	}

	return r.mapToReadingPosition(positionsData[0])
}

// GetAllReadingPositions retrieves all reading positions for a user from Supabase
func (r *UserPreferencesRepository) GetAllReadingPositions(userID string, token string) (map[string]*domain.ReadingPosition, error) {
	// Use client with token for RLS policies
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to get client with token: %w", err)
	}
	if client == nil {
		return nil, fmt.Errorf("supabase client not initialized")
	}

	data, _, err := client.From("reading_positions").
		Select("*", "", false).
		Eq("user_id", userID).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get reading positions: %w", err)
	}

	var positionsData []map[string]interface{}
	if err := json.Unmarshal(data, &positionsData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	positionsMap := make(map[string]*domain.ReadingPosition)
	for _, posData := range positionsData {
		position, err := r.mapToReadingPosition(posData)
		if err != nil {
			r.logger.Warn("Failed to map reading position", "error", err)
			continue
		}
		positionsMap[position.DocumentID] = position
	}

	return positionsMap, nil
}

// UpdateReadingPosition updates or creates reading position in Supabase
func (r *UserPreferencesRepository) UpdateReadingPosition(position *domain.ReadingPosition, token string) error {
	// Use client with token for RLS policies
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return fmt.Errorf("failed to get client with token: %w", err)
	}
	if client == nil {
		return fmt.Errorf("supabase client not initialized")
	}

	data := map[string]interface{}{
		"user_id":     position.UserID,
		"document_id": position.DocumentID,
		"progress":    position.Progress,
		"page_number": position.PageNumber,
		"updated_at":  position.UpdatedAt,
		// Don't send updated_at - the database trigger will handle it
	}

	// Use upsert to insert or update
	_, _, err = client.From("reading_positions").
		Upsert(data, "", "", "").
		Execute()
	if err != nil {
		return fmt.Errorf("failed to update reading position: %w", err)
	}

	r.logger.Info("Reading position updated successfully",
		"user_id", position.UserID,
		"document_id", position.DocumentID,
		"progress", position.Progress,
		"page_number", position.PageNumber,
		"updated_at", position.UpdatedAt)
	return nil
}

// mapToPreferences converts a map to a UserPreferences struct
func (r *UserPreferencesRepository) mapToPreferences(data map[string]interface{}) (*domain.UserPreferences, error) {
	prefs := &domain.UserPreferences{
		UserID:            getString(data, "user_id"),
		FontSize:          getInt(data, "font_size"),
		FontFamily:        getString(data, "font_family"),
		Theme:             getString(data, "theme"),
		SubscriptionPlan:  getString(data, "subscription_plan"),
		StorageLimitBytes: getInt64(data, "storage_limit_bytes"),
		AccountDisabled:   getBool(data, "account_disabled"),
		Tags:              []string{}, // Tags are loaded separately from user_tags table
		UpdatedAt:         time.Now(),
	}

	// Backfill defaults for older rows.
	if prefs.SubscriptionPlan == "" {
		prefs.SubscriptionPlan = "free"
	}
	if prefs.StorageLimitBytes <= 0 {
		// If storage_limit_bytes is missing, derive it from the plan so Pro users
		// still get the correct quota.
		prefs.StorageLimitBytes = domain.StorageLimitBytesForPlan(prefs.SubscriptionPlan)
	}

	// Parse updated_at if available
	if updatedAtStr := getString(data, "updated_at"); updatedAtStr != "" {
		if t, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
			prefs.UpdatedAt = t
		} else if t, err := time.Parse(time.RFC3339Nano, updatedAtStr); err == nil {
			prefs.UpdatedAt = t
		}
	}

	return prefs, nil
}

func getInt64(data map[string]interface{}, key string) int64 {
	if val, ok := data[key]; ok && val != nil {
		switch v := val.(type) {
		case int:
			return int64(v)
		case int64:
			return v
		case float64:
			return int64(v)
		}
	}
	return 0
}

// mapToReadingPosition converts a map to a ReadingPosition struct
func (r *UserPreferencesRepository) mapToReadingPosition(data map[string]interface{}) (*domain.ReadingPosition, error) {
	position := &domain.ReadingPosition{
		UserID:     getString(data, "user_id"),
		DocumentID: getString(data, "document_id"),
		Progress:   float32(getFloat64(data, "progress")),
		PageNumber: getInt(data, "page_number"),
		UpdatedAt:  time.Now(),
	}

	return position, nil
}

// Helper functions for type conversion
func getInt(data map[string]interface{}, key string) int {
	if val, ok := data[key]; ok && val != nil {
		switch v := val.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
	}
	return 0
}

func getFloat64(data map[string]interface{}, key string) float64 {
	if val, ok := data[key]; ok && val != nil {
		switch v := val.(type) {
		case float64:
			return v
		case int:
			return float64(v)
		case int64:
			return float64(v)
		}
	}
	return 0.0
}

func getBool(data map[string]interface{}, key string) bool {
	if val, ok := data[key]; ok && val != nil {
		switch v := val.(type) {
		case bool:
			return v
		case string:
			return v == "true" || v == "1"
		case float64:
			return v != 0
		case int:
			return v != 0
		case int64:
			return v != 0
		}
	}
	return false
}
