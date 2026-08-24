package backupCompose

import (
	composeType "github.com/compose-spec/compose-go/types"
)

type composeYaml struct {
	Services map[string]composeType.ServiceConfig `yaml:"services" json:"services"`
	Networks map[string]composeType.NetworkConfig `yaml:"networks,omitempty" json:"networks,omitempty"`
	Warnings []string                             `yaml:"x-docker-copilot-warnings,omitempty" json:"x-docker-copilot-warnings,omitempty"`
}
