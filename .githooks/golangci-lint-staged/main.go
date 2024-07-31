//go:build githooks
// +build githooks

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

var (
	golangciLintArgs    = []string{"run"}
	gitGetStagedCmdArgs = []string{"diff-index", "--ignore-submodules", "--no-color", "--no-ext-diff", "--name-only", "HEAD", "--"}
)

func main() {
	/*
		Problem:
			1) pre-commit tool runs hook and pass staged files to it
			2) golangci-lint takes files, but cannot run some
				linters without info from entire package -> so, lint fails

		Solution:
			get package names from staged files and run golangci-lint on packages
	*/
	fileNames, err := getStagedFileNames()
	exitOnErr(err)

	golangciLintArgs = append(golangciLintArgs, getPkgNames(fileNames)...)
	fmt.Printf("DEBUG %v\n", golangciLintArgs)

	out, err := exec.Command("golangci-lint", golangciLintArgs...).CombinedOutput()
	fmt.Printf("%s\n", out)
	exitOnErr(err)
}

func getStagedFileNames() ([]string, error) {
	out, err := exec.Command("git", gitGetStagedCmdArgs...).CombinedOutput()
	if err != nil {
		fmt.Printf("ERROR %s\n", out)
		return nil, err
	}
	var res []string
	for _, part := range bytes.Split(out, []byte("\n")) {
		if bytes.HasSuffix(part, []byte(".go")) {
			res = append(res, string(part))
		}
	}
	return res, nil
}

func getPkgNames(files []string) []string {
	uniqDirNames := make(map[string]struct{})
	for _, name := range files {
		absName, _ := filepath.Abs(name)
		if _, err := os.Stat(absName); err == nil {
			// do not add deleted files
			uniqDirNames[filepath.Dir(absName)] = struct{}{}
		}
	}
	dirNames := make([]string, 0, len(uniqDirNames))
	for dir := range uniqDirNames {
		dirNames = append(dirNames, dir)
	}
	return dirNames
}

func exitOnErr(err error) {
	if err != nil {
		os.Exit(1)
	}
}
