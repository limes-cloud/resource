package export

type PageExportRequest struct {
	Page     uint32 `json:"page"`
	PageSize uint32 `json:"page_size"`
	UserId   uint32 `json:"user_id"`
}

type ExportFile struct {
	Sha    string `json:"sha"`
	Rename string `json:"rename"`
}

type ExportExcel struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type AddExportRequest struct {
	Name  string        `json:"name"`
	Files []*ExportFile `json:"files"`
	Ids   []uint32      `json:"ids"`
}

type AddExportExcelRequest struct {
	Name string           `json:"name"`
	Rows [][]*ExportExcel `json:"rows"`
}
