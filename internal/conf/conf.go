package conf

import "time"

type Config struct {
	// Secret             string
	// Expire             time.Duration
	DefaultMaxSize uint32

	DefaultAcceptTypes []string
	ChunkSize          uint32
	Export             struct {
		ServerURL string
		LocalDir  string
		Expire    time.Duration
	}
	Storage struct {
		Type            string
		Endpoint        string
		Id              string
		Secret          string
		Bucket          string
		Region          string
		LocalDir        string
		ServerURL       string
		TemporaryExpire time.Duration
	}
}
