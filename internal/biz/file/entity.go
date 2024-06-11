package file

type File struct {
	Id          uint32 `json:"id"`
	DirectoryId uint32 `json:"directoryId"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Size        uint32 `json:"size"`
	Sha         string `json:"sha"`
	Key         string `json:"key"`
	Src         string `json:"src"`
	URL         string `json:"url"`
	Status      string `json:"status"`
	UploadId    string `json:"uploadId"`
	ChunkCount  uint32 `json:"chunkCount"`
	CreatedAt   int64  `json:"createdAt"`
	UpdatedAt   int64  `json:"updatedAt"`
}

type DirectoryLimit struct {
	DirectoryId uint32   `json:"directoryId"`
	Accepts     []string `json:"accepts"`
	MaxSize     uint32   `json:"maxSize"`
}
