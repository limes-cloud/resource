package export

type ListExportRequest struct {
	Page         uint32  `json:"page"`
	PageSize     uint32  `json:"pageSize"`
	Order        *string `json:"order"`
	OrderBy      *string `json:"orderBy"`
	UserId       *uint32 `json:"userId"`
	DepartmentId *uint32 `json:"departmentId"`
}
