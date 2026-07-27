package types

type ComposeProjectCreateReq struct {
	ProjectName string `json:"projectName"`
	Filename    string `json:"filename"`
	Content     string `json:"content"`
}

type ComposeProjectFileReq struct {
	ProjectID string `path:"id"`
	Filename  string `path:"filename"`
}

type ComposeProjectFileUpdateReq struct {
	ProjectID string `path:"id"`
	Filename  string `path:"filename"`
	Content   string `json:"content"`
	Version   string `json:"version"`
}

type ComposeProjectValidateReq struct {
	ProjectID string `json:"projectId"`
	Filename  string `json:"filename"`
	Content   string `json:"content"`
}

type ComposeProjectVersionReq struct {
	ProjectID string `path:"id"`
	Version   string `path:"version"`
}
