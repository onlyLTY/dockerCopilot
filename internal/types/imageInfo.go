package types

import (
	docker "github.com/docker/docker/api/types"
)

type Image struct {
	docker.ImageSummary
	ImageName  string `json:"imageName"`
	ImageTag   string `json:"imageTag"`
	InUsed     bool   `json:"inUsed"`
	SizeFormat string `json:"sizeFormat"`
}
