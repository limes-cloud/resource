package file

type PageFileRequest struct {
	Page        uint32 `json:"page"`
	PageSize    uint32 `json:"page_size"`
	Name        string `json:"name"`
	DirectoryId uint32 `json:"directory_id"`
}

type GetDirectoryByAppRequest struct {
	App      string `json:"app"`
	ParentID uint32 `json:"parent_id"`
}

type PrepareUploadFileRequest struct {
	DirectoryId   uint32 `json:"directory_id"`
	DirectoryPath string `json:"directory_path"`
	App           string `json:"app"`
	Name          string `json:"name"`
	Sha           string `json:"sha"`
	Size          uint32 `json:"size"`
}

type PrepareUploadFileReply struct {
	Uploaded     *bool   `json:"uploaded"`
	Src          *string `json:"src"`
	ChunkSize    *uint32 `json:"chunk_size"`
	ChunkCount   *uint32 `json:"chunk_count"`
	UploadId     *string `json:"upload_id"`
	UploadChunks []int   `json:"upload_chunks"`
	Sha          *string `json:"sha"`
}

type UploadFileRequest struct {
	Data     []byte `json:"data"`
	UploadId string `json:"upload_id"`
	Index    uint32 `json:"index"`
}

type UploadFileReply struct {
	Src string
	Sha string
}

type GetFileRequest struct {
	Src     string `json:"src"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	Mode    string `json:"mode"`
	IsRange bool   `json:"is_range"`
	Start   int64  `json:"start"`
	End     int64  `json:"end"`
}

type GetFileResponse struct {
	Data []byte `json:"data"`
	Mime string `json:"mime"`
}
