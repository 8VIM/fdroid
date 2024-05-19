package cli

import (
	"fmt"
	"log"
	"metascoop/apps"
	"metascoop/file"
	"metascoop/md"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type PrCmd struct {
	App    string      `help:"app" required:""`
	Number int         `help:"Pr number" required:""`
	Add    PrAddCmd    `cmd:"" help:"Add apk from a PR"`
	Delete PrDeleteCmd `cmd:"" help:"Add apk from a PR"`
}

type PrAddCmd struct {
	ArtifactID int    `arg:"" help:"Artifact id"`
	SHA        string `arg:"" help:"SHA ref"`
}

type PrDeleteCmd struct {
}

func (a *PrAddCmd) Run(g *Globals, c *PrCmd) (err error) {
	var appName string
	appName, err = g.loader.FromPR(g.RepoDir, c.App, c.Number, a.ArtifactID, a.SHA)
	if err != nil {
		return
	}

	path := filepath.Dir(g.RepoDir)
	app := filepath.Join(g.RepoDir, appName)
	out := fmt.Sprintf("%s.apk", app)
	config, err := apps.ParseFdroidConfig(filepath.Join(path, "config.yml"))
	if err != nil {
		return err
	}
	jks := fmt.Sprintf("%s.jks", strings.TrimSuffix(config.Keystore, ".p12"))
	cmd := exec.Command("keytool", "-importkeystore", "-srckeystore", config.Keystore, "-srcstoretype", "pkcs12", "-srckeypass", config.Keypass, "-srcstorepass", config.Keystorepass, "-srcalias", config.Alias, "-destkeystore", jks, "-destkeypass", config.Keypass, "-deststorepass", config.Keystorepass, "-destalias", config.Alias)
	cmd.Dir = path
	_ = cmd.Run()

	tools := os.ExpandEnv(filepath.Join(config.SdkPath, "build-tools", g.appFile.BuildToolsVersion, "apksigner"))
	cmd = exec.Command(tools, "sign", "--ks", filepath.Join(filepath.Dir(g.RepoDir), jks), "--ks-key-alias", config.Alias, "--ks-pass", fmt.Sprintf("pass:%s", config.Keystorepass), "--ks-pass", fmt.Sprintf("pass:%s", config.Keypass), "--out", out, app)
	if err := cmd.Run(); err != nil {
		return err
	}
	_ = file.Move(out, app)
	_ = os.Remove(fmt.Sprintf("%s.idsig", out))
	err = g.updateAndPull()
	return
}

func (d *PrDeleteCmd) Run(g *Globals, c *PrCmd) error {
	prefix := fmt.Sprintf("%s_pr_%d", c.App, c.Number)

	fdroidIndexFilePath := filepath.Join(g.RepoDir, "index-v1.json")
	fdroidIndex, err := apps.ReadIndex(fdroidIndexFilePath)
	if err != nil {
		return err
	}
	for _, packages := range fdroidIndex.Packages {
		if strings.HasPrefix(packages[0].ApkName, c.App) {
			packageName := packages[0].PackageName
			toRemovePaths := make([]string, 0)
			versionCodes := make(map[int]struct{})

			for _, p := range fdroidIndex.Packages[packageName] {
				if strings.HasPrefix(p.ApkName, prefix) {
					file := filepath.Join(filepath.Dir(g.RepoDir), "metadata", p.PackageName, "en-US", "changelogs", fmt.Sprintf("%d.txt", p.VersionCode))
					toRemovePaths = append(toRemovePaths, file)
					toRemovePaths = append(toRemovePaths, filepath.Join(g.RepoDir, p.ApkName))
					versionCodes[p.VersionCode] = struct{}{}
				}
			}

			if len(toRemovePaths) == 0 {
				log.Printf("No files found for %d\n", c.Number)
				continue
			}

			for _, path := range toRemovePaths {
				_ = os.Remove(path)
			}

			if len(fdroidIndex.Packages[packageName]) == len(toRemovePaths)/2 {
				_ = os.RemoveAll(filepath.Join(filepath.Dir(g.RepoDir), "metadata", packageName))
				_ = os.RemoveAll(filepath.Join(g.RepoDir, packageName))
			}

			if err := runFdroidUpdate(g.RepoDir); err != nil {
				return err
			}

			fdroidIndex, _ = apps.ReadIndex(fdroidIndexFilePath)
			if lastest, ok := fdroidIndex.FindLatestPackage(packageName); ok {
				path := filepath.Join(filepath.Dir(g.RepoDir), "metadata", fmt.Sprintf("%s.yml", packageName))

				meta, err := apps.ReadMetaFile(path)
				if err != nil {
					log.Printf("Reading meta file %q: %s", path, err.Error())
					return err
				}

				builds := make([]map[string]interface{}, 0)
				for _, b := range meta["Builds"].([]interface{}) {
					build := b.(map[string]interface{})
					versionCode := build["versionCode"].(int)
					if _, ok := versionCodes[versionCode]; !ok {
						builds = append(builds, build)
					}
				}

				sortBuilds(builds)

				meta["CurrentVersion"] = lastest.VersionName
				meta["CurrentVersionCode"] = lastest.VersionCode
				meta["Builds"] = builds

				err = apps.WriteMetaFile(path, meta)

				if err != nil {
					log.Printf("Writing meta file %q: %s", path, err.Error())
					return err
				}

			}

		}
	}

	if err := runFdroidUpdate(g.RepoDir); err != nil {
		return err
	}
	if err := apps.GenerateBadges(g.AppFile, g.RepoDir); err != nil {
		return err
	}

	return md.RegenerateReadme(g.RepoDir)
}
