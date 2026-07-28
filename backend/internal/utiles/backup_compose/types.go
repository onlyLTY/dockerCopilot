package backupCompose

import (
	composeType "github.com/compose-spec/compose-go/types"
)

type composeYaml struct {
	Services map[string]composeType.ServiceConfig `yaml:"services" json:"services"`
}
