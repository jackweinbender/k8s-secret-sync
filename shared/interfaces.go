package shared

import "context"

type SecretProvider interface {
	GetSecretValue(ctx context.Context, secretID string) (string, error)
}
