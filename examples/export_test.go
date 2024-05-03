package examples

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/limes-cloud/kratosx"
	"github.com/zeebo/assert"

	exportpb "github.com/limes-cloud/resource/api/export/v1"
)

func TestExport(t *testing.T) {
	ctx := kratosx.MustContext(context.Background())
	insecure, err := grpc.DialInsecure(ctx, grpc.WithEndpoint("127.0.0.1:8003"))
	assert.NoError(t, err)

	client := exportpb.NewServiceClient(insecure)
	_, err = client.AddExport(ctx, &exportpb.AddExportRequest{
		Name: "test.zip",
		Files: []*exportpb.AddExportRequest_ExportFile{
			{
				Sha:    "2a0786fe9127b8116bc30ed2ce9581e2",
				Rename: "1",
			},
			{
				Sha:    "36e2e87f7b73219343da52a28ba47eec",
				Rename: "2",
			},
			{
				Sha:    "6d06733ef579fbcef68b9f95745a3e99",
				Rename: "3",
			},
		},
	})
	assert.NoError(t, err)
}

func TestExportExcel(t *testing.T) {
	ctx := kratosx.MustContext(context.Background())
	insecure, err := grpc.DialInsecure(ctx, grpc.WithEndpoint("127.0.0.1:8003"))
	assert.NoError(t, err)

	client := exportpb.NewServiceClient(insecure)
	_, err = client.AddExportExcel(ctx, &exportpb.AddExportExcelRequest{
		Name: "test.zip",
		Rows: []*exportpb.AddExportExcelRequest_Row{
			{
				Cols: []*exportpb.AddExportExcelRequest_Col{
					{
						Type:  "string",
						Value: "hello",
					},
					{
						Type:  "string",
						Value: "world",
					},
					{
						Type:  "image",
						Value: "2a0786fe9127b8116bc30ed2ce9581e2",
					},
				},
			},
			{
				Cols: []*exportpb.AddExportExcelRequest_Col{
					{
						Type:  "string",
						Value: "hello1",
					},
					{
						Type:  "string",
						Value: "world2",
					},
					{
						Type:  "image",
						Value: "36e2e87f7b73219343da52a28ba47eec",
					},
				},
			},
		},
	})
	assert.NoError(t, err)
}
