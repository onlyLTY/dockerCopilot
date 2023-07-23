package types

type HubImageInfo struct {
	Creator             int        `json:"creator"`
	ID                  int        `json:"id"`
	Images              []HubImage `json:"images"`
	LastUpdated         string     `json:"last_updated"`
	LastUpdater         int        `json:"last_updater"`
	LastUpdaterUsername string     `json:"last_updater_username"`
	Name                string     `json:"name"`
	Repository          int        `json:"repository"`
	FullSize            int        `json:"full_size"`
	V2                  bool       `json:"v2"`
	TagStatus           string     `json:"tag_status"`
	TagLastPulled       string     `json:"tag_last_pulled"`
	TagLastPushed       string     `json:"tag_last_pushed"`
	MediaType           string     `json:"media_type"`
	ContentType         string     `json:"content_type"`
	Digest              string     `json:"digest"`
}
type HubImage struct {
	Architecture string      `json:"architecture"`
	Features     string      `json:"features"`
	Variant      interface{} `json:"variant"`
	Digest       string      `json:"digest"`
	OS           string      `json:"os"`
	OSFeatures   string      `json:"os_features"`
	OSVersion    interface{} `json:"os_version"`
	Size         int         `json:"size"`
	Status       string      `json:"status"`
	LastPulled   string      `json:"last_pulled"`
	LastPushed   string      `json:"last_pushed"`
}
