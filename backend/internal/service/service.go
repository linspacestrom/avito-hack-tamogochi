package service

import (
	"context"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/rewards"
)

type Services struct {
	Auth       *AuthService
	Rewards    *rewards.Service
	DailyCycle *rewards.DailyCycleService
}

func New(
	auth *AuthService,
	rewardsService *rewards.Service,
	dailyCycle *rewards.DailyCycleService,
) *Services {
	return &Services{Auth: auth, Rewards: rewardsService, DailyCycle: dailyCycle}
}

type TransactorFunc func(ctx context.Context, fn func(context.Context) error) error

func (f TransactorFunc) Do(
	ctx context.Context,
	fn func(context.Context) error,
) error {
	return f(ctx, fn)
}
