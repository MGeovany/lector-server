package domain

type AuthService interface {
	ValidateToken(token string) (*SupabaseUser, error)
}
