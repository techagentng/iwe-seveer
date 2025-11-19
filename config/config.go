package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Debug                        bool   `envconfig:"debug"`
	Port                         int    `envconfig:"port"`
	DatabaseURL                  string `envconfig:"database_url"` 
	PostgresHost                 string `envconfig:"postgres_host"`
	PostgresUser                 string `envconfig:"postgres_user"`
	PostgresDB                   string `envconfig:"postgres_db"`
	MailgunApiKey                string `envconfig:"mg_public_api_key"`
	MgEmailFrom                  string `envconfig:"email_from"`
	BaseUrl                      string `envconfig:"base_url"`
	Env                          string `envconfig:"env"`
	PostgresPort                 int    `envconfig:"postgres_port"`
	PostgresPassword             string `envconfig:"postgres_password"`
	JWTSecret                    string `envconfig:"jwt_secret"`
	MgDomain                     string `envconfig:"mg_domain"`
	Host                         string `envconfig:"host"`
	GoogleClientID               string `envconfig:"google_client_id"`
	GoogleClientSecret           string `envconfig:"google_client_secret"`
	GoogleRedirectURL            string `envconfig:"google_redirect_url"`
	GoogleApplicationCredentials string `envconfig:"google_application_credentials"`
	FacebookAppId                string `envconfig:"facebook_app_id"`
	FacebookAppSecret            string `envconfig:"facebook_app_secret"`
	FacebookRedirectURL          string `envconfig:"facebook_redirect_url"`
	GoogleMapsApiKey             string `envconfig:"google_maps_api_key"`
	AccessControlAllowOrigin     string `envconfig:"accessc_control_allow_origin"`
	AWS_BUCKET                   string `envconfig:"aws_bucket"`
	AWS_REGION                   string `envconfig:"aws_region"`
	AWS_ACCESS_KEY_ID            string `envconfig:"aws_access_key_id"`
	AWS_SECRET_ACCESS_KEY        string `envconfig:"aws_secret_access_key"`
	FRONTEND_URL        string `envconfig:"frontend_url"`
	GOOGLE_CLOUD_PROJECT string `envconfig:"google_cloud_project"`
	
	// Redis Configuration
	RedisURL      string `envconfig:"redis_url"`      // For production (Render/Railway)
	RedisHost     string `envconfig:"redis_host"`     // For local development
	RedisPort     string `envconfig:"redis_port"`     // For local development
	RedisPassword string `envconfig:"redis_password"` // Optional
	RedisDB       int    `envconfig:"redis_db"`       // Default 0
	
	// OpenAI Configuration
	OpenAIAPIKey string `envconfig:"openai_api_key"` // OpenAI API key for GPT-4o-mini
}

func (c *Config) GetDBUrl() string {
	// If DATABASE_URL is provided (Render, Railway, Heroku), use it directly
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}
	
	// Otherwise, construct from individual fields
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		c.PostgresUser,
		c.PostgresPassword,
		c.PostgresHost,
		c.PostgresPort,
		c.PostgresDB,
	)
}

func (c *Config) GetRedisAddr() string {
	// If REDIS_URL is provided (production), use it
	if c.RedisURL != "" {
		return c.RedisURL
	}
	
	// Otherwise, construct from host:port (local development)
	if c.RedisHost == "" {
		c.RedisHost = "localhost"
	}
	if c.RedisPort == "" {
		c.RedisPort = "6379"
	}
	
	return fmt.Sprintf("%s:%s", c.RedisHost, c.RedisPort)
}

func Load() (*Config, error) {
	env := os.Getenv("GIN_MODE")
	if env != "release" {
		if err := godotenv.Load("./.env"); err != nil {
			log.Printf("couldn't load env vars: %v", err)
		}
	}

	c := &Config{}
	err := envconfig.Process("citizenx", c)
	if err != nil {
		return nil, err
	}
	return c, nil
}
