package directory

type GetDirectoryRequest struct {
	Id *uint32 `json:"id"`
}

type ListDirectoryRequest struct {
	Order   *string `json:"order"`
	OrderBy *string `json:"orderBy"`
}
