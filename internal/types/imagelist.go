package types

type Image struct {
	Containers  int64             `json:"Containers"`
	Created     int64             `json:"Created"`
	ID          string            `json:"Id"`
	Labels      map[string]string `json:"Labels"`
	ParentID    string            `json:"ParentId"`
	RepoDigests []string          `json:"RepoDigests"`
	RepoTags    []string          `json:"RepoTags"`
	SharedSize  int64             `json:"SharedSize"`
	Size        int64             `json:"Size"`
	VirtualSize int64             `json:"VirtualSize"`
	ImageName   string            `json:"imageName"`
	ImageTag    string            `json:"imageTag"`
	States      int               `json:"States"`
	SizeFormat  string            `json:"SizeFormat"`
}
