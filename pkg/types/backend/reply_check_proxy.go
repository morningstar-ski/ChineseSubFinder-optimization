package backend

import "time"

type ReplyCheckStatus struct {
	SubSiteStatus []SiteStatus `json:"sub_site_status"`
}

type SiteStatus struct {
	Name          string    `json:"name"`
	Valid         bool      `json:"valid"`
	Speed         int64     `json:"speed"`
	Enabled       bool      `json:"enabled,omitempty"`
	RuntimeMode   string    `json:"runtime_mode,omitempty"`
	Reason        string    `json:"reason,omitempty"`
	LastCheckedAt time.Time `json:"last_checked_at,omitempty"`
}
