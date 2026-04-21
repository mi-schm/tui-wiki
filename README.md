# 🧠 TUI-Wiki

A lightning-fast, minimalist, and offline-first personal wiki for the terminal. Built with **Go** and **Bubble Tea**, powered by **SQLite**.

TUI-Wiki is a lightweight alternative to bloated note-taking apps. It turns your terminal into a powerful second brain, keeping your knowledge organized in a clean, hierarchical tree structure.

## ✨ Key Features

* **Keyboard-Centric:** Navigate, search, and edit without ever touching your mouse.
* **Privacy First:** Everything stays on your machine in a local SQLite database. No cloud, no tracking.
* **Interlinked:** Automatic backlink detection and `[[Wiki-Style]]` linking.
* **Editor Agnostic:** Works with Vim, Emacs, Nano, or Notepad via the `$EDITOR` environment variable.
* **Collapsible Tree View:** Manage deep hierarchies with toggleable nodes.
* **Real-time Suggestions:** Prevents duplicate pages by suggesting similar titles during creation.
* **Full-Text Search:** Blazing fast search across all your notes using SQLite FTS5.
* **Dynamic Layout:** Adjust sidebar width on the fly using simple keybindings.

## 🚀 Installation

1. **Clone the repository:**
   ```bash
   git clone [https://github.com/mi-schm/tui-wiki.git](https://github.com/yourusername/tui-wiki.git)
   cd tui-wiki
   
2. **Initialize dependencies:**
	go mod tidy
	
3. **Build and Run:**
	go build -o tui-wiki main.go
	./tui-wiki

## 🛠 Configuration
	TUI-Wiki respects your system settings. To use a specific editor, set your EDITOR environment variable:

	PowerShell: $env:EDITOR = 'C:\Path\To\your-editor.exe'

	Bash/Zsh: export EDITOR='emacs'
	
## 💡 Development Vibe
	This project was built using an AI-collaborative development approach. It emphasizes clean, iterative code and rapid prototyping. It's a stable "vibe-coded" tool designed for daily productivity. Contributions and pull requests are welcome!
	
## License
	This project is licensed under the MIT License - see the LICENSE file for details.
