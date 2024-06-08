package store

import (
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type Config struct {
	Endpoint        string
	Id              string
	Secret          string
	Bucket          string
	DB              *gorm.DB
	Cache           *redis.Client
	LocalDir        string
	ServerURL       string
	TemporaryExpire time.Duration
}
