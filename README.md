# tui-wiki

A minimalist, terminal-based personal wiki. Built with Go, Bubble Tea, and SQLite.

## Features
- Tree-based navigation (collapsible nodes)
- Full-text search (SQLite FTS5)
- Auto-backlink discovery
- Global `$EDITOR` integration
- Single binary, local SQLite storage
- Export to .md preserving the full tree structure
- Import local folders to automatically reconstruct the wiki hierarchy

Import local folders to automatically reconstruct the wiki hierarchy

## Requirements
- Go 1.21+
- A terminal with Unicode support
- An editor set in your `$EDITOR` environment variable

## Installation
```bash
go mod tidy
go build -o tui-wiki
```
## Screenshot
<img width="1483" height="760" alt="image" src="https://github.com/user-attachments/assets/6e60423e-88dc-42bf-8e62-05d0a890c075" />

## License
	This project is licensed under the MIT License - see the LICENSE file for details.
