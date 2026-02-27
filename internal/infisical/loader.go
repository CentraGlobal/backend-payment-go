package infisical

import (
	"context"
	"fmt"
	"log"
	"os"

	infisical "github.com/infisical/go-sdk"
)

// LoadSecrets fetches all secrets from the Infisical instance and attaches
// them to the process environment. It is a no-op when the required
// INFISICAL_UNIVERSAL_AUTH_CLIENT_ID or INFISICAL_UNIVERSAL_AUTH_CLIENT_SECRET
// environment variables are not set.
func LoadSecrets(ctx context.Context) error {
	clientID := os.Getenv("INFISICAL_UNIVERSAL_AUTH_CLIENT_ID")
	clientSecret := os.Getenv("INFISICAL_UNIVERSAL_AUTH_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		log.Println("infisical: INFISICAL_UNIVERSAL_AUTH_CLIENT_ID or INFISICAL_UNIVERSAL_AUTH_CLIENT_SECRET not set, skipping secret load")
		return nil
	}

	projectID := os.Getenv("INFISICAL_PROJECT_ID")
	if projectID == "" {
		return fmt.Errorf("infisical: INFISICAL_PROJECT_ID is required when universal auth credentials are provided")
	}

	siteURL := os.Getenv("INFISICAL_SITE_URL")
	if siteURL == "" {
		siteURL = "https://app.infisical.com"
	}

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}

	client := infisical.NewInfisicalClient(ctx, infisical.Config{
		SiteUrl: siteURL,
	})

	if _, err := client.Auth().UniversalAuthLogin(clientID, clientSecret); err != nil {
		return fmt.Errorf("infisical: authentication failed: %w", err)
	}

	_, err := client.Secrets().List(infisical.ListSecretsOptions{
		ProjectID:          projectID,
		Environment:        env,
		SecretPath:         "/",
		AttachToProcessEnv: true,
	})
	if err != nil {
		return fmt.Errorf("infisical: failed to fetch secrets: %w", err)
	}

	log.Printf("infisical: secrets loaded successfully for environment %q", env)
	return nil
}
