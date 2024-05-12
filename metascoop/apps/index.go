package apps

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/hashicorp/go-version"
)

type RepoIndex struct {
	Repo     map[string]interface{}   `json:"repo"`
	Requests map[string]interface{}   `json:"requests"`
	Apps     []map[string]interface{} `json:"apps"`

	Packages map[string][]PackageInfo `json:"packages"`
}

type PackageInfo struct {
	Added            int64    `json:"added"`
	ApkName          string   `json:"apkName"`
	Hash             string   `json:"hash"`
	HashType         string   `json:"hashType"`
	MinSdkVersion    int      `json:"minSdkVersion"`
	Nativecode       []string `json:"nativecode"`
	PackageName      string   `json:"packageName"`
	Sig              string   `json:"sig"`
	Signer           string   `json:"signer"`
	Size             int      `json:"size"`
	TargetSdkVersion int      `json:"targetSdkVersion"`
	VersionCode      int      `json:"versionCode,omitempty"`
	VersionName      string   `json:"versionName"`
}

type indexV2 struct {
	Repo     map[string]interface{}    `json:"repo"`
	Packages map[string]indexV2Package `json:"packages"`
}

type indexV2Package struct {
	Metadata map[string]interface{}    `json:"metadata"`
	Versions map[string]indexV2Version `json:"versions"`
}

type indexV2Version struct {
	Added    int                    `json:"added"`
	File     map[string]interface{} `json:"file"`
	Manifest map[string]interface{} `json:"manifest"`
}

type entry struct {
	Timestamp int                    `json:"timestamp"`
	Version   int                    `json:"version"`
	Index     map[string]interface{} `json:"index"`
	Diffs     map[string]interface{} `json:"diffs"`
}

func readEntry(path string) (e *entry, err error) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(&e)

	return
}

func readIndexV2(path string) (index *indexV2, err error) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(&index)

	return
}

func SyncV2(repoDir string) (err error) {
	fdroidIndexFilePath := filepath.Join(repoDir, "index-v1.json")
	path := filepath.Join(repoDir, "index-v2.json")
	entryPath := filepath.Join(repoDir, "entry.json")

	r, err := ReadIndex(fdroidIndexFilePath)

	if err != nil {
		return
	}

	var index *indexV2
	var e *entry
	index, err = readIndexV2(path)
	if err != nil {
		return
	}

	e, err = readEntry(entryPath)
	if err != nil {
		return
	}

	for name, packages := range r.Packages {
		for _, p := range packages {
			b, e := os.ReadFile(filepath.Join(
				filepath.Dir(filepath.Dir(path)),
				"metadata",
				name,
				"en-US",
				"changelogs",
				fmt.Sprintf("%d.txt", p.VersionCode)),
			)
			if e != nil {
				continue
			}
			whatsNew := make(map[string]string)
			whatsNew["en-US"] = string(b)
			index.Packages[name].Versions[p.Hash].Manifest["whatsNew"] = whatsNew
		}
	}
	var b []byte
	b, err = json.Marshal(index)
	if err != nil {
		return
	}

	err = os.WriteFile(path, b, 0644)
	if err != nil {
		return
	}

	h := sha256.New()
	h.Write(b)
	sum := h.Sum(nil)
	e.Index["sha256"] = string(sum)

	b, err = json.Marshal(e)
	if err != nil {
		return
	}

	err = os.WriteFile(path, b, 0644)

	return
}

func (r *RepoIndex) FindLatestPackage(pkgName string) (p PackageInfo, ok bool) {
	pkgs, ok := r.Packages[pkgName]
	if !ok {
		return p, false
	}

	sort.Slice(pkgs, func(i, j int) bool {
		if pkgs[i].VersionCode != pkgs[j].VersionCode {
			return pkgs[i].VersionCode < pkgs[j].VersionCode
		}

		v1, err := version.NewVersion(pkgs[i].VersionName)
		if err != nil {
			return true
		}

		v2, err := version.NewVersion(pkgs[i].VersionName)
		if err != nil {
			return false
		}

		return v1.LessThan(v2)
	})

	// Return the one with the latest version
	return pkgs[len(pkgs)-1], true
}

func ReadIndex(path string) (index *RepoIndex, err error) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(&index)

	return
}
