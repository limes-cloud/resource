package file

import "github.com/limes-cloud/kratosx/types"

type Directory struct {
	types.BaseModel
	ParentID uint32 `json:"parent_id"`
	Name     string `json:"name"`
	App      string `json:"app"`
}

type File struct {
	types.BaseModel
	DirectoryID uint32     `json:"directory_id"`
	Name        string     `json:"name"`
	Type        string     `json:"type"`
	Size        uint32     `json:"size"`
	Sha         string     `json:"sha"`
	Src         string     `json:"src"`
	UploadID    *string    `json:"upload_id"`
	ChunkCount  uint32     `json:"chunk_count"`
	Storage     string     `json:"storage"`
	Status      string     `json:"status"`
	Directory   *Directory `json:"directory"`
}
