package main

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// --- Styles ---
var (
	docStyle             = lipgloss.NewStyle().Margin(1, 2)
	helpStyle            = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Margin(1, 0)
	successStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true).Margin(1, 0)
	dateStyle            = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true).MarginBottom(1)
	warnStyle            = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	confirmBoxStyle      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2).BorderForeground(lipgloss.Color("196"))
	
	activeContentStyle   = lipgloss.NewStyle().Border(lipgloss.ThickBorder(), false, false, false, true).BorderForeground(lipgloss.Color("205")).PaddingLeft(1)
	inactiveContentStyle = lipgloss.NewStyle().PaddingLeft(2)
	
	suggestionStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true)
)

type editFinishedMsg struct{ content string }

type item struct {
	id          int
	parentID    int
	title       string
	content     string
	updatedAt   string
	indent      int
	collapsed   bool
	hasChildren bool
}

func (i item) Title() string {
	prefix := strings.Repeat("  ", i.indent)
	indicator := "  "
	if i.hasChildren {
		if i.collapsed { indicator = "▸ " } else { indicator = "▾ " }
	}
	treeSymbol := ""
	if i.indent > 0 { treeSymbol = "└─ " }
	return prefix + indicator + treeSymbol + i.title
}

func (i item) Description() string { return "" }
func (i item) FilterValue() string { return i.title }

type model struct {
	list            list.Model
	viewport        viewport.Model
	db              *sql.DB
	lastUpdate      string
	input           textinput.Model
	searchInput     textinput.Model
	collapsedStates map[int]bool
	suggestions     []string 
	creatingNew     bool
	createRoot      bool
	searching       bool
	deleting        bool
	importing       bool // NEU: Status für den Import-Modus
	focusOnContent  bool 
	width, height   int
	sidebarWidth    int 
	statusMsg       string
	statusTimer     time.Time
}

func (m *model) resizePanes() {
	m.list.SetSize(m.sidebarWidth, m.height-8)
	m.viewport.Width = m.width - m.sidebarWidth - 6
	m.viewport.Height = m.height - 10
	if i, ok := m.list.SelectedItem().(item); ok {
		m.renderPage(i)
	}
}

func (m *model) renderPage(i item) {
	layout := "2006-01-02 15:04:05"
	t, err := time.ParseInLocation(layout, i.updatedAt, time.UTC)
	if err != nil { t, err = time.Parse(time.RFC3339, i.updatedAt) }

	if err == nil && t.Year() > 1000 {
		m.lastUpdate = "Last modified: " + t.Local().Format("Jan 02, 2006, 15:04")
	} else {
		m.lastUpdate = "Last modified: " + i.updatedAt 
	}
	
	var backlinks []string
	query := `SELECT title FROM pages WHERE (LOWER(content) LIKE LOWER(?) OR LOWER(content) LIKE LOWER(?)) AND id != ?`
	rows, _ := m.db.Query(query, "%[["+i.title+"]]%", "%"+i.title+"%", i.id)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var bt string
			rows.Scan(&bt)
			backlinks = append(backlinks, bt)
		}
	}

	re := regexp.MustCompile(`\[\[(.*?)\]\]`)
	displayContent := re.ReplaceAllString(i.content, "[$1](#)")
	if len(backlinks) > 0 {
		displayContent += "\n\n---\n**BACKLINKS:**\n"
		uniqueLinks := make(map[string]bool)
		for _, bl := range backlinks {
			if !uniqueLinks[bl] {
				displayContent += fmt.Sprintf("* [[%s]]\n", bl)
				uniqueLinks[bl] = true
			}
		}
	}
	
	renderer, _ := glamour.NewTermRenderer(glamour.WithAutoStyle(), glamour.WithWordWrap(m.viewport.Width-2))
	out, _ := renderer.Render(displayContent)
	m.viewport.SetContent(out)
}

