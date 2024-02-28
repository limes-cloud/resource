package local

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"unsafe"

	"github.com/google/uuid"
	"github.com/limes-cloud/kratosx/pkg/util"
	"gorm.io/gorm"

	store2 "github.com/limes-cloud/resource/internal/pkg/store"
)

type local struct {
	dir string
	db  *gorm.DB
}

type upload struct {
	key   string
	uuid  string
	local *local
}

func New(conf *store2.Config) (store2.Store, error) {
	if conf.LocalDir == "" {
		return nil, errors.New("upload config error")
	}
	return &local{
		dir: conf.LocalDir,
		db:  conf.DB,
	}, nil
}

func (s *local) PutBytes(key string, in []byte) error {
	return s.Put(key, bytes.NewReader(in))
}

func (s *local) Put(key string, r io.Reader) error {
	path := s.dir + "/" + key
	if err := s.makeDir(path); err != nil {
		return err
	}

	saveFile, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	defer saveFile.Close()
	_, err = io.Copy(saveFile, r)
	return nil
}

func (s *local) PutFromLocal(key string, localPath string) error {
	path := s.dir + "/" + key
	if err := s.makeDir(path); err != nil {
		return err
	}
	return os.Rename(localPath, path)
}

func (s *local) Get(key string) (io.ReadCloser, error) {
	path := s.dir + "/" + key
	return os.OpenFile(path, os.O_RDONLY, os.ModePerm)
}

func (s *local) Delete(key string) error {
	return os.Remove(s.dir + "/" + key)
}

func (s *local) Size(key string) (int64, error) {
	path := s.dir + "/" + key
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func (s *local) Exists(key string) (bool, error) {
	_, err := os.Stat(key)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (s *local) makeDir(path string) error {
	dir := path[:strings.LastIndex(path, "/")]
	if is, err := s.Exists(dir); !is {
		if err != nil {
			return err
		}
		if err = os.MkdirAll(dir, os.ModePerm); err != nil {
			return err
		}
	}
	return nil
}

func (s *local) NewPutChunk(key string) (store2.PutChunk, error) {
	return &upload{
		uuid:  uuid.NewString(),
		local: s,
		key:   key,
	}, nil
}

func (s *local) NewPutChunkByUploadID(key, id string) (store2.PutChunk, error) {
	return &upload{
		uuid:  id,
		local: s,
		key:   key,
	}, nil
}

func (u *upload) ChunkCount() int {
	chunk := Chunk{}
	chunks, _ := chunk.Parts(u.local.db, u.uuid)
	return len(chunks)
}

func (u *upload) UploadedChunkIndex() []int {
	var arr []int
	chunk := Chunk{}
	chunks, _ := chunk.Parts(u.local.db, u.uuid)
	for _, item := range chunks {
		arr = append(arr, item.Index)
	}
	return arr
}

func (u *upload) UploadID() string {
	return u.uuid
}

func (u *upload) Append(r io.Reader, index int) error {
	all, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	sha := util.Sha256(all)

	oldChunk := Chunk{}
	// 查询是否已经存在数据
	if err := oldChunk.OneBySha(u.local.db, sha); err == nil {
		return oldChunk.Copy(u.local.db, u.uuid, index)
	}

	chunk := Chunk{
		UploadID: u.uuid,
		Index:    index,
		Sha:      util.Sha256(all),
		Size:     len(all),
		Data:     *(*string)(unsafe.Pointer(&all)),
	}

	return chunk.Add(u.local.db)
}

func (u *upload) AppendBytes(r []byte, index int) error {
	chunk := Chunk{
		UploadID: u.uuid,
		Index:    index,
		Size:     len(r),
		Sha:      util.Sha256(r),
		Data:     *(*string)(unsafe.Pointer(&r)),
	}

	return chunk.Add(u.local.db)
}

func (u *upload) Abort() error {
	chunk := Chunk{}
	return chunk.Delete(u.local.db, u.uuid)
}

func (u *upload) Complete() error {
	chunk := Chunk{}
	chunks, err := chunk.Parts(u.local.db, u.uuid)
	if err != nil {
		return err
	}
	path := u.local.dir + "/" + u.key
	if err := u.local.makeDir(path); err != nil {
		return err
	}

	// 如果已经存在文件了，则直接删除
	_ = os.Remove(path)

	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, item := range chunks {
		rt := []byte(item.Data)

		if _, err = file.Write(rt); err != nil {
			_ = os.Remove(u.local.dir + "/" + u.key)
			return err
		}
	}
	_ = chunk.Delete(u.local.db, u.uuid)
	return nil
}
