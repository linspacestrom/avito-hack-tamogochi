package entity

import "time"

type TokenPair struct {
	AccessToken      string
	RefreshToken     string
	RefreshExpiresAt time.Time
}

type AuthResult struct {
	User   User
	Tokens TokenPair
}
