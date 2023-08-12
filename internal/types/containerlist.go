package types

type Container struct {
	Command         string          `json:"Command"`
	Created         int             `json:"Created"`
	HostConfig      HostConfig      `json:"HostConfig"`
	ID              string          `json:"Id"`
	Image           string          `json:"Image"`
	ImageID         string          `json:"ImageID"`
	Labels          Labels          `json:"Labels"`
	Mounts          []Mount         `json:"Mounts"`
	Names           []string        `json:"Names"`
	NetworkSettings NetworkSettings `json:"NetworkSettings"`
	Portainer       Portainer       `json:"Portainer"`
	Ports           []Port          `json:"Ports"`
	State           string          `json:"State"`
	Status          string          `json:"Status"`
	Update          bool            `json:"Update"`
}

type HostConfig struct {
	NetworkMode string `json:"NetworkMode"`
}

type Labels struct {
	Created     string `json:"org.opencontainers.image.created"`
	Description string `json:"org.opencontainers.image.description"`
	Licenses    string `json:"org.opencontainers.image.licenses"`
	Revision    string `json:"org.opencontainers.image.revision"`
	Source      string `json:"org.opencontainers.image.source"`
	Title       string `json:"org.opencontainers.image.title"`
	URL         string `json:"org.opencontainers.image.url"`
	Version     string `json:"org.opencontainers.image.version"`
}

type Mount struct {
	Destination string `json:"Destination"`
	Mode        string `json:"Mode"`
	Propagation string `json:"Propagation"`
	RW          bool   `json:"RW"`
	Source      string `json:"Source"`
	Type        string `json:"Type"`
}

type NetworkSettings struct {
	Networks map[string]Network `json:"Networks"`
}

type Network struct {
	Aliases             []string    `json:"Aliases"`
	DriverOpts          interface{} `json:"DriverOpts"`
	EndpointID          string      `json:"EndpointID"`
	Gateway             string      `json:"Gateway"`
	GlobalIPv6Address   string      `json:"GlobalIPv6Address"`
	GlobalIPv6PrefixLen int         `json:"GlobalIPv6PrefixLen"`
	IPAMConfig          interface{} `json:"IPAMConfig"`
	IPAddress           string      `json:"IPAddress"`
	IPPrefixLen         int         `json:"IPPrefixLen"`
	IPv6Gateway         string      `json:"IPv6Gateway"`
	Links               interface{} `json:"Links"`
	MacAddress          string      `json:"MacAddress"`
	NetworkID           string      `json:"NetworkID"`
}

type Portainer struct {
	ResourceControl ResourceControl `json:"ResourceControl"`
}

type ResourceControl struct {
	ID                 int           `json:"Id"`
	ResourceId         string        `json:"ResourceId"`
	SubResourceIds     []interface{} `json:"SubResourceIds"`
	Type               int           `json:"Type"`
	UserAccesses       []UserAccess  `json:"UserAccesses"`
	TeamAccesses       []interface{} `json:"TeamAccesses"`
	Public             bool          `json:"Public"`
	AdministratorsOnly bool          `json:"AdministratorsOnly"`
	System             bool          `json:"System"`
}

type UserAccess struct {
	UserId      int `json:"UserId"`
	AccessLevel int `json:"AccessLevel"`
}

type Port struct {
	IP          string `json:"IP"`
	PrivatePort int    `json:"PrivatePort"`
	PublicPort  int    `json:"PublicPort"`
	Type        string `json:"Type"`
}
