package export

import "github.com/limes-cloud/kratosx/types"

const (
	StatusProcess = "process"
	StatusFinish  = "finish"
	StatusFail    = "fail"
	StatusExpire  = "expire"
)

type Export struct {
	types.BaseModel
	UserId  uint32 `json:"user_id"`
	Name    string `json:"name"`
	Size    uint32 `json:"size"`
	Src     string `json:"src"`
	Version string `json:"version"`
	Reason  string `json:"reason"`
	Status  string `json:"status"`
}
