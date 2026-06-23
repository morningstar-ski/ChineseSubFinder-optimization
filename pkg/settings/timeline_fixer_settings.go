package settings

type TimelineFixerSettings struct {
	MaxOffsetTime int     `json:"max_offset_time"` // 最大支持校正时间偏移的范围，单位秒
	MinOffset     float64 `json:"min_offset"`      // 最小的时间片校正偏移，低于这个（正负）就跳过不校正，单位秒
	Engine        string  `json:"engine"`          // 时间轴修复引擎，默认 ffsubsync
}

func NewTimelineFixerSettings() *TimelineFixerSettings {
	return &TimelineFixerSettings{
		MaxOffsetTime: 700,
		MinOffset:     0.2,
		Engine:        TimelineFixerEngineFFSubSync,
	}
}

func (t *TimelineFixerSettings) Check() {
	if t.MaxOffsetTime <= 0 || t.MaxOffsetTime > 700 {
		t.MaxOffsetTime = 700 // 60s
	}

	if t.MinOffset <= 0 || t.MinOffset > 1 {
		t.MinOffset = 0.2 // 100ms
	}

	if t.Engine == "" {
		t.Engine = TimelineFixerEngineFFSubSync
	}
	if t.Engine != TimelineFixerEngineFFSubSync {
		t.Engine = TimelineFixerEngineFFSubSync
	}
}

const TimelineFixerEngineFFSubSync = "ffsubsync"
