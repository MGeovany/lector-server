package domain

type AuthService interface {
	ValidateToken(token string) (*SupabaseUser, error)
	IsAccountDisabled(userID string, token string) (bool, error)
}
