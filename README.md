# Bonsai

> Serene Git browsing. Zero visual clutter.

Bonsai is an ultra-minimal, high-contrast Go web server and static site generator (SSG) for your local Git repositories, commits, and diffs.

Designed for keyboard-native development workflows, Bonsai replaces heavy electron layers, complex grids, and distracting visual clutter with a fast, system-aware, and beautifully typeset monospace workspace.

## Installation

Ensure you have Go installed on your machine. Run the following command to compile and install the binary globally:

```bash
go install github.com/himanshusardana/bonsai@latest
```

## Commands

Bonsai operates via three primary commands:

```bash
bonsai init # Initializes a new Bonsai workspace in your current directory. This creates a `bonsai.yaml` configuration file where you can define the title and subtitle
bonsai scan # Scans the current working directory for all Git repositories, parses their history, and statically generates the static pages inside a local `.bonsai` directory.
bonsai serve # Launches a local, high-performance web server to serve the generated static assets over `http://localhost:8000`.
```
