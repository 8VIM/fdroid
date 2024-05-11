package apps

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/semver"
)

func (a *AppFile) GenerateBadges(repoDir string) (err error) {
	badges := make(map[string]string)
	fdroidIndexFilePath := filepath.Join(repoDir, "index-v1.json")
	index, err := ReadIndex(fdroidIndexFilePath)
	if err != nil {
		return err
	}
	for _, packages := range index.Packages {
	next:
		for _, p := range packages {
			for name := range a.apps {
				if _, ok := badges[name]; !ok && strings.HasPrefix(p.ApkName, name) {
					latest, _ := index.FindLatestPackage(p.PackageName)
					version := fmt.Sprintf("v%s", latest.VersionName)
					if !semver.IsValid(version) {
						version = latest.VersionName
					}
					badges[name] = version
					break next
				}
			}
		}
	}
	f, err := os.Create(filepath.Join(filepath.Dir(filepath.Dir(repoDir)), "badges.json"))
	if err != nil {
		return
	}
	defer f.Close()
	err = json.NewEncoder(f).Encode(badges)
	return
}
