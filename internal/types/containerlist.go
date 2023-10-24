package types

import (
	docker "github.com/docker/docker/api/types"
)

type Container struct {
	docker.Container
	Update bool `json:"Update"`
}
