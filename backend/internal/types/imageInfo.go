package types

import (
	"github.com/docker/docker/api/types/image"
)

type Image struct {
	image.Summary
	ImageName  string `json:"imageName"`
	ImageTag   string `json:"imageTag"`
	InUsed     bool   `json:"inUsed"`
	SizeFormat string `json:"sizeFormat"`
}
