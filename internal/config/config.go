package config

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type HTTPServer struct {
	Addr string `yaml:"address"`
}

type Database struct {
	Host            string        `yaml:"PG_HOST" env:"PG_HOST" env-default:"0.0.0.0"`
	Port            string        `yaml:"PG_PORT" env:"PG_PORT" env-default:"5432"`
	User            string        `yaml:"PG_USER" env:"PG_USER" env-required:"true"`
	Password        string        `yaml:"PG_PASSWORD" env:"PG_PASSWORD" env-required:"true"`
	Name            string        `yaml:"PG_DBNAME" env:"PG_DBNAME" env-required:"true"`
	SSLMode         string        `yaml:"PG_SSLMODE" env:"PG_SSLMODE" env-default:"require"`
	MaxOpenConns    int           `yaml:"MAX_OPEN_CONNS" env:"MAX_OPEN_CONNS" env-default:"25"`
	MaxIdleConns    int           `yaml:"MAX_IDLE_CONNS" env:"MAX_IDLE_CONNS" env-default:"10"`
	ConnMaxLifetime time.Duration `yaml:"CONN_MAX_LIFETIME" env:"CONN_MAX_LIFETIME" env-default:"5m"`
	ConnMaxIdleTime time.Duration `yaml:"CONN_MAX_IDLE_TIME" env:"CONN_MAX_IDLE_TIME" env-default:"1m"`
}

type RedisConnect struct {
	Host     string `yaml:"REDIS_HOST" env:"REDIS_HOST"`
	Username string `yaml:"REDIS_USER" env:"REDIS_USER" env-required:"true"`
	Password string `yaml:"REDIS_PASSWORD" env:"REDIS_PASSWORD" env-required:"true"`
	DB       int    `yaml:"REDIS_DB" env:"REDIS_DB" env-default:"0"`
}

type RateConfig struct {
	MaxAttempts int64         `yaml:"MAX_ATTEMPTS" env:"MAX_ATTEMPTS" env-default:"5"`
	WindowSize  time.Duration `yaml:"WINDOW_SIZE" env:"WINDOW_SIZE" env-default:"15s"`
}

type Stripe struct {
	APIKey              string   `yaml:"STRIPE_API_KEY" env:"STRIPE_API_KEY" env-default:""`
	WebhookSecret       string   `yaml:"STRIPE_WEBHOOK_SECRET" env:"STRIPE_WEBHOOK_SECRET" env-default:""`
	PaymentMethods      []string `yaml:"STRIPE_PAYMENT_METHODS" env:"STRIPE_PAYMENT_METHODS" env-default:"card,bank_transfer"`
	SupportedCurrencies []string `yaml:"STRIPE_SUPPORTED_CURRENCIES" env:"STRIPE_SUPPORTED_CURRENCIES" env-default:"inr, usd, eur"`
}

type SendGrid struct {
	APIKey     string `yaml:"API_KEY" env:"API_KEY" env-default:""`
	FromEmail  string `yaml:"FROM_EMAIL" env:"FROM_EMAIL" env-default:"noreply@example.com"`
	FromName   string `yaml:"FROM_NAME" env:"FROM_NAME" env-default:"Notification Service"`
	SMSEnabled bool   `yaml:"SMSENABLED" env:"SMSENABLED" env-default:"false"`
}

type Security struct {
	JWTKey         string `yaml:"JWT_KEY" env:"JWT_KEY" env-required:"true"`
	JWTExpiryHours int    `yaml:"JWT_EXPIRY_HOURS" env:"JWT_EXPIRY_HOURS" env-default:"24"`
}

type Config struct {
	Env          string `yaml:"env" env:"ENV" env-required:"true"`
	StoragePath  string `yaml:"storage_path" env-required:"true"`
	HTTPServer   `yaml:"http_server"`
	Database     Database     `yaml:"database"`
	RedisConnect RedisConnect `yaml:"redis"`
	RateConfig   RateConfig   `yaml:"rateConfig"`
	Stripe       Stripe       `yaml:"stripe"`
	SendGrid     SendGrid     `yaml:"sendgrid"`
	Security     Security     `yaml:"security"`
}

func MustLoad() *Config {

	var configPath string

	configPath = os.Getenv("CONFIG_PATH")

	if configPath == "" {

		flags := flag.String("config", "", "gets the config flag value")

		flag.Parse()

		configPath = *flags

		if configPath == "" {

			log.Fatal("Config path is not set")

		}

	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("config file does not exist: %s", configPath)
	}

	var cfg Config

	err := cleanenv.ReadConfig(configPath, &cfg)

	if err != nil {

		log.Fatalf("can not read config file: %s", err.Error())
	}

	return &cfg

}

func (d *Database) GetDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Name, d.SSLMode)
}
