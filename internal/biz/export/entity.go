package export

type Export struct {
	Id           uint32  `json:"id"`
	UserId       uint32  `json:"userId"`
	DepartmentId uint32  `json:"departmentId"`
	Scene        string  `json:"scene"`
	Name         string  `json:"name"`
	Size         uint32  `json:"size"`
	Sha          string  `json:"sha"`
	Src          string  `json:"src"`
	URL          string  `json:"url"`
	Status       string  `json:"status"`
	Reason       *string `json:"reason"`
	ExpiredAt    int64   `json:"expiredAt"`
	CreatedAt    int64   `json:"createdAt"`
	UpdatedAt    int64   `json:"updatedAt"`
}
