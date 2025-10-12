package types

type ListExportRequest struct {
	Page      uint32  `json:"page"`
	PageSize  uint32  `json:"pageSize"`
	Order     *string `json:"order"`
	OrderBy   *string `json:"orderBy"`
	Name      *string `json:"name"`
	Status    *string `json:"status"`
	ExpiredAt *int64  `json:"expiredAt"`
}

type GetExportFileCountRequest struct {
	Sha    string `json:"sha"`
	Status string `json:"status"`
}

type GetExportRequest struct {
	Id  *uint32 `json:"id"`
	Sha *string `json:"sha"`
}

type ExportExcelCol struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type ExportExcelRequest struct {
	Name    string              `json:"name"`
	Files   []*ExportFileItem   `json:"files"`
	Headers []string            `json:"headers"`
	Rows    [][]*ExportExcelCol `json:"rows"`
}

type ExportExcelReply struct {
	Id  uint32 `json:"id"`
	Sha string `json:"sha"`
	Key string `json:"key"`
}

type CopyExportRequest struct {
	Name string `json:"name"`
}

type ExportFileItem struct {
	Value  string `json:"value"`
	Rename string `json:"rename"`
}

type ExportFileRequest struct {
	Name  string            `json:"name"`
	Files []*ExportFileItem `json:"files"`
	Ids   []uint32          `json:"ids"`
}

type ExportFileReply struct {
	Id uint32 `json:"id"`
}
