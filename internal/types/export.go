package types

type ListExportRequest struct {
	Page          uint32   `json:"page"`
	PageSize      uint32   `json:"pageSize"`
	Order         *string  `json:"order"`
	OrderBy       *string  `json:"orderBy"`
	All           bool     `json:"all"`
	UserIds       []uint32 `json:"userIds"`
	DepartmentIds []uint32 `json:"departmentIds"`
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
	UserId       uint32              `json:"user_id"`
	DepartmentId uint32              `json:"department_id"`
	Scene        string              `json:"scene"`
	Name         string              `json:"name"`
	Files        []*ExportFileItem   `json:"files"`
	Headers      []string            `json:"headers"`
	Rows         [][]*ExportExcelCol `json:"rows"`
}

type ExportExcelReply struct {
	Id  uint32 `json:"id"`
	Sha string `json:"sha"`
	Src string `json:"src"`
}

type CopyExportRequest struct {
	UserId       uint32 `json:"user_id"`
	DepartmentId uint32 `json:"department_id"`
	Scene        string `json:"scene"`
	Name         string `json:"name"`
}

type ExportFileItem struct {
	Value  string `json:"value"`
	Rename string `json:"rename"`
}

type ExportFileRequest struct {
	UserId       uint32            `json:"userId"`
	DepartmentId uint32            `json:"departmentId"`
	Scene        string            `json:"scene"`
	Name         string            `json:"name"`
	Files        []*ExportFileItem `json:"files"`
	Ids          []uint32          `json:"ids"`
}

type ExportFileReply struct {
	Id uint32 `json:"id"`
}
