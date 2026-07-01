// Package config loads runtime configuration from environment variables.
package config

import (
	"os"
)

// Config holds all runtime settings
type Config struct {
	Addr            string
	FrontendOrigin  string
	DevMode         string
	SocrataAppToken string
}

// Load reads configuration from the environment and applies defaults
// when variables are not set.
func Load() Config {
	cfg := Config{
		Addr:            env("MASSDATA_ADDR", ":8080"),
		FrontendOrigin:  env("MASSDATA_FRONTEND_ORIGIN", "https://baystate.info"),
		DevMode:         env("MASSDATA_DEV_MODE", "false"),
		SocrataAppToken: env("SOCRATA_APP_TOKEN", ""),
	}

	return cfg
}

// env sets a given fallback value if a variable is not set in the
// environment.
// NOTE: environment variables added to Cloud Secrets need to be
// made available to the Cloud Run service when running in GCP.
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
