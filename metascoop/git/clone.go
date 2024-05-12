package git

import (
	"fmt"
	"os"
	"os/exec"
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
	cmd = exec.Command("git", "log", "-1", "--no-merges", "--pretty=%B")
	cmd.Dir = dirPath
	var b []byte
	b, err = cmd.Output()
	if err != nil {
		return
	}
	commit = string(b)
	return
}