func (m *model) buildTree(parentID int, indent int) []list.Item {
	var items []list.Item
	query := "SELECT id, parent_id, title, content, updated_at FROM pages WHERE parent_id = ? ORDER BY title ASC"
	rows, err := m.db.Query(query, parentID)
	if err != nil { return nil }
	defer rows.Close()

	for rows.Next() {
		var i item
		if err := rows.Scan(&i.id, &i.parentID, &i.title, &i.content, &i.updatedAt); err == nil {
			i.indent = indent
			var count int
			m.db.QueryRow("SELECT COUNT(*) FROM pages WHERE parent_id = ?", i.id).Scan(&count)
			i.hasChildren = count > 0
			i.collapsed = m.collapsedStates[i.id]
			items = append(items, i)
			if !i.collapsed { items = append(items, m.buildTree(i.id, indent+1)...) }
		}
	}
	return items
}

func (m *model) refreshList() { m.list.SetItems(m.buildTree(0, 0)) }

func initialModel(database *sql.DB) model {
	d := list.NewDefaultDelegate()
	d.ShowDescription = false
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.Foreground(lipgloss.Color("205")).Bold(true)
	ti, si := textinput.New(), textinput.New()
	ti.Placeholder, si.Placeholder = "Title...", "Search..."
	m := model{
		list:            list.New(nil, d, 35, 20),
		viewport:        viewport.New(0, 0),
		db:              database,
		input:           ti,
		searchInput:     si,
		collapsedStates: make(map[int]bool),
		sidebarWidth:    35,
	}
	m.list.Title = "Wiki Tree"
	m.refreshList()
	return m
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.resizePanes()
		return m, nil

	case editFinishedMsg:
		if a, ok := m.list.SelectedItem().(item); ok {
			m.db.Exec("UPDATE pages SET content = ? WHERE id = ?", msg.content, a.id)
			m.refreshList()
		}
		return m, nil

	case tea.KeyMsg:
		s := msg.String()

		if m.importing {
			if s == "enter" && m.input.Value() != "" {
				path := m.input.Value()
				m.startImport(path)
				m.importing = false
				m.refreshList()
				m.statusMsg = "Imported from " + path
				m.statusTimer = time.Now().Add(5 * time.Second)
				m.input.Blur()
				return m, nil
			}
			if s == "esc" {
				m.importing = false
				m.input.Blur()
				return m, nil
			}
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}

		if m.creatingNew {
			if s == "enter" && m.input.Value() != "" {
				newT, pID := m.input.Value(), 0
				if !m.createRoot { if a, ok := m.list.SelectedItem().(item); ok { pID = a.id } }
				m.db.Exec("INSERT INTO pages (title, content, parent_id) VALUES (?, ?, ?)", newT, "# "+newT, pID)
				m.creatingNew = false
				m.refreshList()
				return m, nil
			}
			if s == "esc" { m.creatingNew = false; return m, nil }
			m.input, cmd = m.input.Update(msg)
			m.suggestions = nil
			if m.input.Value() != "" {
				rows, _ := m.db.Query("SELECT title FROM pages WHERE title LIKE ? LIMIT 5", m.input.Value()+"%")
				if rows != nil {
					for rows.Next() {
						var t string
						rows.Scan(&t)
						m.suggestions = append(m.suggestions, t)
					}
					rows.Close()
				}
			}
			return m, cmd
		}

		if m.searching {
			if s == "enter" || s == "esc" { m.searching = false; m.refreshList(); return m, nil }
			m.searchInput, cmd = m.searchInput.Update(msg)
			term := m.searchInput.Value()
			query := `SELECT id, parent_id, title, content, updated_at FROM pages WHERE id IN (SELECT rowid FROM pages_search WHERE pages_search MATCH ?)`
			rows, _ := m.db.Query(query, `"`+term+`"*`)
			var results []list.Item
			if rows != nil {
				defer rows.Close()
				for rows.Next() {
					var i item
					rows.Scan(&i.id, &i.parentID, &i.title, &i.content, &i.updatedAt)
					results = append(results, i)
				}
			}
			m.list.SetItems(results)
			return m, cmd
		}

		if m.deleting {
			if s == "y" {
				if active, ok := m.list.SelectedItem().(item); ok {
					m.db.Exec("UPDATE pages SET parent_id = ? WHERE parent_id = ?", active.parentID, active.id)
					m.db.Exec("DELETE FROM pages WHERE id = ?", active.id)
					m.refreshList()
				}
				m.deleting = false
				return m, nil
			}
			if s == "n" || s == "esc" { m.deleting = false; return m, nil }
			return m, nil
		}

		if !m.focusOnContent {
			switch s {
			case ",": 
				if m.sidebarWidth > 15 { m.sidebarWidth -= 2; m.resizePanes() }
				return m, nil
			case ".": 
				if m.sidebarWidth < 80 { m.sidebarWidth += 2; m.resizePanes() }
				return m, nil
			case " ":
				if a, ok := m.list.SelectedItem().(item); ok && a.hasChildren {
					m.collapsedStates[a.id] = !m.collapsedStates[a.id]
					m.refreshList()
					return m, nil
				}
			}
		}

		if s == "l" && !m.focusOnContent { m.focusOnContent = true; return m, nil }
		if s == "h" && m.focusOnContent { m.focusOnContent = false; return m, nil }

		if m.focusOnContent {
			switch s {
			case "j": m.viewport.LineDown(1); return m, nil
			case "k": m.viewport.LineUp(1); return m, nil
			case "g": m.viewport.GotoTop(); return m, nil
			case "G": m.viewport.GotoBottom(); return m, nil
			case "esc": m.focusOnContent = false; return m, nil
			}
		}

		switch s {
		case "q", "ctrl+c": return m, tea.Quit
		case "u":
			if a, ok := m.list.SelectedItem().(item); ok && a.parentID != 0 {
				for idx, li := range m.list.Items() {
					if li.(item).id == a.parentID { m.list.Select(idx); break }
				}
			}
		case "n", "N":
			m.creatingNew, m.createRoot = true, (s == "N")
			m.input.Focus(); m.input.SetValue(""); return m, textinput.Blink
		case "s":
			m.searching = true
			m.searchInput.Focus(); m.searchInput.SetValue(""); return m, textinput.Blink
		case "d": m.deleting = true; return m, nil
		case "x":
			m.statusMsg = m.startExport()
			m.statusTimer = time.Now().Add(5 * time.Second)
			return m, nil
		case "i":
			m.importing = true    // NEU: Setzt das Programm in den Import-Modus
			m.input.Focus()       // Setzt den Cursor ins Eingabefeld
			m.input.SetValue("")  // Leert das Eingabefeld
			return m, textinput.Blink
		case "e":
			if a, ok := m.list.SelectedItem().(item); ok {
				f, _ := os.CreateTemp("", "wiki-*.md")
				os.WriteFile(f.Name(), []byte(a.content), 0644)
				editor := os.Getenv("EDITOR")
				if editor == "" { if runtime.GOOS == "windows" { editor = "notepad" } else { editor = "nano" } }
				c := exec.Command(editor, f.Name())
				c.Env = append(os.Environ(), "LANG=en_US.UTF-8", "LC_ALL=en_US.UTF-8")
				return m, tea.Batch(tea.ExecProcess(c, func(err error) tea.Msg {
					upd, _ := os.ReadFile(f.Name())
					return editFinishedMsg{content: string(upd)}
				}), tea.ClearScreen)
			}
		}
	}

	if !m.focusOnContent && !m.creatingNew && !m.searching && !m.deleting && !m.importing {
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	}

	if !m.creatingNew && !m.searching && !m.deleting && !m.importing {
		if i, ok := m.list.SelectedItem().(item); ok { m.renderPage(i) }
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if m.importing {
		view := "IMPORT DIRECTORY PATH (z.B. ./export_2026...):\n" + m.input.View()
		return docStyle.Render(view)
	}
	if m.creatingNew {
		view := "NEW PAGE TITLE:\n" + m.input.View()
		if len(m.suggestions) > 0 {
			view += "\n\n" + suggestionStyle.Render("Existing or similar pages:") + "\n"
			for _, s := range m.suggestions {
				view += suggestionStyle.Render("• " + s) + "\n"
			}
		}
		return docStyle.Render(view)
	}
	if m.searching { return docStyle.Render("SEARCH:\n" + m.searchInput.View() + "\n\n" + m.list.View()) }
	
	sidebarStyle := lipgloss.NewStyle().Width(m.sidebarWidth).Border(lipgloss.NormalBorder(), false, true, false, false).PaddingRight(2)
	contentView := m.viewport.View()
	if m.focusOnContent { contentView = activeContentStyle.Render(contentView) } else { contentView = inactiveContentStyle.Render(contentView) }
	
	rightPane := lipgloss.JoinVertical(lipgloss.Left, dateStyle.Render(m.lastUpdate), contentView)

	if m.deleting {
		a, _ := m.list.SelectedItem().(item)
		box := confirmBoxStyle.Render(warnStyle.Render("DELETE PAGE?\n\n") + fmt.Sprintf("Delete '%s' permanently?\n(Sub-pages will be moved up)\n[y] Yes  [n] No", a.title))
		rightPane = lipgloss.Place(m.width-m.sidebarWidth-8, m.height-6, lipgloss.Center, lipgloss.Center, box)
	}
	
	mainView := lipgloss.JoinHorizontal(lipgloss.Top, sidebarStyle.Render(m.list.View()), rightPane)
	
	var footer string
	if m.statusMsg != "" && time.Now().Before(m.statusTimer) {
		footer = successStyle.Render("✔ " + m.statusMsg)
	} else {
		if m.statusMsg != "" {
			m.statusMsg = "" // Reset
		}
		help := fmt.Sprintf("[L: Content | Space: Toggle | , / . : Resize] | n: child | e: edit | s: search | x: export | i: import | q: quit")
		footer = helpStyle.Render(help)
	}

	return docStyle.Render(lipgloss.JoinVertical(lipgloss.Left, mainView, footer))
}

