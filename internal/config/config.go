package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

// configurations for hte server
type ServerConfig struct {
	Host string
	Port uint16
	
	ReadTimeout time.Duration
	WriteTimeout time.Duration
	IdleTimeout time.Duration
	ShutdownTimeout time.Duration

	// this might come in production or if ther's a need for a client
	// to access the site over the internet
	TLSEnabled bool
	TLSCertFile string
	TLSKeyFile string
}

type PostgresConfig struct {
	DSN string
	PoolSizeLimit uint8
}

type RedisConfig struct {
	Address string      // e.g. "localhost:6379"
	Password string		// by default it's not set, so it'll be ""
	DBIndex uint8         // this will make sense later
}

type AuthConfig struct {
	JWTSecret string
	RefreshTTL uint8
}

type RateLimitsConfig struct {
	count uint8
}

// for persistent storage
// MinIO for local development and S3 for production for example
type StorageConfig struct {
	Endpoint string
	Region string
	Bucket string
	AccessKeyID string
	SecretAccessKey string
	UseSSL bool
}

// and now a whole sum Config for instanciating all those configurations
type Config struct {
	server ServerConfig
	postgres PostgresConfig
	redis RedisConfig
	auth AuthConfig
	ratelimits RateLimitsConfig
	storage StorageConfig
}

// and a whole sum function to load all the config and return a whole sum Config
func Load() Config {
	// first we fuck the server configurations
	var server ServerConfig
	server.Host = os.Getenv("DATABASE_URL")
	sport, err := strconv.Atoi(os.Getenv("SERVER_PORT"))
	if err != nil {
		log.Fatal("failed to read env server port: ", err)
	}

	server.Port = uint16(sport)
	server.ReadTimeout = time.Second * 10
	server.WriteTimeout = time.Second * 20
	server.IdleTimeout = time.Second * 30
	server.ShutdownTimeout = time.Second * 60
	server.TLSEnabled = false
	// these being empty for the moment is allowed
	server.TLSCertFile = os.Getenv("TLSCERTFILE")
	server.TLSKeyFile = os.Getenv("TLSKEYFILE")

	// next we fuck postgres
	dburl := os.Getenv("DATABASE_URL")
	if dburl == "" {
		log.Printf("database can't be null error, proceeding anyway....")
	}
	var postgres PostgresConfig = PostgresConfig{
		DSN: dburl,
		PoolSizeLimit: 20,
	}

	// fuck redis!
	redis_url := os.Getenv("REDIS_URL")
	redis_password := os.Getenv("REDIS_PASSWORD")
	var redis = RedisConfig {
		Address: redis_url,
		Password: redis_password,
		// worth taking a look later if there are more need for database instances inside one redis cluster
		DBIndex: 0,
	}

	var auth = AuthConfig {
		// critical, shouldn't be empty
		JWTSecret: "",
		RefreshTTL: 20,   // these are minutes not uncles or aunties
	}

	var ratelimit = RateLimitsConfig {
		count: uint8(30),		// 30 just came to mind, worth knowing real world limits
	}

	var persistent_storage = StorageConfig {
		Endpoint: "",
		Region: "",
		Bucket: "",
		AccessKeyID: "",
		SecretAccessKey: "",
		UseSSL: false,
	}

	config := Config {
		server: server,
		postgres: postgres,
		redis: redis,
		auth: auth,
		ratelimits: ratelimit,
		storage: persistent_storage,
	}

	return config
}
