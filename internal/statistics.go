package internal

import (
	"bufio"
	"io"
	"strings"
)

// 1 one rev lot of commits
// we traverse every file in this revision and

type Statistics struct {
	CommitAuthor map[string]string
	CommitLines  map[string]int
	Filename     string
	authorLabel  string
}

func NewStatistics(useCommitter bool) *Statistics {
	label := "author"
	if useCommitter {
		label = "committer"
	}
	return &Statistics{
		CommitAuthor: make(map[string]string),
		CommitLines:  make(map[string]int),
		Filename:     "",
		authorLabel:  label,
	}
}

func (stat *Statistics) Count(reader *bufio.Reader) {
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		line = strings.TrimSuffix(line, "\n")
		split := strings.Split(line, " ")
		commitHash := split[0]
		if v := stat.CommitLines[commitHash]; v == 0 {
			stat.CommitLines[commitHash] = 1
			for {
				line_, err_ := reader.ReadString('\n')
				if err_ == io.EOF {
					break
				}
				if err_ != nil {
					panic(err_)
				}
				line_ = strings.TrimSuffix(line_, "\n")
				split_ := strings.Split(line_, " ")
				if split_[0] == stat.authorLabel {
					stat.CommitAuthor[commitHash] = strings.Join(split_[1:], " ")
				}
				if split_[0] == "filename" {
					stat.Filename = split_[1]
					_, _ = reader.ReadString('\n')
					break
				}
			}
			continue
		}
		stat.CommitLines[commitHash]++
		_, _ = reader.ReadString('\n')
	}
}
