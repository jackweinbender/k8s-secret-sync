package op

import (
	"context"
	"log"
	"os"

	"github.com/1password/onepassword-sdk-go"
)

type SecretProvider struct {
	Client *onepassword.Client
}

func (p SecretProvider) GetSecretValue(ctx context.Context, secretID string) (string, error) {
	value, err := p.Client.Secrets().Resolve(ctx, secretID)
	if err != nil {
		log.Printf("Failed to resolve 1Password secret URI %s: %v", secretID, err)
		return "", err
	}

	return value, nil
}

func InitClient() (*onepassword.Client, error) {
	token := os.Getenv("OP_SERVICE_ACCOUNT_TOKEN")

	client, err := onepassword.NewClient(
		context.TODO(),
		onepassword.WithServiceAccountToken(token),
		onepassword.WithIntegrationInfo("My k8s secret sync operator", "v0"),
	)
	if err != nil {
		return nil, err
	}

	return client, nil
}
