package types

type GetFileRequest struct {
	Src    string `json:"src"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Mode   string `json:"mode"`
}

type GetFileResponse struct {
	Data []byte
	Mime string
}
