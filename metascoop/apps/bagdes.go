package apps

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v3"
)

type debugVersion struct {
	versionCode int
	versionName string
}

func (a *AppFile) GenerateBadges(repoDir string) (err error) {
	badges := make(map[string]string)
	fdroidIndexFilePath := filepath.Join(repoDir, "index-v1.json")
	index, err := ReadIndex(fdroidIndexFilePath)
	if err != nil {
		return err
	}
	debugs := make(map[string][]debugVersion)

	for _, packages := range index.Packages {
	next:
		for _, p := range packages {
			for name, app := range a.Apps {
				if _, ok := badges[name]; !ok && strings.HasPrefix(p.ApkName, name) {
					latest, _ := index.FindLatestPackage(p.PackageName)
					version := fmt.Sprintf("v%s", latest.VersionName)
					if !semver.IsValid(version) {
						version = latest.VersionName
					}
					if app.Debug {
						if _, ok := debugs[name]; !ok {
							debugs[name] = make([]debugVersion, 0)
						}
						debugs[name] = append(debugs[name], debugVersion{versionCode: latest.VersionCode, versionName: version})
					} else {
						badges[name] = version
					}
					break next
				}
			}
		}
	}
	for name, debug := range debugs {
		if len(debug) == 0 {
			continue
		}
		slices.SortFunc(debug, func(a, b debugVersion) int {
			return cmp.Compare(b.versionCode, a.versionCode)
		})
		badges[name] = debug[0].versionName
	}
	f, err := os.Create(filepath.Join(filepath.Dir(filepath.Dir(repoDir)), "badges.yaml"))
	if err != nil {
		return
	}
	defer f.Close()
	err = yaml.NewEncoder(f).Encode(badges)
	return
}
