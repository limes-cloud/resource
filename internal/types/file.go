package types

type GetFileBytesRequest struct {
	Id  *uint32 `json:"id"`
	Sha *string `json:"sha"`
	Src *string `json:"src"`
}

type GetFileBytesFunc func([]byte) error

type GetFileRequest struct {
	Id  *uint32 `json:"id"`
	Sha *string `json:"sha"`
	Src *string `json:"src"`
}

type ListFileRequest struct {
	Page        uint32   `json:"page"`
	PageSize    uint32   `json:"pageSize"`
	Order       *string  `json:"order"`
	OrderBy     *string  `json:"orderBy"`
	DirectoryId *uint32  `json:"directoryId"`
	Status      *string  `json:"status"`
	Name        *string  `json:"name"`
	ShaList     []string `json:"shaList"`
}

type PrepareUploadFileRequest struct {
	Store         *string `json:"store"`
	DirectoryId   *uint32 `json:"directoryId"`
	DirectoryPath *string `json:"directoryPath"`
	Name          string  `json:"name"`
	Size          uint32  `json:"size"`
	Sha           string  `json:"sha"`
}

type PrepareUploadFileReply struct {
	Uploaded     bool     `json:"uploaded"`
	Src          *string  `json:"src"`
	ChunkSize    *uint32  `json:"chunkSize"`
	ChunkCount   *uint32  `json:"chunkCount"`
	UploadId     *string  `json:"uploadId"`
	UploadChunks []uint32 `json:"uploadChunks"`
	Sha          *string  `json:"sha"`
	Url          *string  `json:"url"`
}

type UploadFileRequest struct {
	Data     []byte `json:"data"`
	UploadId string `json:"uploadId"`
	Index    uint32 `json:"index"`
}

type UploadFileReply struct {
	Src string `json:"src"`
	Sha string `json:"sha"`
	Url string `json:"url"`
}
