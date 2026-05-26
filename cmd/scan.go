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
	Name        string
	Path        string
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

	header, description := parseBonsaiConfig(root)

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
		repo.LastMessage, repo.LastTime = getLastCommitInfo(repoPath)

		repos = append(repos, repo)
		return filepath.SkipDir
	})

	outDir := filepath.Join(absRoot, ".bonsai")
	os.MkdirAll(outDir, 0755)

	funcMap := template.FuncMap{"toJSON": toJSON}

	for _, repo := range repos {
		repoOutDir := filepath.Join(outDir, repo.Name)
		os.MkdirAll(repoOutDir, 0755)

		indexPath := filepath.Join(repoOutDir, "index.html")
		f, err := os.Create(indexPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating %s: %v\n", indexPath, err)
			continue
		}

		tmpl := template.Must(template.New("index").Funcs(funcMap).Parse(detailHTML))
		tmpl.Execute(f, repo)
		f.Close()
	}

	mainIndexPath := filepath.Join(outDir, "index.html")
	f, err := os.Create(mainIndexPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating index: %v\n", err)
		os.Exit(1)
	}
	tmpl := template.Must(template.New("main").Parse(mainHTML))
	tmpl.Execute(f, SiteIndex{Header: header, Description: description, Repos: repos})
	f.Close()

	if len(repos) == 0 {
		fmt.Println("No git repositories found")
	} else {
		fmt.Printf("Scanned %d repositories\n", len(repos))
	}
}

func parseBonsaiConfig(root string) (header, description string) {
	header = "Bonsai"
	description = "A lightweight Git frontend"
	data, err := os.ReadFile(filepath.Join(root, "bonsai.yaml"))
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "header:") {
			header = strings.TrimSpace(strings.TrimPrefix(line, "header:"))
		} else if strings.HasPrefix(line, "description:") {
			description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
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

const mainHTML = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>{{.Header}}</title>
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #f5f5f5; color: #333; padding: 40px; }
.header { text-align: center; margin-bottom: 40px; }
.header h1 { font-size: 32px; }
.header p { color: #666; margin-top: 6px; font-size: 14px; }
.grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 16px; max-width: 1100px; margin: 0 auto; }
.card { background: #fff; border-radius: 8px; padding: 20px; text-decoration: none; color: inherit; box-shadow: 0 1px 3px rgba(0,0,0,0.1); transition: box-shadow 0.15s; display: flex; flex-direction: column; }
.card:hover { box-shadow: 0 4px 12px rgba(0,0,0,0.15); }
.card h2 { font-size: 16px; margin-bottom: 8px; }
.card .msg { font-size: 13px; color: #555; flex: 1; }
.card .time { font-size: 12px; color: #999; margin-top: 12px; }
</style>
</head>
<body>
<div class="header">
  <h1>{{.Header}}</h1>
  <p>{{.Description}}</p>
</div>
<div class="grid">
{{range .Repos}}
  <a class="card" href="{{.Name}}/">
    <h2>{{.Name}}</h2>
    <div class="msg">{{.LastMessage}}</div>
    <div class="time" data-ts="{{.LastTime}}"></div>
  </a>
{{end}}
</div>
<script>
const rtf = new Intl.RelativeTimeFormat('en', { numeric: 'auto' });
function relTime(unix) {
  if (!unix) return '';
  const delta = Math.floor((Date.now() - unix * 1000) / 1000);
  if (delta < 60) return rtf.format(-delta, 'second');
  if (delta < 3600) return rtf.format(-Math.floor(delta / 60), 'minute');
  if (delta < 86400) return rtf.format(-Math.floor(delta / 3600), 'hour');
  if (delta < 2592000) return rtf.format(-Math.floor(delta / 86400), 'day');
  if (delta < 31536000) return rtf.format(-Math.floor(delta / 2592000), 'month');
  return rtf.format(-Math.floor(delta / 31536000), 'year');
}
document.querySelectorAll('[data-ts]').forEach(el => { el.textContent = relTime(el.dataset.ts); });
</script>
</body>
</html>`

const detailHTML = `<!DOCTYPE html>
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
.commit-entry .time { font-size: 10px; color: #aaa; }
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

const rtf = new Intl.RelativeTimeFormat('en', { numeric: 'auto' });
function relTime(unix) {
  if (!unix) return '';
  const delta = Math.floor((Date.now() - unix * 1000) / 1000);
  if (delta < 60) return rtf.format(-delta, 'second');
  if (delta < 3600) return rtf.format(-Math.floor(delta / 60), 'minute');
  if (delta < 86400) return rtf.format(-Math.floor(delta / 3600), 'hour');
  if (delta < 2592000) return rtf.format(-Math.floor(delta / 86400), 'day');
  if (delta < 31536000) return rtf.format(-Math.floor(delta / 2592000), 'month');
  return rtf.format(-Math.floor(delta / 31536000), 'year');
}

commits.forEach(c => {
  const div = document.createElement('div');
  div.className = 'commit-entry';
  div.dataset.hash = c.hash;
  div.innerHTML = '<div class="hash">' + c.hash + ' <span class="time">' + relTime(c.timestamp) + '</span></div><div class="msg">' + escape(c.message) + '</div>';
  div.addEventListener('click', function (e) { selectCommit(c.hash); });
  list.appendChild(div);
});

if (commits.length) selectCommit(commits[0].hash);
</script>
</body>
</html>`

