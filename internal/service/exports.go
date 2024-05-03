package service

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/limes-cloud/kratosx"
	"github.com/limes-cloud/kratosx/pkg/util"

	"github.com/limes-cloud/resource/api/errors"
	pb "github.com/limes-cloud/resource/api/export/v1"
	biz "github.com/limes-cloud/resource/internal/biz/export"
	"github.com/limes-cloud/resource/internal/config"
	data "github.com/limes-cloud/resource/internal/data/export"
	"github.com/limes-cloud/resource/internal/data/file"
	"github.com/limes-cloud/resource/internal/factory"
)

type ExportService struct {
	pb.UnimplementedServiceServer
	uc   *biz.UseCase
	conf *config.Config
}

func NewExport(conf *config.Config) *ExportService {
	return &ExportService{
		conf: conf,
		uc:   biz.NewUseCase(conf, data.NewRepo(), factory.New(conf, file.NewRepo(), data.NewRepo())),
	}
}

func (fs *ExportService) Config() *config.Config {
	return fs.conf
}

// PageExport 文件分野查询
func (fs *ExportService) PageExport(ctx context.Context, in *pb.PageExportRequest) (*pb.PageExportReply, error) {
	req := biz.PageExportRequest{}
	if err := util.Transform(in, &req); err != nil {
		return nil, errors.TransformFormat(err.Error())
	}

	list, total, err := fs.uc.PageExport(kratosx.MustContext(ctx), &req)
	if err != nil {
		return nil, err
	}

	reply := pb.PageExportReply{Total: &total}
	if err := util.Transform(list, &reply.List); err != nil {
		return nil, errors.TransformFormat(err.Error())
	}
	return &reply, nil
}

// AddExport 删除文件
func (fs *ExportService) AddExport(ctx context.Context, in *pb.AddExportRequest) (*pb.AddExportReply, error) {
	if len(in.Ids) == 0 && len(in.Files) == 0 {
		return nil, errors.Params()
	}
	req := biz.AddExportRequest{}
	if err := util.Transform(in, &req); err != nil {
		return nil, errors.TransformFormat(err.Error())
	}
	id, err := fs.uc.AddExport(kratosx.MustContext(ctx), &req)
	if err != nil {
		return nil, err
	}
	return &pb.AddExportReply{
		Id: id,
	}, nil
}

// AddExportExcel 删除文件
func (fs *ExportService) AddExportExcel(ctx context.Context, in *pb.AddExportExcelRequest) (*pb.AddExportExcelReply, error) {
	req := biz.AddExportExcelRequest{Name: in.Name}
	for _, row := range in.Rows {
		var temp []*biz.ExportExcel
		for _, col := range row.Cols {
			temp = append(temp, &biz.ExportExcel{
				Type:  col.Type,
				Value: col.Value,
			})
		}
		req.Rows = append(req.Rows, temp)
	}

	id, err := fs.uc.AddExportExcel(kratosx.MustContext(ctx), &req)
	if err != nil {
		return nil, err
	}
	return &pb.AddExportExcelReply{
		Id: id,
	}, nil
}

// DeleteExport 删除文件
func (fs *ExportService) DeleteExport(ctx context.Context, in *pb.DeleteExportRequest) (*empty.Empty, error) {
	return &empty.Empty{}, fs.uc.DeleteExport(kratosx.MustContext(ctx), in.Id)
}
