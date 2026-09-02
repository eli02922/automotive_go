package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Postgres PostgresConfig
	MSSQL    MSSQLConfig
	Oracle   OracleConfig
	Redis    RedisConfig
	Kafka    KafkaConfig
	AWS      AWSConfig
}

type ServerConfig struct {
	Port         string        `mapstructure:"PORT"`
	ReadTimeout  time.Duration `mapstructure:"READ_TIMEOUT"`
	WriteTimeout time.Duration `mapstructure:"WRITE_TIMEOUT"`
	JWTSecret    string        `mapstructure:"JWT_SECRET"`
}

type PostgresConfig struct {
	DSN             string        `mapstructure:"POSTGRES_DSN"`
	MaxOpenConns    int           `mapstructure:"POSTGRES_MAX_OPEN_CONNS"`
	MaxIdleConns    int           `mapstructure:"POSTGRES_MAX_IDLE_CONNS"`
	ConnMaxLifetime time.Duration `mapstructure:"POSTGRES_CONN_MAX_LIFETIME"`
}

type MSSQLConfig struct {
	DSN          string `mapstructure:"MSSQL_DSN"`
	MaxOpenConns int    `mapstructure:"MSSQL_MAX_OPEN_CONNS"`
}

type OracleConfig struct {
	DSN          string `mapstructure:"ORACLE_DSN"`
	MaxOpenConns int    `mapstructure:"ORACLE_MAX_OPEN_CONNS"`
}

type RedisConfig struct {
	Addr     string        `mapstructure:"REDIS_ADDR"`
	Password string        `mapstructure:"REDIS_PASSWORD"`
	DB       int           `mapstructure:"REDIS_DB"`
	TTL      time.Duration `mapstructure:"REDIS_TTL"`
}

type KafkaConfig struct {
	Brokers         []string      `mapstructure:"KAFKA_BROKERS"`
	ProductTopic    string        `mapstructure:"KAFKA_PRODUCT_TOPIC"`
	InventoryTopic  string        `mapstructure:"KAFKA_INVENTORY_TOPIC"`
	FitmentTopic    string        `mapstructure:"KAFKA_FITMENT_TOPIC"`
	ConsumerGroup   string        `mapstructure:"KAFKA_CONSUMER_GROUP"`
	CommitInterval  time.Duration `mapstructure:"KAFKA_COMMIT_INTERVAL"`
}

type AWSConfig struct {
	Region          string `mapstructure:"AWS_REGION"`
	S3Bucket        string `mapstructure:"AWS_S3_BUCKET"`
	AccessKeyID     string `mapstructure:"AWS_ACCESS_KEY_ID"`
	SecretAccessKey string `mapstructure:"AWS_SECRET_ACCESS_KEY"`
}

func Load() (*Config, error) {
	v := viper.New()

	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")
	v.AddConfigPath("../")

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	setDefaults(v)

	_ = v.ReadInConfig()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("PORT", "8080")
	v.SetDefault("READ_TIMEOUT", "30s")
	v.SetDefault("WRITE_TIMEOUT", "30s")

	v.SetDefault("POSTGRES_MAX_OPEN_CONNS", 25)
	v.SetDefault("POSTGRES_MAX_IDLE_CONNS", 10)
	v.SetDefault("POSTGRES_CONN_MAX_LIFETIME", "5m")

	v.SetDefault("MSSQL_MAX_OPEN_CONNS", 10)
	v.SetDefault("ORACLE_MAX_OPEN_CONNS", 10)

	v.SetDefault("REDIS_ADDR", "localhost:6379")
	v.SetDefault("REDIS_DB", 0)
	v.SetDefault("REDIS_TTL", "15m")

	v.SetDefault("KAFKA_PRODUCT_TOPIC", "automotive.products")
	v.SetDefault("KAFKA_INVENTORY_TOPIC", "automotive.inventory")
	v.SetDefault("KAFKA_FITMENT_TOPIC", "automotive.fitments")
	v.SetDefault("KAFKA_CONSUMER_GROUP", "catalog-service")
	v.SetDefault("KAFKA_COMMIT_INTERVAL", "1s")

	v.SetDefault("AWS_REGION", "us-east-1")
}
