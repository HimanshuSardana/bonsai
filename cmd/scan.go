package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

var funcMap = template.FuncMap{"toJSON": toJSON}

type Repo struct {
	Name        string
	Path        string
	CloneURL    string
	Readme      string
	Files       []string
	Commits     []Commit
	LastMessage string
	LastTime    int64
}

type SiteIndex struct {
	Header      string
	Description string
	Repos       []Repo
}

type Commit struct {
	Hash      string `json:"hash"`
	Message   string `json:"message"`
	Diff      string `json:"diff"`
	Timestamp int64  `json:"timestamp"`
}

func Scan(root string) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving path: %v\n", err)
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting working directory: %v\n", err)
		os.Exit(1)
	}

	header, description, cloneBase := parseBonsaiConfig(cwd)

	templateDir := findTemplateDir()

	var repos []Repo

	filepath.Walk(absRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() || info.Name() != ".git" {
			return nil
		}

		repoPath := filepath.Dir(path)

		if !isGitRepo(repoPath) {
			return nil
		}

		repo := Repo{
			Name: filepath.Base(repoPath),
			Path: repoPath,
		}

		repo.CloneURL = repo.Path
		if cloneBase != "" {
			repo.CloneURL = cloneBase + "/" + repo.Name
		}

		repo.Readme = getReadme(repoPath)
		repo.Commits = getCommits(repoPath)
		repo.Files = getFiles(repoPath)
		repo.LastMessage, repo.LastTime = getLastCommitInfo(repoPath)

		repos = append(repos, repo)
		return filepath.SkipDir
	})

	outDir := filepath.Join(absRoot, ".bonsai")
	os.MkdirAll(outDir, 0755)

	detailTmpl := template.Must(template.New("detail.html").Funcs(funcMap).ParseFiles(filepath.Join(templateDir, "detail.html")))
	mainTmpl := template.Must(template.New("main.html").Funcs(funcMap).ParseFiles(filepath.Join(templateDir, "main.html")))

	for _, repo := range repos {
		repoOutDir := filepath.Join(outDir, repo.Name)
		os.MkdirAll(repoOutDir, 0755)

		indexPath := filepath.Join(repoOutDir, "index.html")
		f, err := os.Create(indexPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating %s: %v\n", indexPath, err)
			continue
		}

		detailTmpl.Execute(f, repo)
		f.Close()
	}

	mainIndexPath := filepath.Join(outDir, "index.html")
	f, err := os.Create(mainIndexPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating index: %v\n", err)
		os.Exit(1)
	}
	mainTmpl.Execute(f, SiteIndex{Header: header, Description: description, Repos: repos})
	f.Close()

	if len(repos) == 0 {
		fmt.Println("No git repositories found")
	} else {
		fmt.Printf("Scanned %d repositories\n", len(repos))
	}
}

func findTemplateDir() string {
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Join(filepath.Dir(exe), "cmd", "templates")
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	cwd, err := os.Getwd()
	if err == nil {
		dir := filepath.Join(cwd, "cmd", "templates")
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	fmt.Fprintf(os.Stderr, "Error: cannot find cmd/templates/ directory\n")
	os.Exit(1)
	return ""
}

func parseBonsaiConfig(root string) (header, description, cloneBase string) {
	header = "Bonsai"
	description = "A lightweight Git frontend"
	data, err := os.ReadFile(filepath.Join(root, "bonsai.yaml"))
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "header:"):
			header = strings.TrimSpace(strings.TrimPrefix(line, "header:"))
		case strings.HasPrefix(line, "description:"):
			description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
		case strings.HasPrefix(line, "clone_url:"):
			cloneBase = strings.TrimSpace(strings.TrimPrefix(line, "clone_url:"))
		}
	}
	return
}

func getLastCommitInfo(path string) (message string, ts int64) {
	cmd := exec.Command("git", "log", "-1", "--format=%s")
	cmd.Dir = path
	out, err := cmd.Output()
	if err == nil {
		message = strings.TrimSpace(string(out))
	}

	cmd = exec.Command("git", "log", "-1", "--format=%at")
	cmd.Dir = path
	out, err = cmd.Output()
	if err == nil {
		fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &ts)
	}
	return
}

func isGitRepo(path string) bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = path
	return cmd.Run() == nil
}

func getReadme(path string) string {
	for _, name := range []string{"README.md", "README.txt", "README"} {
		data, err := os.ReadFile(filepath.Join(path, name))
		if err == nil {
			return string(data)
		}
	}
	return ""
}

func getCommits(path string) []Commit {
	cmd := exec.Command("git", "log", "--format=%H %at %s", "-50")
	cmd.Dir = path
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var commits []Commit
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 3)
		if len(parts) >= 3 {
			var ts int64
			fmt.Sscanf(parts[1], "%d", &ts)
			commits = append(commits, Commit{
				Hash:      parts[0],
				Timestamp: ts,
				Message:   parts[2],
				Diff:      getCommitDiff(path, parts[0]),
			})
		}
	}
	return commits
}

func toJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func getCommitDiff(path, hash string) string {
	cmd := exec.Command("git", "diff", hash+"^.."+hash)
	cmd.Dir = path
	out, err := cmd.Output()
	if err == nil {
		return string(out)
	}
	cmd = exec.Command("git", "show", "--format=", hash)
	cmd.Dir = path
	out, err = cmd.Output()
	if err == nil {
		return string(out)
	}
	return ""
}

func getFiles(path string) []string {
	cmd := exec.Command("git", "ls-tree", "--name-only", "-r", "HEAD")
	cmd.Dir = path
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}
	return lines
}



