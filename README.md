# tui-wiki

A minimalist, terminal-based personal wiki. Built with Go, Bubble Tea, and SQLite.

## Features
- Tree-based navigation (collapsible nodes)
- Full-text search (SQLite FTS5)
- Auto-backlink discovery
- Global `$EDITOR` integration
- Single binary, local SQLite storage

## Requirements
- Go 1.21+
- A terminal with Unicode support
- An editor set in your `$EDITOR` environment variable

## Installation
```bash
go mod tidy
go build -o tui-wiki .

<img width="1483" height="760" alt="image" src="https://github.com/user-attachments/assets/6e60423e-88dc-42bf-8e62-05d0a890c075" />

build with ai
## License
	This project is licensed under the MIT License - see the LICENSE file for details.
