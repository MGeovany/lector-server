package service

import (
	"errors"
	"testing"

	"pdf-text-reader/internal/domain"
)

type mockUserPreferencesRepo struct {
	prefs        map[string]*domain.UserPreferences
	positions    map[string]map[string]*domain.ReadingPosition
	lastUpdated  *domain.UserPreferences
	lastPosition *domain.ReadingPosition
}

func newMockUserPreferencesRepo() *mockUserPreferencesRepo {
	return &mockUserPreferencesRepo{
		prefs:     make(map[string]*domain.UserPreferences),
		positions: make(map[string]map[string]*domain.ReadingPosition),
	}
}

func (m *mockUserPreferencesRepo) GetPreferences(userID string, token string) (*domain.UserPreferences, error) {
	prefs, ok := m.prefs[userID]
	if !ok {
		return nil, errors.New("preferences not found")
	}
	return prefs, nil
}

func (m *mockUserPreferencesRepo) UpdatePreferences(prefs *domain.UserPreferences, token string) error {
	m.lastUpdated = prefs
	m.prefs[prefs.UserID] = prefs
	return nil
}

func (m *mockUserPreferencesRepo) GetReadingPosition(userID, documentID string, token string) (*domain.ReadingPosition, error) {
	userPositions, ok := m.positions[userID]
	if !ok {
		return nil, errors.New("position not found")
	}
	position, ok := userPositions[documentID]
	if !ok {
		return nil, errors.New("position not found")
	}
	return position, nil
}

func (m *mockUserPreferencesRepo) GetAllReadingPositions(userID string, token string) (map[string]*domain.ReadingPosition, error) {
	userPositions, ok := m.positions[userID]
	if !ok {
		return map[string]*domain.ReadingPosition{}, nil
	}
	return userPositions, nil
}

func (m *mockUserPreferencesRepo) UpdateReadingPosition(position *domain.ReadingPosition, token string) error {
	m.lastPosition = position
	if m.positions[position.UserID] == nil {
		m.positions[position.UserID] = make(map[string]*domain.ReadingPosition)
	}
	m.positions[position.UserID][position.DocumentID] = position
	return nil
}

func TestUserPreferencesService_GetPreferences(t *testing.T) {
	repo := newMockUserPreferencesRepo()
	logger := NewMockLogger()

	prefs := &domain.UserPreferences{UserID: "user-1", FontSize: 18}
	repo.prefs["user-1"] = prefs

	svc := NewUserPreferencesService(repo, logger)
	got, err := svc.GetPreferences("user-1", "token")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != prefs {
		t.Fatalf("expected preferences to be returned from repo")
	}
}

func TestUserPreferencesService_UpdatePreferences(t *testing.T) {
	repo := newMockUserPreferencesRepo()
	logger := NewMockLogger()

	svc := NewUserPreferencesService(repo, logger)
	prefs := &domain.UserPreferences{FontSize: 20}

	if err := svc.UpdatePreferences("user-2", prefs, "token"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.lastUpdated == nil {
		t.Fatalf("expected repo to receive updated preferences")
	}
	if repo.lastUpdated.UserID != "user-2" {
		t.Fatalf("expected user id to be set, got %s", repo.lastUpdated.UserID)
	}
	if repo.lastUpdated.UpdatedAt.IsZero() {
		t.Fatalf("expected updated at to be set")
	}
}

func TestUserPreferencesService_GetAllReadingPositions(t *testing.T) {
	repo := newMockUserPreferencesRepo()
	logger := NewMockLogger()

	position := &domain.ReadingPosition{UserID: "user-3", DocumentID: "doc-1", Progress: 0.5, PageNumber: 2}
	repo.positions["user-3"] = map[string]*domain.ReadingPosition{"doc-1": position}

	svc := NewUserPreferencesService(repo, logger)
	got, err := svc.GetAllReadingPositions("user-3", "token")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 position, got %d", len(got))
	}
	if got["doc-1"].DocumentID != "doc-1" {
		t.Fatalf("expected position for doc-1")
	}
}

func TestUserPreferencesService_UpdateReadingPosition(t *testing.T) {
	repo := newMockUserPreferencesRepo()
	logger := NewMockLogger()

	svc := NewUserPreferencesService(repo, logger)
	position := &domain.ReadingPosition{Progress: 0.25, PageNumber: 4}

	if err := svc.UpdateReadingPosition("user-4", "doc-2", position, "token"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.lastPosition == nil {
		t.Fatalf("expected repo to receive updated position")
	}
	if repo.lastPosition.UserID != "user-4" {
		t.Fatalf("expected user id user-4, got %s", repo.lastPosition.UserID)
	}
	if repo.lastPosition.DocumentID != "doc-2" {
		t.Fatalf("expected document id doc-2, got %s", repo.lastPosition.DocumentID)
	}
	if repo.lastPosition.UpdatedAt.IsZero() {
		t.Fatalf("expected updated at to be set")
	}
}
