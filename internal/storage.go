package internal

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"
)

type Stat struct {
	Author     string              `json:"author"`
	LinesCount int                 `json:"lines_count"`
	Files      map[string]struct{} `json:"files"`
	Commits    map[string]struct{} `json:"commits"`
}
type Storage struct {
	m           *sync.Mutex
	Lst         []*Stat
	AuthorToArr map[string]int
}

func NewStorage() *Storage {
	return &Storage{
		m:           &sync.Mutex{},
		AuthorToArr: make(map[string]int),
		Lst:         make([]*Stat, 0),
	}
}

func (s *Storage) Add(commitAuthor map[string]string, CommitLines map[string]int, filename string) {
	s.m.Lock()
	defer s.m.Unlock()
	for commit, author := range commitAuthor {
		if _, ok := s.AuthorToArr[author]; !ok {
			s.AuthorToArr[author] = len(s.Lst)
			s.Lst = append(s.Lst, &Stat{
				Author:     author,
				LinesCount: 0,
				Files:      make(map[string]struct{}),
				Commits:    make(map[string]struct{}),
			})
		}
		s.Lst[s.AuthorToArr[author]].Commits[commit] = struct{}{}
		s.Lst[s.AuthorToArr[author]].LinesCount += CommitLines[commit]
		s.Lst[s.AuthorToArr[author]].Files[filename] = struct{}{}
	}
}

func (s *Storage) AddEmptyFile(author, filename string) {
	s.m.Lock()
	defer s.m.Unlock()
	if _, ok := s.AuthorToArr[author]; !ok {
		s.AuthorToArr[author] = len(s.Lst)
		s.Lst = append(s.Lst, &Stat{
			Author:     author,
			LinesCount: 0,
			Files:      make(map[string]struct{}),
		})
	}
	s.Lst[s.AuthorToArr[author]].Files[filename] = struct{}{}
}

func (s *Storage) PrintStats(format, orderBy string) {
	sort.SliceStable(s.Lst, func(i, j int) bool {
		primaryI, primaryJ := s.Lst[i].LinesCount, s.Lst[j].LinesCount
		secondaryI, secondaryJ := len(s.Lst[i].Commits), len(s.Lst[j].Commits)
		tertiaryI, tertiaryJ := len(s.Lst[i].Files), len(s.Lst[j].Files)

		switch orderBy {
		case "commits":
			primaryI, primaryJ = secondaryI, secondaryJ
			secondaryI, secondaryJ = s.Lst[i].LinesCount, s.Lst[j].LinesCount
		case "files":
			primaryI, primaryJ = tertiaryI, tertiaryJ
			secondaryI, secondaryJ = s.Lst[i].LinesCount, s.Lst[j].LinesCount
			tertiaryI, tertiaryJ = len(s.Lst[i].Commits), len(s.Lst[j].Commits)
		}

		if primaryI != primaryJ {
			return primaryI > primaryJ
		}
		if secondaryI != secondaryJ {
			return secondaryI > secondaryJ
		}
		if tertiaryI != tertiaryJ {
			return tertiaryI > tertiaryJ
		}
		return s.Lst[i].Author < s.Lst[j].Author
	})

	switch format {
	case "tabular":
		maxNameLen := len("Name")
		maxLinesLen, maxCommitsLen, maxFilesLen := len("Lines"), len("Commits"), len("Files")
		for _, stat := range s.Lst {
			if len(stat.Author) > maxNameLen {
				maxNameLen = len(stat.Author)
			}
			if l := len(fmt.Sprintf("%d", stat.LinesCount)); l > maxLinesLen {
				maxLinesLen = l
			}
			if l := len(fmt.Sprintf("%d", len(stat.Commits))); l > maxCommitsLen {
				maxCommitsLen = l
			}
			if l := len(fmt.Sprintf("%d", len(stat.Files))); l > maxFilesLen {
				maxFilesLen = l
			}
		}
		fmt.Printf("%-*s %-*s %-*s %-*s\n", maxNameLen, "Name", maxLinesLen, "Lines", maxCommitsLen, "Commits", maxFilesLen, "Files")
		for _, stat := range s.Lst {
			fmt.Printf("%-*s %-*d %-*d %d\n", maxNameLen, stat.Author, maxLinesLen, stat.LinesCount, maxCommitsLen, len(stat.Commits), len(stat.Files))
		}
	case "csv":
		writer := csv.NewWriter(os.Stdout)
		_ = writer.Write([]string{"Name", "Lines", "Commits", "Files"})
		for _, stat := range s.Lst {
			_ = writer.Write([]string{stat.Author, fmt.Sprintf("%d", stat.LinesCount), fmt.Sprintf("%d", len(stat.Commits)), fmt.Sprintf("%d", len(stat.Files))})
		}
		writer.Flush()
	case "json":
		output, _ := json.MarshalIndent(s.formatJSON(), "", "  ")
		fmt.Println(string(output))
	case "json-lines":
		for _, stat := range s.formatJSON() {
			output, _ := json.Marshal(stat)
			fmt.Println(string(output))
		}
	default:
		fmt.Println("Unknown format")
	}
}

func (s *Storage) formatJSON() []map[string]interface{} {
	var result []map[string]interface{}
	for _, stat := range s.Lst {
		result = append(result, map[string]interface{}{
			"name":    stat.Author,
			"lines":   stat.LinesCount,
			"commits": len(stat.Commits),
			"files":   len(stat.Files),
		})
	}
	return result
}
