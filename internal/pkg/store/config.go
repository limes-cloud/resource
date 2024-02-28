package store

import "gorm.io/gorm"

type Config struct {
	Endpoint string
	Key      string
	Secret   string
	Bucket   string
	LocalDir string
	DB       *gorm.DB
}
