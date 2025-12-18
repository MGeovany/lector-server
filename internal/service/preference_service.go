package service

import (
	"time"

	"pdf-text-reader/internal/domain"
)

type preferenceService struct {
	preferenceRepo domain.PreferenceRepository
	logger         domain.Logger
}

func NewPreferenceService(
	preferenceRepo domain.PreferenceRepository,
	logger domain.Logger,
) domain.PreferenceService {
	return &preferenceService{
		preferenceRepo: preferenceRepo,
		logger:         logger,
	}
}

// GetPreferences retrieves user preferences
func (s *preferenceService) GetPreferences(userID string, token string) (*domain.UserPreferences, error) {
	return s.preferenceRepo.GetPreferences(userID, token)
}

// UpdatePreferences updates user preferences
func (s *preferenceService) UpdatePreferences(userID string, prefs *domain.UserPreferences, token string) error {
	prefs.UserID = userID
	prefs.UpdatedAt = time.Now()
	return s.preferenceRepo.UpdatePreferences(prefs, token)
}

// GetReadingPosition retrieves reading position for a document
func (s *preferenceService) GetReadingPosition(userID, documentID string, token string) (*domain.ReadingPosition, error) {
	return s.preferenceRepo.GetReadingPosition(userID, documentID, token)
}

// UpdateReadingPosition updates reading position for a document
func (s *preferenceService) UpdateReadingPosition(userID, documentID string, position *domain.ReadingPosition, token string) error {
	position.UserID = userID
	position.DocumentID = documentID
	position.UpdatedAt = time.Now()
	return s.preferenceRepo.UpdateReadingPosition(position, token)
}
