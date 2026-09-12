package version

var (
	Version = "dev"
	Commit  = ""
	Date    = ""
)

type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit,omitempty"`
	Date    string `json:"date,omitempty"`
}

func Get() Info {
	return Info{Version: Version, Commit: Commit, Date: Date}
}
