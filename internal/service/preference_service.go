package service

import (
	"time"

	"pdf-text-reader/internal/domain"
)

type userPreferencesService struct {
	userPreferencesRepo domain.UserPreferencesRepository
	logger              domain.Logger
}

func NewUserPreferencesService(
	userPreferencesRepo domain.UserPreferencesRepository,
	logger domain.Logger,
) domain.UserPreferencesService {
	return &userPreferencesService{
		userPreferencesRepo: userPreferencesRepo,
		logger:              logger,
	}
}

// GetPreferences retrieves user preferences
func (s *userPreferencesService) GetPreferences(userID string, token string) (*domain.UserPreferences, error) {
	return s.userPreferencesRepo.GetPreferences(userID, token)
}

// UpdatePreferences updates user preferences
func (s *userPreferencesService) UpdatePreferences(userID string, prefs *domain.UserPreferences, token string) error {
	prefs.UserID = userID
	prefs.UpdatedAt = time.Now()
	return s.userPreferencesRepo.UpdatePreferences(prefs, token)
}

// GetReadingPosition retrieves reading position for a document
func (s *userPreferencesService) GetReadingPosition(userID, documentID string, token string) (*domain.ReadingPosition, error) {
	return s.userPreferencesRepo.GetReadingPosition(userID, documentID, token)
}

// GetAllReadingPositions retrieves all reading positions for a user
func (s *userPreferencesService) GetAllReadingPositions(userID string, token string) (map[string]*domain.ReadingPosition, error) {
	return s.userPreferencesRepo.GetAllReadingPositions(userID, token)
}

// UpdateReadingPosition updates reading position for a document
func (s *userPreferencesService) UpdateReadingPosition(userID, documentID string, position *domain.ReadingPosition, token string) error {
	position.UserID = userID
	position.DocumentID = documentID
	position.UpdatedAt = time.Now()
	return s.userPreferencesRepo.UpdateReadingPosition(position, token)
}
