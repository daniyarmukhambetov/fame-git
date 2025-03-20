package internal

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
)

func TraverseRepo(curDir string, revision string, wg *sync.WaitGroup, blobHandler func(revision, dir, filename string)) {
	cmd := exec.Command("git", "ls-tree", "-r", revision)
	cmd.Dir = curDir
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	reader := bufio.NewReader(bytes.NewBuffer(out))

	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		line = strings.TrimSuffix(line, "\n")
		split := strings.FieldsFunc(line, func(r rune) bool {
			return r == '\t' || r == ' '
		})
		fileOrDirName := strings.Join(split[3:], " ")
		objType := split[1]
		switch objType {
		case "blob":
			wg.Add(1)
			go blobHandler(revision, curDir, fileOrDirName)
		case "tree":
			TraverseRepo(fmt.Sprintf("%s/%s", curDir, fileOrDirName), revision, wg, blobHandler)
		}
	}
}
