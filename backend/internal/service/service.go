package service

import (
	"context"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/event"
)

type Services struct {
	Auth  *AuthService
	Event *event.Service
}

func New(auth *AuthService, eventService *event.Service) *Services {
	return &Services{Auth: auth, Event: eventService}
}

type TransactorFunc func(ctx context.Context, fn func(context.Context) error) error

func (f TransactorFunc) Do(
	ctx context.Context,
	fn func(context.Context) error,
) error {
	return f(ctx, fn)
}
