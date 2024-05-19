package apps

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"metascoop/git"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/google/go-github/v61/github"
	"golang.org/x/mod/semver"
)

type AppLoader struct {
	apps         *AppFile
	githubClient *github.Client
}

func (a *AppFile) NewLoader(githubClient *github.Client) *AppLoader {
	return &AppLoader{githubClient: githubClient, apps: a}
}

func (l *AppLoader) All(repoDir string) (err error) {
	apkInfoMap := make(map[string]*AppInfo)
	for _, app := range l.apps.Apps {
		fmt.Printf("App: %s/%s\n", app.Author(), app.Name())
		var repo Repo

		repo, err = RepoInfo(app.GitURL)
		if err != nil {
			err = fmt.Errorf("error while getting repo info from URL %q: %s", app.GitURL, err.Error())
			return
		}

		log.Printf("Looking up %s/%s on GitHub", repo.Author, repo.Name)
		var gitHubRepo *github.Repository
		gitHubRepo, _, err = l.githubClient.Repositories.Get(context.Background(), repo.Author, repo.Name)

		if err != nil {
			log.Printf("Error while looking up repo: %s", err.Error())
		} else {
			app.Summary = gitHubRepo.GetDescription()

			if gitHubRepo.License != nil && gitHubRepo.License.SPDXID != nil {
				app.License = *gitHubRepo.License.SPDXID
			}
		}
		log.Printf("Data from GitHub: summary=%q, license=%q", app.Summary, app.License)
		var releases []*github.RepositoryRelease
		releases, err = ListAllReleases(l.githubClient, repo.Author, repo.Name)
		slices.SortFunc(releases, func(a *github.RepositoryRelease, b *github.RepositoryRelease) int {
			return semver.Compare(a.GetTagName(), b.GetTagName())
		})
		if err != nil {
			err = fmt.Errorf("error while listing repo releases for %q: %s", app.GitURL, err.Error())
			return
		}
		log.Printf("Received %d releases", len(releases))

		for _, release := range releases {
			fmt.Printf("::group::Release %s\n", release.GetTagName())
			func() {
				defer fmt.Println("::endgroup::")

				if !release.GetPrerelease() {
					log.Printf("Skipping non prerelease %q", release.GetTagName())
					return
				}

				if !semver.IsValid(release.GetTagName()) {
					log.Printf("%q is not a semver", release.GetTagName())
					return
				}
				log.Printf("Working on release with tag name %q", release.GetTagName())

				apk := FindAPKRelease(release)
				if apk == nil {
					log.Printf("Couldn't find a release asset with extension \".apk\"")
					return
				}
				var appName string
				appName, err = app.Download(l.githubClient, release, repo, repoDir)
				if err != nil {
					return
				}
				apkInfoMap[appName] = app
			}()
		}
		l.apps.Apps = apkInfoMap
	}
	return
}

func (app *AppInfo) Download(githubClient *github.Client, release *github.RepositoryRelease, repo Repo, repoDir string) (appName string, err error) {
	apk := FindAPKRelease(release)
	if apk == nil {
		err = fmt.Errorf("Couldn't find a release asset with extension \".apk\"")
		return
	}
	appName = GenerateReleaseFilename(app.Name(), release.GetTagName())
	app.ReleaseDescription = release.GetBody()
	if app.ReleaseDescription != "" {
		log.Printf("Release notes: %s", app.ReleaseDescription)
	}
	log.Printf("Target APK name: %s", appName)
	appTargetPath := filepath.Join(repoDir, appName)
	_, err = os.Stat(appTargetPath)
	// If the app file already exists for this version, we continue
	if !errors.Is(err, os.ErrNotExist) {
		log.Printf("Already have APK for version %q at %q", release.GetTagName(), appTargetPath)
		err = nil
		return
	}
	err = nil
	dlCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var appStream io.ReadCloser
	appStream, _, err = githubClient.Repositories.DownloadReleaseAsset(dlCtx, repo.Author, repo.Name, apk.GetID(), http.DefaultClient)
	if err != nil {
		err = fmt.Errorf("error while downloading app %q (artifact id %d) from from release %q: %s", app.GitURL, apk.GetID(), release.GetTagName(), err.Error())
		return
	}

	err = downloadStream(appTargetPath, appStream)
	if err != nil {
		err = fmt.Errorf("error while downloading app %q (artifact id %d) from from release %q to %q: %s", app.GitURL, *apk.ID, *release.TagName, appTargetPath, err.Error())
		return
	}

	log.Printf("Successfully downloaded app for version %q", release.GetTagName())
	return
}

