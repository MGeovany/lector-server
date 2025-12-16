package domain

import "github.com/supabase-community/supabase-go"

type SupabaseClient interface {
	Initialize() error
	ValidateToken(token string) (*SupabaseUser, error)

	DB() *supabase.Client
	GetClientWithToken(token string) (*supabase.Client, error)
}
