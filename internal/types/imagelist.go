package types

type Image struct {
	Containers  int               `json:"Containers"`
	Created     int64             `json:"Created"`
	ID          string            `json:"Id"`
	Labels      map[string]string `json:"Labels"`
	ParentID    string            `json:"ParentId"`
	RepoDigests []string          `json:"RepoDigests"`
	RepoTags    []string          `json:"RepoTags"`
	SharedSize  int               `json:"SharedSize"`
	Size        int               `json:"Size"`
	VirtualSize int               `json:"VirtualSize"`
	ImageName   string            `json:"image_name"`
	ImageTag    string            `json:"image_tag"`
	States      int               `json:"States"`
	SizeFormat  string            `json:"SizeFormat"`
}
