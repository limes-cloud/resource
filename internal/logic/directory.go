package logic

import (
	v1 "resource/api/v1"
	"resource/config"
	"resource/internal/model"
	"resource/pkg/util"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/limes-cloud/kratosx"
)

type Directory struct {
	conf *config.Config
}

func NewDirectory(conf *config.Config) *Directory {
	return &Directory{
		conf: conf,
	}
}

func (f *Directory) Get(ctx kratosx.Context, in *v1.GetDirectoryRequest) (*v1.GetDirectoryReply, error) {
	dir := model.Directory{}
	list, err := dir.AllByParentID(ctx, in.App, in.ParentId)
	if err != nil {
		return nil, v1.DatabaseError()
	}

	reply := &v1.GetDirectoryReply{}
	if err := util.Transform(list, &reply.List); err != nil {
		return nil, v1.TransformError()
	}
	return reply, nil
}

func (f *Directory) Add(ctx kratosx.Context, in *v1.AddDirectoryRequest) (*v1.Directory, error) {
	oldDir := model.Directory{}

	if in.ParentId != 0 {
		if err := oldDir.OneByID(ctx, in.ParentId); err != nil {
			return nil, v1.NotExistDirectoryError()
		}
		if oldDir.App != in.App {
			return nil, v1.SystemError()
		}
	}

	dir := model.Directory{}
	if err := util.Transform(in, &dir); err != nil {
		return nil, v1.TransformError()
	}

	if in.ParentId != 0 && oldDir.OneByID(ctx, in.ParentId) != nil {
		return nil, v1.AddDirectoryErrorFormat("上级目录不存在")
	}
	if oldDir.OneByName(ctx, in.ParentId, in.Name) == nil {
		return nil, v1.AddDirectoryErrorFormat("文件目录已存在")
	}

	if err := dir.Create(ctx); err != nil {
		return nil, v1.DatabaseError()
	}

	reply := &v1.Directory{}
	if err := util.Transform(dir, reply); err != nil {
		return nil, v1.TransformError()
	}
	return reply, nil
}

func (f *Directory) Update(ctx kratosx.Context, in *v1.UpdateDirectoryRequest) (*empty.Empty, error) {
	oldDir := model.Directory{}
	if err := oldDir.OneByID(ctx, in.Id); err != nil {
		return nil, v1.NotExistDirectoryError()
	}

	if oldDir.App != in.App {
		return nil, v1.SystemError()
	}

	dir := model.Directory{}
	if err := util.Transform(in, &dir); err != nil {
		return nil, v1.TransformError()
	}

	oldDir = model.Directory{}
	if oldDir.OneByName(ctx, oldDir.ParentID, oldDir.Name) == nil {
		return nil, v1.UpdateDirectoryErrorFormat("目录名已存在")
	}

	if err := dir.Update(ctx); err != nil {
		return nil, v1.DatabaseError()
	}

	return nil, nil
}

func (f *Directory) Delete(ctx kratosx.Context, in *v1.DeleteDirectoryRequest) (*empty.Empty, error) {
	oldDir := model.Directory{}
	if err := oldDir.OneByID(ctx, in.Id); err != nil {
		return nil, v1.NotExistDirectoryError()
	}

	if oldDir.App != in.App {
		return nil, v1.SystemError()
	}

	file := model.File{}
	count, err := file.CountByDirectoryID(ctx, in.Id)
	if err != nil {
		return nil, v1.DatabaseError()
	}

	if count != 0 {
		return nil, v1.DeleteDirectoryErrorFormat("当前目录下存在文件")
	}

	dir := model.Directory{}
	if err := dir.DeleteByID(ctx, in.Id); err != nil {
		return nil, v1.DatabaseError()
	}

	return nil, nil
}
