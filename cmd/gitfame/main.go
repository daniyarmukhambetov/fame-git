package main

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

//go:embed language_extensions.json
var langFile []byte
var extToLang = map[string]string{}

func main() {
	pwd, _ := os.Getwd()
	repo := flag.String("repository", pwd+"/yamlembed", "Путь до Git репозитория")
	rev := flag.String("revision", "HEAD", "Указатель на коммит")
	orderBy := flag.String("order-by", "lines", "Ключ сортировки: lines, commits, files")
	useCommitter := flag.Bool("use-committer", false, "Использовать коммиттера вместо автора")
	format := flag.String("format", "tabular", "Формат вывода: tabular, csv, json, json-lines")
	extensions := flag.String("extensions", "", "Список расширений через запятую, например '.go,.md'")
	languages := flag.String("languages", "", "Список языков через запятую, например 'go,markdown'")
	exclude := flag.String("exclude", "", "Набор Glob паттернов, исключающих файлы")
	restrictTo := flag.String("restrict-to", "", "Набор Glob паттернов, включающих только определённые файлы")
	setUpExtToLang()
	flag.Parse()
	extensionsList := make([]string, 0)
	languagesList := make([]string, 0)
	excludePatterns := make([]string, 0)
	restrictToList := make([]string, 0)

	if *extensions != "" {
		extensionsList = strings.Split(*extensions, ",")
	}
	if *languages != "" {
		languagesList = strings.Split(*languages, ",")
	}
	if *exclude != "" {
		excludePatterns = strings.Split(*exclude, ",")
	}
	if *restrictTo != "" {
		restrictToList = strings.Split(*restrictTo, ",")
	}
	if *orderBy != "lines" && *orderBy != "commits" && *orderBy != "files" {
		os.Exit(1)
	}
	if *format != "tabular" && *format != "json" && *format != "json-lines" && *format != "csv" {
		os.Exit(1)
	}
	storage := internal.NewStorage()
	wg := &sync.WaitGroup{}
	internal.TraverseRepo(*repo, *rev, wg, func(revision, path, filename string) {
		defer wg.Done()
		ext := filepath.Ext(filename)
		if !checkExtensions(extensionsList, ext) {
			return
		}
		if !checkLanguages(filename, languagesList) {
			return
		}
		if !checkFileExcluded(filename, excludePatterns) {
			return
		}
		if !checkFileRestricted(filename, restrictToList) {
			return
		}
		cmd := exec.Command("git", "blame", "--porcelain", revision, "--", filename)
		cmd.Dir = path
		pipe, _ := cmd.StdoutPipe()
		if err := cmd.Start(); err != nil {
			panic(err)
		}
		reader := bufio.NewReader(pipe)
		statistic := internal.NewStatistics(*useCommitter)
		statistic.Count(reader)
		_ = cmd.Wait()
		if len(statistic.CommitAuthor) == 0 {
			cmd1 := exec.Command("git", "log", "-1", revision, "--", filename)
			cmd1.Dir = path
			output, err := cmd1.Output()
			if err != nil {
				panic(err)
			}
			stringReader := strings.NewReader(string(output))
			scanner := bufio.NewScanner(stringReader)
			commit := ""
			author := ""
			for scanner.Scan() {
				line := scanner.Text()
				line = strings.TrimSuffix(line, "\n")
				split := strings.Split(line, " ")
				if split[0] == "commit" {
					commit = split[1]
				}
				if split[0] == "Author:" {
					author = strings.Join(split[1:3], " ")
					if nameSplit := strings.Split(split[1], "\t"); len(nameSplit) >= 2 {
						author = split[1]
					}
				}
			}
			statistic.CommitAuthor[commit] = author
		}
		storage.Add(statistic.CommitAuthor, statistic.CommitLines, statistic.Filename)
	})
	wg.Wait()
	storage.PrintStats(*format, *orderBy)
	//config := Config{
	//	Repository:   *repo,
	//	Revision:     *rev,
	//	OrderBy:      *orderBy,
	//	UseCommitter: *useCommitter,
	//	Format:       *format,
	//	Extensions:   parseList(*extensions),
	//	Languages:    parseList(*languages),
	//	Exclude:      parseList(*exclude),
	//	RestrictTo:   parseList(*restrictTo),
	//}
}

func checkExtensions(extensions []string, ext string) bool {
	if len(extensions) == 0 {
		return true
	}
	for _, e := range extensions {
		if e == ext {
			return true
		}
	}
	return false
}

type Language struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Extensions []string `json:"extensions"`
}

func setUpExtToLang() {
	var languageConfigs []Language
	if err := json.Unmarshal(langFile, &languageConfigs); err != nil {
		return
	}
	for _, config := range languageConfigs {
		for _, ext := range config.Extensions {
			extToLang[strings.ToLower(ext)] = strings.ToLower(config.Name)
		}
	}
}

func checkLanguages(filename string, languages []string) bool {
	if len(languages) == 0 {
		return true
	}

	ext := filepath.Ext(filename)
	name, ok := extToLang[ext]
	if !ok {
		return false
	}
	res := contains(languages, name)
	return res
}

func contains(slice []string, item string) bool {
	for _, v := range slice {
		if strings.EqualFold(v, item) {
			return true
		}
	}
	return false
}

func checkFileExcluded(filename string, patterns []string) bool {
	if len(patterns) == 0 {
		return true
	}
	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, filename)
		if err == nil && matched {
			return false
		}
	}
	return true
}

func checkFileRestricted(filename string, restrictToList []string) bool {
	if len(restrictToList) == 0 {
		return true
	}
	for _, pattern := range restrictToList {
		matched, err := filepath.Match(pattern, filename)
		if err == nil && matched {
			return true
		}
	}
	return false
}
