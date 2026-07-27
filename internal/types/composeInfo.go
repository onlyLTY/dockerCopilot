package types

import "time"

type ComposePort struct {
	HostIP        string `json:"hostIP"`
	HostPort      string `json:"hostPort"`
	ContainerPort string `json:"containerPort"`
	Protocol      string `json:"protocol"`
	Published     bool   `json:"published"`
}

type ComposeContainer struct {
	ID      string        `json:"id"`
	Name    string        `json:"name"`
	Service string        `json:"service"`
	State   string        `json:"state"`
	Ports   []ComposePort `json:"ports"`
}

type ComposeFile struct {
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	Size       int64     `json:"size"`
	ModifiedAt time.Time `json:"modifiedAt"`
	Valid      bool      `json:"valid"`
	Error      string    `json:"error,omitempty"`
}

type ComposeProject struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	Root       string             `json:"root"`
	Files      []ComposeFile      `json:"files"`
	Status     string             `json:"status"`
	Containers []ComposeContainer `json:"containers"`
	Ports      []ComposePort      `json:"ports"`
	Warnings   []string           `json:"warnings,omitempty"`
}

type ComposeProjectsSummary struct {
	Total   int `json:"total"`
	Using   int `json:"using"`
	Stopped int `json:"stopped"`
	Unused  int `json:"unused"`
	Unknown int `json:"unknown"`
}

type ComposeProjectsResponse struct {
	Summary  ComposeProjectsSummary `json:"summary"`
	Projects []ComposeProject       `json:"projects"`
}

type PortUsage struct {
	Project       string `json:"project"`
	ContainerID   string `json:"containerID"`
	ContainerName string `json:"containerName"`
	State         string `json:"state"`
	HostIP        string `json:"hostIP"`
	HostPort      string `json:"hostPort"`
	ContainerPort string `json:"containerPort"`
	Protocol      string `json:"protocol"`
	Published     bool   `json:"published"`
	ConflictKey   string `json:"conflictKey,omitempty"`
}

type PortsResponse struct {
	Ports     []PortUsage `json:"ports"`
	Conflicts []string    `json:"conflicts"`
}
