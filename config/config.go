package config

type Config struct {
	Storage         string
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
