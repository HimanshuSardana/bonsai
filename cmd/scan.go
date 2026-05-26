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

type Repo struct {
	Name    string
	Path    string
	Readme  string
	Files   []string
	Commits []Commit
}

type Commit struct {
	Hash    string `json:"hash"`
	Message string `json:"message"`
	Diff    string `json:"diff"`
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

		funcMap := template.FuncMap{"toJSON": toJSON}
		tmpl := template.Must(template.New("index").Funcs(funcMap).Parse(indexHTML))
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
			commits = append(commits, Commit{
				Hash:    parts[0],
				Message: parts[1],
				Diff:    getCommitDiff(path, parts[0]),
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

const indexHTML = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>{{.Name}}</title>
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
html, body { height: 100%; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; }
.layout { display: flex; height: 100vh; }
.sidebar { width: 320px; min-width: 320px; border-right: 1px solid #ddd; overflow-y: auto; background: #fafafa; }
.sidebar-header { padding: 16px; border-bottom: 1px solid #ddd; }
.sidebar-header h1 { font-size: 16px; }
.sidebar-header p { font-size: 12px; color: #666; margin-top: 4px; }
.commit-list { list-style: none; }
.commit-entry { padding: 10px 16px; border-bottom: 1px solid #eee; cursor: pointer; }
.commit-entry:hover { background: #eef; }
.commit-entry.active { background: #dde; }
.commit-entry .hash { font-family: monospace; font-size: 11px; color: #888; }
.commit-entry .msg { font-size: 13px; margin-top: 2px; }
.main { flex: 1; overflow-y: auto; padding: 20px; }
.main h2 { font-size: 14px; color: #333; margin-bottom: 8px; }
.readme { background: #f5f5f5; padding: 12px; border-radius: 4px; font-family: monospace; font-size: 12px; white-space: pre-wrap; margin-bottom: 20px; }
.files { margin-bottom: 20px; }
.files li { font-family: monospace; font-size: 12px; margin: 2px 0; }
.diff-view { font-family: monospace; font-size: 12px; line-height: 1.5; }
.diff-view .d-header { color: #888; }
.diff-view .d-hunk { background: #f0f8ff; padding: 0 8px; }
.diff-view .d-add { background: #e6ffed; padding: 0 8px; }
.diff-view .d-del { background: #ffeef0; padding: 0 8px; }
.diff-view .d-ctx { padding: 0 8px; }
</style>
</head>
<body>
<div class="layout">
  <div class="sidebar">
    <div class="sidebar-header">
      <h1>{{.Name}}</h1>
      <p>{{.Path}}</p>
    </div>
    <ul class="commit-list" id="commit-list">
    </ul>
  </div>
  <div class="main" id="main">
    <div id="readme-area">{{if .Readme}}<div class="readme">{{.Readme}}</div>{{end}}</div>
    <div class="files">
      <h2>Files</h2>
      <ul>{{range .Files}}<li>{{.}}</li>{{end}}</ul>
    </div>
    <h2>Diff</h2>
    <div class="diff-view" id="diff-view"></div>
  </div>
</div>
<script id="commits-data" type="application/json">{{.Commits | toJSON}}</script>
<script>
const commits = JSON.parse(document.getElementById('commits-data').textContent);
const list = document.getElementById('commit-list');
const diffView = document.getElementById('diff-view');

function renderDiff(diff) {
  if (!diff) return '<em>(no diff)</em>';
  return diff.split('\n').map(line => {
    let cls = 'd-ctx';
    if (line.startsWith('+++') || line.startsWith('---')) cls = 'd-header';
    else if (line.startsWith('@@')) cls = 'd-hunk';
    else if (line.startsWith('+')) cls = 'd-add';
    else if (line.startsWith('-')) cls = 'd-del';
    return '<div class="' + cls + '">' + escape(line) + '</div>';
  }).join('');
}

function escape(s) {
  return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');
}

function selectCommit(hash) {
  document.querySelectorAll('.commit-entry').forEach(e => e.classList.remove('active'));
  const el = document.querySelector('[data-hash="' + hash + '"]');
  if (el) el.classList.add('active');
  const commit = commits.find(c => c.hash === hash);
  if (commit) diffView.innerHTML = renderDiff(commit.diff);
}

commits.forEach(c => {
  const div = document.createElement('div');
  div.className = 'commit-entry';
  div.dataset.hash = c.hash;
  div.innerHTML = '<div class="hash">' + c.hash + '</div><div class="msg">' + escape(c.message) + '</div>';
  div.addEventListener('click', function (e) { selectCommit(c.hash); });
  list.appendChild(div);
});

if (commits.length) selectCommit(commits[0].hash);
</script>
</body>
</html>`

