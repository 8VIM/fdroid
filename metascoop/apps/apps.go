package apps

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type AppFile struct {
	BuildToolsVersion string              `yaml:"build_tools_version"`
	Apps              map[string]*AppInfo `yaml:"apps"`
}

func (a *AppFile) Apks() map[string]*AppInfo { return a.Apps }

type AppInfo struct {
	GitURL  string `yaml:"git"`
	Summary string `yaml:"summary"`

	AuthorName string `yaml:"author"`
	repoAuthor string

	FriendlyName string `yaml:"name"`
	keyName      string

	Description string `yaml:"description"`

	Categories []string `yaml:"categories"`

	AntiFeatures []string `yaml:"anti_features"`

	ReleaseDescription string

	License      string
	Website      string `yaml:"website"`
	IssueTracker string `yaml:"issue_tracker"`
	Debug        bool
}

func (a *AppInfo) Name() string {
	return a.keyName
}

func (a *AppInfo) Author() string {
	if a.AuthorName != "" {
		return a.AuthorName
	}
	return a.repoAuthor
}

// ParseAppFile returns the list of apps from the app file
func ParseAppFile(filepath string) (appFile *AppFile, err error) {
	f, err := os.Open(filepath)
	if err != nil {
		return
	}
	defer f.Close()

	err = yaml.NewDecoder(f).Decode(&appFile)
	if err != nil {
		return
	}

	for k, a := range appFile.Apps {
		a.keyName = k

		u, uerr := url.ParseRequestURI(a.GitURL)
		if uerr != nil {
			err = fmt.Errorf("problem with given git URL %q for app with key=%q, name=%q: %w", a.GitURL, k, a.Name(), uerr)
			return
		}

		split := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(split) == 0 {
			return
		}
		a.repoAuthor = split[0]
	}
	return
}
