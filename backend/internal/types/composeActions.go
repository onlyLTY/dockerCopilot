package types

type ComposeDeployReq struct {
	ProjectID       string `json:"projectId"`
	Filename        string `json:"filename"`
	ConfirmToken    string `json:"confirmToken"`
	ConfirmWarnings bool   `json:"confirmWarnings"`
	PullImages      bool   `json:"pullImages"`
}

type ComposeDeployPreviewReq struct {
	ProjectID string `json:"projectId"`
	Filename  string `json:"filename"`
}

type ComposeCleanupPreviewReq struct {
	ProjectID string `json:"projectId"`
}

type ComposeCleanupReq struct {
	ProjectID    string `json:"projectId"`
	PreviewToken string `json:"previewToken"`
	DeleteDir    bool   `json:"deleteDir"`
	Confirm      bool   `json:"confirm"`
}
