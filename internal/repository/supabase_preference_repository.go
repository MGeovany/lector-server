package repository

import (
	"encoding/json"
	"fmt"

	"pdf-text-reader/internal/domain"
)

// SupabasePreferenceRepository implements the domain.PreferenceRepository interface
type SupabasePreferenceRepository struct {
	supabaseClient domain.SupabaseClient
	logger         domain.Logger
}

// NewSupabasePreferenceRepository creates a new Supabase preference repository
func NewSupabasePreferenceRepository(supabaseClient domain.SupabaseClient, logger domain.Logger) domain.PreferenceRepository {
	return &SupabasePreferenceRepository{
		supabaseClient: supabaseClient,
		logger:         logger,
	}
}

// GetPreferences retrieves user preferences from Supabase
func (r *SupabasePreferenceRepository) GetPreferences(userID string) (*domain.UserPreferences, error) {
	client := r.supabaseClient.DB()
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

	if len(prefsData) == 0 {
		// Return default preferences if none exist
		return &domain.UserPreferences{
			UserID:          userID,
			FontSize:        16,
			FontFamily:      "system-ui",
			TextColor:       "#000000",
			BackgroundColor: "#ffffff",
			LineHeight:      1.5,
			MaxWidth:        800,
			Theme:           "light",
		}, nil
	}

	return r.mapToPreferences(prefsData[0])
}

// UpdatePreferences updates or creates user preferences in Supabase
func (r *SupabasePreferenceRepository) UpdatePreferences(prefs *domain.UserPreferences) error {
	client := r.supabaseClient.DB()
	if client == nil {
		return fmt.Errorf("supabase client not initialized")
	}

	data := map[string]interface{}{
		"user_id":          prefs.UserID,
		"font_size":        prefs.FontSize,
		"font_family":      prefs.FontFamily,
		"text_color":       prefs.TextColor,
		"background_color": prefs.BackgroundColor,
		"line_height":      prefs.LineHeight,
		"max_width":        prefs.MaxWidth,
		"theme":            prefs.Theme,
		"updated_at":       prefs.UpdatedAt,
	}

	// Use upsert to insert or update
	_, _, err := client.From("user_preferences").
		Upsert(data, "", "", "").
		Execute()
	if err != nil {
		return fmt.Errorf("failed to update preferences: %w", err)
	}

	r.logger.Info("Preferences updated successfully", "user_id", prefs.UserID)
	return nil
}

// GetReadingPosition retrieves reading position for a document from Supabase
func (r *SupabasePreferenceRepository) GetReadingPosition(userID, documentID string) (*domain.ReadingPosition, error) {
	client := r.supabaseClient.DB()
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
			Position:   0,
			PageNumber: 1,
		}, nil
	}

	return r.mapToReadingPosition(positionsData[0])
}

// UpdateReadingPosition updates or creates reading position in Supabase
func (r *SupabasePreferenceRepository) UpdateReadingPosition(position *domain.ReadingPosition) error {
	client := r.supabaseClient.DB()
	if client == nil {
		return fmt.Errorf("supabase client not initialized")
	}

	data := map[string]interface{}{
		"user_id":     position.UserID,
		"document_id": position.DocumentID,
		"position":    position.Position,
		"page_number": position.PageNumber,
		"updated_at":  position.UpdatedAt,
	}

	// Use upsert to insert or update
	_, _, err := client.From("reading_positions").
		Upsert(data, "", "", "").
		Execute()
	if err != nil {
		return fmt.Errorf("failed to update reading position: %w", err)
	}

	r.logger.Info("Reading position updated successfully",
		"user_id", position.UserID,
		"document_id", position.DocumentID,
		"position", position.Position)
	return nil
}

// mapToPreferences converts a map to a UserPreferences struct
func (r *SupabasePreferenceRepository) mapToPreferences(data map[string]interface{}) (*domain.UserPreferences, error) {
	prefs := &domain.UserPreferences{
		UserID:          getString(data, "user_id"),
		FontSize:        getInt(data, "font_size"),
		FontFamily:      getString(data, "font_family"),
		TextColor:       getString(data, "text_color"),
		BackgroundColor: getString(data, "background_color"),
		LineHeight:      getFloat64(data, "line_height"),
		MaxWidth:        getInt(data, "max_width"),
		Theme:           getString(data, "theme"),
	}

	return prefs, nil
}

// mapToReadingPosition converts a map to a ReadingPosition struct
func (r *SupabasePreferenceRepository) mapToReadingPosition(data map[string]interface{}) (*domain.ReadingPosition, error) {
	position := &domain.ReadingPosition{
		UserID:     getString(data, "user_id"),
		DocumentID: getString(data, "document_id"),
		Position:   getInt(data, "position"),
		PageNumber: getInt(data, "page_number"),
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