func (l *AppLoader) FromRelease(repoDir string, appKey string, version string) (err error) {
	app, ok := l.apps.Apps[appKey]
	if !ok {
		err = fmt.Errorf("unknown app: %s", appKey)
		return
	}
	var repo Repo
	repo, err = RepoInfo(app.GitURL)
	if err != nil {
		err = fmt.Errorf("error while getting repo info from URL %q: %s", app.GitURL, err.Error())
		return
	}
	log.Printf("Looking up %s/%s on GitHub", repo.Author, repo.Name)
	var gitHubRepo *github.Repository
	gitHubRepo, _, err = l.githubClient.Repositories.Get(context.Background(), repo.Author, repo.Name)

	if err != nil {
		log.Printf("Error while looking up repo: %s", err.Error())
	} else {
		app.Summary = gitHubRepo.GetDescription()

		if gitHubRepo.License != nil && gitHubRepo.License.SPDXID != nil {
			app.License = *gitHubRepo.License.SPDXID
		}
	}
	dlCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	var release *github.RepositoryRelease
	release, _, err = l.githubClient.Repositories.GetReleaseByTag(dlCtx, repo.Author, repo.Name, version)
	if err != nil {
		return
	}
	var appName string
	apkInfoMap := make(map[string]*AppInfo)
	appName, err = app.Download(l.githubClient, release, repo, repoDir)
	if _, ok := apkInfoMap[appName]; !ok {
		apkInfoMap[appName] = app
	}
	l.apps.Apps = apkInfoMap
	return

}
func (l *AppLoader) FromPR(repoDir string, appKey string, prNumber int, artifact int, sha string) (appName string, err error) {
	app, ok := l.apps.Apps[appKey]
	if !ok {
		err = fmt.Errorf("unknown app: %s", appKey)
		return
	}
	var repo Repo
	repo, err = RepoInfo(app.GitURL)
	if err != nil {
		err = fmt.Errorf("error while getting repo info from URL %q: %s", app.GitURL, err.Error())
		return
	}
	log.Printf("Looking up %s/%s on GitHub", repo.Author, repo.Name)
	apkInfoMap := make(map[string]*AppInfo)

	appName = fmt.Sprintf("%s_pr_%d_%s.apk", app.Name(), prNumber, sha)
	appTargetPath := filepath.Join(repoDir, appName)

	_, err = os.Stat(appTargetPath)
	// If the app file already exists for this version, we continue
	if !errors.Is(err, os.ErrNotExist) {
		log.Printf("Already have APK for version %q at %q", appName, appTargetPath)
	} else {
		err = l.downloadArtifact(appTargetPath, repo.Author, repo.Name, artifact)
		if err != nil {
			return
		}
	}

	var str string
	str, err = git.GetPrCommit(app.GitURL, prNumber, sha)
	if err != nil {
		log.Printf("Error cloning %s for %d on %s: %s\n", app.GitURL, prNumber, sha, err.Error())
		return
	}
	dlCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	var pr *github.PullRequest
	pr, _, err = l.githubClient.PullRequests.Get(dlCtx, repo.Author, repo.Name, prNumber)
	if err != nil {
		return
	}
	app.Summary = fmt.Sprintf(`PR #%d
%s`, prNumber, pr.GetBody())
	app.FriendlyName = fmt.Sprintf("%s PR: %d", app.FriendlyName, prNumber)
	app.ReleaseDescription = fmt.Sprintf(`Commit (%s): %s`, sha, str)
	apkInfoMap[appName] = app
	l.apps.Apps = apkInfoMap

	return
}

func (l *AppLoader) downloadArtifact(appTargetPath, author, name string, artifact int) (err error) {
	dlCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	var u *url.URL
	u, _, err = l.githubClient.Actions.DownloadArtifact(dlCtx, author, name, int64(artifact), 1)
	if err != nil {
		return
	}

	var req *http.Request
	var resp *http.Response
	if req, err = http.NewRequestWithContext(context.Background(),
		http.MethodGet,
		u.String(),
		http.NoBody); err != nil {
		return
	}
	if resp, err = http.DefaultClient.Do(req); err != nil {
		return
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("bad status: %s", resp.Status)
		return
	}
	var body []byte
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	var zipReader *zip.Reader
	zipReader, err = zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return
	}
	for _, zipFile := range zipReader.File {
		if strings.HasSuffix(zipFile.Name, ".apk") {
			var rc io.ReadCloser
			rc, err = zipFile.Open()
			if err != nil {
				return
			}
			downloadStream(appTargetPath, rc)
			break
		}
	}
	return
}

func downloadStream(targetFile string, rc io.ReadCloser) (err error) {
	defer rc.Close()

	targetTemp := targetFile + ".tmp"

	f, err := os.Create(targetTemp)
	if err != nil {
		return
	}

	_, err = io.Copy(f, rc)
	if err != nil {
		_ = f.Close()
		_ = os.Remove(targetTemp)

		return
	}

	err = f.Close()
	if err != nil {
		return
	}

	err = os.Rename(targetTemp, targetFile)
	return
}
