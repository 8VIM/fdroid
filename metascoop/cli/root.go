package cli

import (
	"cmp"
	"context"
	"fmt"
	"io/fs"
	"log"
	"metascoop/apps"
	"metascoop/file"
	"metascoop/git"
	"metascoop/md"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/google/go-github/v61/github"
	"golang.org/x/oauth2"
)

type Globals struct {
	githubClient *github.Client
	appFile      *apps.AppFile
	loader       *apps.AppLoader
	AppFile      string `help:"Path to apps.yaml file" type:"path" short:"a" default:"apps.yaml"`
	RepoDir      string `help:"path to fdroid \"repo\" directory" type:"path" short:"r" default:"fdroid/repo"`
	AccessToken  string `help:"GitHub personal access token" short:"t"`
	Debug        bool   `help:"Debug mode won't run the fdroid command" short:"d" default:"false"`
}

type CLI struct {
	Globals

	Release ReleaseCmd `cmd:"" help:"Get releases"`
	Pr      PrCmd      `cmd:"" help:"Get apk from a PR"`
	Badges  BadgesCmd  `cmd:"" help:"Generate badges"`
}

type BadgesCmd struct{}

func (g *Globals) AfterApply() error {
	appFile, err := apps.ParseAppFile(g.AppFile)
	if err != nil {
		return err
	}
	g.appFile = appFile
	var authenticatedClient *http.Client = nil
	if g.AccessToken != "" {
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: g.AccessToken},
		)
		authenticatedClient = oauth2.NewClient(ctx, ts)
	}

	g.githubClient = github.NewClient(authenticatedClient)
	g.loader = g.appFile.NewLoader(g.githubClient)
	return nil
}

func (c *BadgesCmd) Run(g *Globals) error {
	return apps.GenerateBadges(g.AppFile, g.RepoDir)
}

