package service

import (
	"context"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/event"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/rewards"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/tasks"
)

type Services struct {
	Auth       *AuthService
	Event      *event.Service
	Rewards    *rewards.Service
	DailyCycle *rewards.DailyCycleService
	Tasks      *tasks.Service
}

func New(
	auth *AuthService,
	eventService *event.Service,
	rewardsService *rewards.Service,
	dailyCycle *rewards.DailyCycleService,
	tasksService *tasks.Service,
) *Services {
	return &Services{
		Auth:       auth,
		Event:      eventService,
		Rewards:    rewardsService,
		DailyCycle: dailyCycle,
	}
	return &Services{Auth: auth, Rewards: rewardsService, DailyCycle: dailyCycle, Tasks: tasksService}
}

type TransactorFunc func(ctx context.Context, fn func(context.Context) error) error

func (f TransactorFunc) Do(
	ctx context.Context,
	fn func(context.Context) error,
) error {
	return f(ctx, fn)
}
