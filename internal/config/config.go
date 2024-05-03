package config

import "time"

type Config struct {
	Export struct {
		LocalDir string
		Expire   time.Duration
	}
	Storage struct {
		Type            string
		Endpoint        string
		Key             string
		Secret          string
		Bucket          string
		Region          string
		ServerPath      string
		LocalDir        string
		MaxSingularSize int64
		MaxChunkSize    int64
		MaxChunkCount   int64
		AcceptTypes     []string
	}
}
