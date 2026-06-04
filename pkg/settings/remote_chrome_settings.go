package settings

type RemoteChromeSettings struct {
	Enable            bool   `json:"enable"`
	RemoteDockerURL   string `json:"remote_docker_url"`    // DevTools ws URL for the remote Chrome instance.
	ReMoteUserDataDir string `json:"remote_user_data_dir"` // Optional remote Chrome user-data directory.
	// Runtime currently consumes only the two fields above. Cache cleanup remains external.
}
