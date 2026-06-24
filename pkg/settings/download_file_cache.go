package settings

type DownloadFileCache struct {
	TTL  int    `json:"ttl" default:"180"` // 默认按天填写，默认值为 180 天
	Unit string `json:"unit" default:"day"` // day, hour; second is kept as a legacy-compatible unit
}

func NewDownloadFileCache() *DownloadFileCache {
	return &DownloadFileCache{TTL: 180, Unit: "day"}
}

func (d DownloadFileCache) Check() {
	if d.Unit == "second" {
		// 兼容旧配置，按 180-365 天范围换算秒数
		if d.TTL < 15552000 || d.TTL > 31536000 {
			d.TTL = 15552000
		}
	}
	if d.Unit == "hour" {
		// 180-365 天的小时数
		if d.TTL < 4320 || d.TTL > 8760 {
			d.TTL = 4320
		}
	}
	if d.Unit != "second" && d.Unit != "hour" && d.Unit != "day" {
		d.Unit = "day"
		d.TTL = 180
	}
	if d.Unit == "day" {
		if d.TTL < 180 || d.TTL > 365 {
			d.TTL = 180
		}
	}
}
