package service

import "context"

type Services struct {
	Auth *AuthService
}

func New(auth *AuthService) *Services {
	return &Services{Auth: auth}
}

type TransactorFunc func(ctx context.Context, fn func(context.Context) error) error

func (f TransactorFunc) Do(
	ctx context.Context,
	fn func(context.Context) error,
) error {
	return f(ctx, fn)
}
