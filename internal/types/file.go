package types

type GetUserFileRequest struct {
	DirectoryId uint32
	UserId      uint32
	FileId      uint32
	Directory   *string `json:"directory"`
	Id          *uint32 `json:"id"`
	Key         *string `json:"key"`
}

type GetFileBytesFunc func([]byte) error

type ListFileRequest struct {
	Page        uint32   `json:"page"`
	PageSize    uint32   `json:"pageSize"`
	Order       *string  `json:"order"`
	OrderBy     *string  `json:"orderBy"`
	DirectoryId *uint32  `json:"directoryId"`
	Status      *string  `json:"status"`
	Name        *string  `json:"name"`
	KeyList     []string `json:"keyList"`
}

type PrepareUploadFileRequest struct {
	DirectoryId   *uint32 `json:"directoryId"`
	DirectoryPath *string `json:"directoryPath"`
	Store         string  `json:"store"`
	Name          string  `json:"name"`
	Size          uint32  `json:"size"`
	Sha           string  `json:"sha"`
	Key           string  `json:"key"`
}

type PrepareUploadFileReply struct {
	Uploaded     bool     `json:"uploaded"`
	ChunkSize    *uint32  `json:"chunkSize"`
	ChunkCount   *uint32  `json:"chunkCount"`
	UploadId     *string  `json:"uploadId"`
	UploadChunks []uint32 `json:"uploadChunks"`
	Sha          *string  `json:"sha"`
	Key          *string  `json:"key"`
}

type UploadFileRequest struct {
	DirectoryId   *uint32 `json:"directoryId"`
	DirectoryPath *string `json:"directoryPath"`
	Store         string  `json:"store"`
	Name          string  `json:"name"`
	Sha           string  `json:"sha"`
	Data          []byte  `json:"data"`
}

type UploadChunkFileRequest struct {
	Data     []byte `json:"data"`
	UploadId string `json:"uploadId"`
	Index    uint32 `json:"index"`
}

type UploadFileReply struct {
	Sha string `json:"sha"`
	Key string `json:"key"`
}

type UploadFileByURLRequest struct {
	DirectoryId   *uint32 `json:"directoryId"`
	DirectoryPath *string `json:"directoryPath"`
	Store         string  `json:"store"`
	URL           string  `json:"url"`
	Name          string  `json:"name"`
}

type UploadFileByURLReply struct {
	Sha string `json:"sha"`
	Key string `json:"key"`
}

type ParserQuery struct {
	Expire   string
	Sign     string
	Width    uint32
	Height   uint32
	Mode     string
	Download bool
	SaveName string
}