// --- Export & Import ---

func (m *model) exportRecursive(parentID int, currentPath string) error {
	query := "SELECT id, title, content FROM pages WHERE parent_id = ?"
	rows, err := m.db.Query(query, parentID)
	if err != nil { return err }
	defer rows.Close()

	for rows.Next() {
		var id int
		var title, content string
		rows.Scan(&id, &title, &content)

		re := regexp.MustCompile(`[^a-zA-Z0-9_\-]`)
		cleanTitle := re.ReplaceAllString(title, "_")
		pagePath := filepath.Join(currentPath, cleanTitle)

		os.MkdirAll(pagePath, 0755)
		os.WriteFile(filepath.Join(pagePath, "index.md"), []byte(content), 0644)

		m.exportRecursive(id, pagePath)
	}
	return nil
}

func (m *model) startExport() string {
	dirName := "export_" + time.Now().Format("20060102_1504")
	err := m.exportRecursive(0, dirName)
	if err != nil {
		return "Export failed: " + err.Error()
	}
	return "Exported to ./" + dirName
}

func (m *model) startImport(dirPath string) {
	absPath, err := filepath.Abs(dirPath)
	if err != nil { return }

	// idMap merkt sich: "Ordnerpfad" -> "Datenbank-ID"
	idMap := make(map[string]int)
	
	// Das Startverzeichnis ist die Root-Ebene (0)
	idMap[absPath] = 0

	filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil { return err }
		
		// WICHTIG: Wir triggern auf den ORDNER, nicht auf die index.md!
		// So garantieren wir, dass Eltern vor ihren Kindern in die DB geschrieben werden.
		if info.IsDir() {
			// Den übergeordneten Export-Ordner selbst überspringen
			if path == absPath {
				return nil
			}

			title := filepath.Base(path)
			parentDir := filepath.Dir(path)
			parentID := idMap[parentDir] // ID des Eltern-Ordners holen

			// Wir lesen den Inhalt der index.md in diesem Ordner
			content, _ := os.ReadFile(filepath.Join(path, "index.md"))

			// Jetzt MIT korrekter parentID einfügen
			res, err := m.db.Exec("INSERT INTO pages (title, content, parent_id) VALUES (?, ?, ?)", title, string(content), parentID)
			
			if err == nil {
				// Die neu erstellte ID aus der DB merken, falls dieser Ordner Unterordner hat
				newID, _ := res.LastInsertId()
				idMap[path] = int(newID)
			}
		}
		return nil
	})
}

func main() {
	db := initDB()
	defer db.Close()
	p := tea.NewProgram(initialModel(db), tea.WithAltScreen())
	if _, err := p.Run(); err != nil { fmt.Fprintf(os.Stderr, "Error: %v", err); os.Exit(1) }
}