func (g *Globals) updateAndPull() error {
	if !g.Debug {
		if err := runFdroidUpdate(g.RepoDir); err != nil {
			return err
		}
	}
	fdroidIndexFilePath := filepath.Join(g.RepoDir, "index-v1.json")
	fdroidIndex, err := apps.ReadIndex(fdroidIndexFilePath)
	if err != nil {
		return fmt.Errorf("reading f-droid repo index: %s\n::endgroup::\n", err.Error())
	}
	apkInfoMap := g.appFile.Apks()
	var toRemovePaths []string

	walkPath := filepath.Join(filepath.Dir(g.RepoDir), "metadata")
	err = filepath.WalkDir(walkPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".yml") {
			return err
		}

		pkgname := strings.TrimSuffix(filepath.Base(path), ".yml")

		fmt.Printf("::group::%s\n", pkgname)

		return func() error {
			defer fmt.Println("::endgroup::")
			log.Printf("Working on %q", pkgname)

			meta, err := apps.ReadMetaFile(path)
			if err != nil {
				log.Printf("Reading meta file %q: %s", path, err.Error())
				return nil
			}

			latestPackage, ok := fdroidIndex.FindLatestPackage(pkgname)
			if !ok {
				return nil
			}

			log.Printf("The latest version is %q with versionCode %d", latestPackage.VersionName, latestPackage.VersionCode)

			apkInfo, ok := apkInfoMap[latestPackage.ApkName]
			if !ok {
				log.Printf("Cannot find apk info for %q", latestPackage.ApkName)
				return nil
			}

			// Now update with some info

			setNonEmpty(meta, "AuthorName", apkInfo.Author())
			fn := apkInfo.FriendlyName
			if fn == "" {
				fn = apkInfo.Name()
			}
			setNonEmpty(meta, "Name", fn)
			setNonEmpty(meta, "SourceCode", apkInfo.GitURL)
			setNonEmpty(meta, "License", apkInfo.License)
			setNonEmpty(meta, "WebSite", apkInfo.Website)
			setNonEmpty(meta, "IssueTracker", apkInfo.IssueTracker)
			setNonEmpty(meta, "Description", apkInfo.Description)

			var summary = apkInfo.Summary
			// See https://f-droid.org/en/docs/Build_Metadata_Reference/#Summary for max length
			const maxSummaryLength = 80
			if len(summary) > maxSummaryLength {
				summary = summary[:maxSummaryLength-3] + "..."

				log.Printf("Truncated summary to length of %d (max length)", len(summary))
			}

			setNonEmpty(meta, "Summary", summary)

			if len(apkInfo.Categories) != 0 {
				meta["Categories"] = apkInfo.Categories
			}

			if len(apkInfo.AntiFeatures) != 0 {
				meta["AntiFeatures"] = strings.Join(apkInfo.AntiFeatures, ",")
			}

			meta["CurrentVersion"] = latestPackage.VersionName
			meta["CurrentVersionCode"] = latestPackage.VersionCode
			builds := make([]map[string]interface{}, 0)
			for _, p := range fdroidIndex.Packages[pkgname] {
				build := make(map[string]interface{})
				build["versionCode"] = p.VersionCode
				build["versionName"] = p.VersionName
				builds = append(builds, build)
			}

			sortBuilds(builds)

			meta["Builds"] = builds
			log.Printf("Set current version info to versionName=%q, versionCode=%d", latestPackage.VersionName, latestPackage.VersionCode)

			err = apps.WriteMetaFile(path, meta)
			if err != nil {
				log.Printf("Writing meta file %q: %s", path, err.Error())
				return nil
			}

			log.Printf("Updated metadata file %q", path)

			if apkInfo.ReleaseDescription != "" {
				destFilePath := filepath.Join(walkPath, latestPackage.PackageName, "en-US", "changelogs", fmt.Sprintf("%d.txt", latestPackage.VersionCode))

				err = os.MkdirAll(filepath.Dir(destFilePath), os.ModePerm)
				if err != nil {
					log.Printf("Creating directory for changelog file %q: %s", destFilePath, err.Error())
					return nil
				}

				err = os.WriteFile(destFilePath, []byte(apkInfo.ReleaseDescription), os.ModePerm)
				if err != nil {
					log.Printf("Writing changelog file %q: %s", destFilePath, err.Error())
					return nil
				}

				log.Printf("Wrote release notes to %q", destFilePath)
			}

			log.Printf("Cloning git repository to search for screenshots")

			gitRepoPath, err := git.CloneRepo(apkInfo.GitURL)
			if err != nil {
				log.Printf("Cloning git repo from %q: %s", apkInfo.GitURL, err.Error())
				return nil
			}
			defer os.RemoveAll(gitRepoPath)

			metadata, err := apps.FindMetadata(gitRepoPath)
			if err != nil {
				log.Printf("finding metadata in git repo %q: %s", gitRepoPath, err.Error())
				return nil
			}

			metaIconPath := filepath.Join(gitRepoPath, "metadata", "en-US", "images", "icon.png")
			iconPath := filepath.Join(walkPath, latestPackage.PackageName, "en-US", "icon.png")
			err = file.Move(metaIconPath, iconPath)

			if err != nil {
				log.Printf("Copying icon file %q to %q: %s", metaIconPath, iconPath, err.Error())
			}

			log.Printf("Wrote icon to %s", iconPath)
			toRemovePaths = append(toRemovePaths, iconPath)

			log.Printf("Found %d screenshots", len(metadata.Screenshots))

			screenshotsPath := filepath.Join(walkPath, latestPackage.PackageName, "en-US", "phoneScreenshots")

			_ = os.RemoveAll(screenshotsPath)

			var sccounter int = 1
			for _, sc := range metadata.Screenshots {
				var ext = filepath.Ext(sc)
				if ext == "" {
					log.Printf("Invalid: screenshot file extension is empty for %q", sc)
					continue
				}

				var newFilePath = filepath.Join(screenshotsPath, fmt.Sprintf("%d%s", sccounter, ext))

				err = os.MkdirAll(filepath.Dir(newFilePath), os.ModePerm)
				if err != nil {
					log.Printf("Creating directory for screenshot file %q: %s", newFilePath, err.Error())
					return nil
				}

				err = file.Move(sc, newFilePath)
				if err != nil {
					log.Printf("Moving screenshot file %q to %q: %s", sc, newFilePath, err.Error())
					return nil
				}

				log.Printf("Wrote screenshot to %s", newFilePath)

				sccounter++
			}

			toRemovePaths = append(toRemovePaths, screenshotsPath)

			return nil
		}()
	})
	if err != nil {
		return err
	}

	if err := runFdroidUpdate(g.RepoDir); err != nil {
		return err
	}
	for _, rmpath := range toRemovePaths {
		err = os.RemoveAll(rmpath)
		if err != nil {
			log.Fatalf("removing path %q: %s\n", rmpath, err.Error())
		}
	}

	if err := apps.GenerateBadges(g.AppFile, g.RepoDir); err != nil {
		return err
	}
	if err := md.RegenerateReadme(g.RepoDir); err != nil {
		return err
	}
	return nil
}

func runFdroidUpdate(repoDir string) error {
	cmd := exec.Command("fdroid", "update", "--pretty", "--create-metadata", "--delete-unknown", "--use-date-from-apk")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Dir = filepath.Dir(repoDir)

	log.Printf("Running %q in %s", cmd.String(), cmd.Dir)
	return cmd.Run()
}

func sortBuilds(builds []map[string]interface{}) {
	slices.SortFunc(builds, func(a, b map[string]interface{}) int {
		return cmp.Compare(a["versionCode"].(int), b["versionCode"].(int))
	})
}

func setNonEmpty(m map[string]interface{}, key string, value string) {
	if value != "" || m[key] == "Unknown" {
		m[key] = value

		log.Printf("Set %s to %q", key, value)
	}
}
func Run() {
	cli := CLI{}
	ctx := kong.Parse(&cli,
		kong.Name("metascoop"),
		kong.Description("A self-sufficient runtime for containers"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}))
	err := ctx.Run(&cli.Globals)
	ctx.FatalIfErrorf(err)
}
