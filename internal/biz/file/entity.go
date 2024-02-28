package file

import "github.com/limes-cloud/kratosx/types"

type Directory struct {
	types.BaseModel
	ParentID uint32 `json:"parent_id" gorm:"uniqueIndex:pna;not null;comment:父id"`
	Name     string `json:"name" gorm:"uniqueIndex:pna;not null;size:128;comment:目录名称"`
	App      string `json:"app" gorm:"uniqueIndex:pna;not null;size:32;comment:所属应用"`
}

type File struct {
	types.BaseModel
	DirectoryID uint32     `json:"directory_id" gorm:"uniqueIndex:dir_name;uniqueIndex:dir_sha;not null;comment:目录id"`
	Name        string     `json:"name" gorm:"uniqueIndex:dir_name;not null;size:128;comment:文件名称"`
	Type        string     `json:"type" gorm:"not null;size:32;comment:文件类型"`
	Size        uint32     `json:"size" gorm:"not null;comment:文件大小"`
	Sha         string     `json:"sha" gorm:"uniqueIndex:dir_sha;not null;size:128;comment:文件sha"`
	Src         string     `json:"src" gorm:"size:256;comment:文件真实路径"`
	UploadID    *string    `json:"upload_id" gorm:"uniqueIndex;size:128;comment:上传id"`
	ChunkCount  uint32     `json:"chunk_count" gorm:"default:1;comment:切片数量"`
	Storage     string     `json:"storage" gorm:"not null;size:32;comment:存储引擎"`
	Status      string     `json:"status" gorm:"default:PROGRESS;size:32;comment:上传状态"`
	Directory   *Directory `json:"directory" gorm:"constraint:onDelete:cascade"`
}
