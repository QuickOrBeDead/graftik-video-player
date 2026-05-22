package plugin

type DiscoveryConfig struct {
	Type    string   `json:"type"`            // "exec" or "lua", defaults to "exec"
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Version string   `json:"version"`
	Command string   `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
}

type Manifest struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Version     string      `json:"version"`
	MenuEntries []MenuEntry `json:"menuEntries"`
	UI          string      `json:"ui,omitempty"`
}

type MenuEntry struct {
	Label  string `json:"label"`
	Action string `json:"action"`
}

type PluginInfo struct {
	ID      string      `json:"id"`
	Name    string      `json:"name"`
	Version string      `json:"version"`
	Status  string      `json:"status"`
	Menu    []MenuEntry `json:"menu"`
	UI      string      `json:"ui,omitempty"`
}

type PlaylistRequest struct {
	Path  string `json:"path"`
	Title string `json:"title,omitempty"`
}
