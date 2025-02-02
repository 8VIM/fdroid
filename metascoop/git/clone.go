package git

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

func CloneRepo(gitUrl string) (dirPath string, err error) {
	dirPath, err = os.MkdirTemp("", "git-*")
	if err != nil {
		return
	}

	cloneCmd := exec.Command("git", "clone", gitUrl, dirPath)
	err = cloneCmd.Run()
	if err != nil {
		_ = os.RemoveAll(dirPath)
		return
	}

	log.Printf("Clong %s into %s", gitUrl, dirPath)
	return
}

func GetPrCommit(gitUrl string, prNumber int, sha string) (commit string, err error) {
	var dirPath string
	dirPath, err = CloneRepo(gitUrl)
	defer os.RemoveAll(dirPath)

	if err != nil {
		return
	}

	cmd := exec.Command("git", "pull", "origin", fmt.Sprintf("pull/%d/head", prNumber))
	cmd.Dir = dirPath

	err = cmd.Run()
	if err != nil {
		return
	}
	cmd = exec.Command("git", "checkout", sha)
	cmd.Dir = dirPath

	err = cmd.Run()
	if err != nil {
		return
	}
	cmd = exec.Command("git", "show", "-s", "--format=%s")
	cmd.Dir = dirPath

	var b []byte
	b, err = cmd.Output()
	if err != nil {
		return
	}
	commit = strings.Trim(string(b), "\n")
	return
}
