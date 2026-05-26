package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

type DiffLine struct {
	Class string
	Text  string
}

type Repo struct {
	Name     string
	Path     string
	Readme   string
	Commits  []Commit
	Files    []string
	DiffLines []DiffLine
}

type Commit struct {
	Hash    string
	Message string
}

func Scan(root string) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving path: %v\n", err)
		os.Exit(1)
	}

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

		repo.Readme = getReadme(repoPath)
		repo.Commits = getCommits(repoPath)
		repo.Files = getFiles(repoPath)
		repo.DiffLines = getDiff(repoPath)

		repos = append(repos, repo)
		return filepath.SkipDir
	})

	outDir := filepath.Join(absRoot, ".bonsai")
	os.MkdirAll(outDir, 0755)

	for _, repo := range repos {
		repoOutDir := filepath.Join(outDir, repo.Name)
		os.MkdirAll(repoOutDir, 0755)

		indexPath := filepath.Join(repoOutDir, "index.html")
		f, err := os.Create(indexPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating %s: %v\n", indexPath, err)
			continue
		}

		tmpl := template.Must(template.New("index").Parse(indexHTML))
		tmpl.Execute(f, repo)
		f.Close()

		fmt.Printf("Scanned %s\n", repo.Name)
	}

	if len(repos) == 0 {
		fmt.Println("No git repositories found")
	} else {
		fmt.Printf("Scanned %d repositories\n", len(repos))
	}
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
	cmd := exec.Command("git", "log", "--oneline", "-50")
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
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			commits = append(commits, Commit{Hash: parts[0], Message: parts[1]})
		}
	}
	return commits
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

func getDiff(path string) []DiffLine {
	cmd := exec.Command("git", "diff", "HEAD~1..HEAD")
	cmd.Dir = path
	out, err := cmd.Output()
	if err != nil {
		return []DiffLine{{Class: "context", Text: "(initial commit — no previous diff)"}}
	}
	lines := strings.Split(string(out), "\n")
	var diffLines []DiffLine
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
			diffLines = append(diffLines, DiffLine{Class: "header", Text: line})
		case strings.HasPrefix(line, "@@"):
			diffLines = append(diffLines, DiffLine{Class: "hunk", Text: line})
		case strings.HasPrefix(line, "+"):
			diffLines = append(diffLines, DiffLine{Class: "add", Text: line})
		case strings.HasPrefix(line, "-"):
			diffLines = append(diffLines, DiffLine{Class: "del", Text: line})
		default:
			diffLines = append(diffLines, DiffLine{Class: "context", Text: line})
		}
	}
	return diffLines
}

const indexHTML = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>{{.Name}}</title>
</head>
<body>
<h1>{{.Name}}</h1>
<p><strong>Path:</strong> {{.Path}}</p>

{{if .Readme}}
<hr>
<h2>README</h2>
<pre>{{.Readme}}</pre>
{{end}}

<hr>
<h2>Commits</h2>
<ul>
{{range .Commits}}
<li><code>{{.Hash}}</code> {{.Message}}</li>
{{end}}
</ul>

<hr>
<h2>Files</h2>
<ul>
{{range .Files}}
<li>{{.}}</li>
{{end}}
</ul>

<hr>
<h2>Latest Diff</h2>
<pre style="padding: 0;">
{{range .DiffLines}}<span style="display:block;padding:0 8px;{{if eq .Class "add"}}background:#e6ffed;{{else if eq .Class "del"}}background:#ffeef0;{{else if eq .Class "hunk"}}background:#f0f8ff;{{else if eq .Class "header"}}color:#888;{{end}}">{{.Text}}</span>
{{end}}</pre>

</body>
</html>`
