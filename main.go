package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- TOKYO NIGHT THEME ---
var (
	bgDark   = lipgloss.Color("#1a1b26")
	bgLight  = lipgloss.Color("#24283b")
	fg       = lipgloss.Color("#a9b1d6")
	blue     = lipgloss.Color("#7aa2f7")
	green    = lipgloss.Color("#9ece6a")
	magenta  = lipgloss.Color("#bb9af7")
	red      = lipgloss.Color("#f7768e")
	yellow   = lipgloss.Color("#e0af68")
	
	cursorStyle        = lipgloss.NewStyle().Foreground(bgDark).Background(blue)
	topBarStyle        = lipgloss.NewStyle().Foreground(fg).Background(bgLight).Padding(0, 1)
	statusNormalStyle  = lipgloss.NewStyle().Foreground(bgDark).Background(blue).Bold(true).Padding(0, 1)
	statusInsertStyle  = lipgloss.NewStyle().Foreground(bgDark).Background(green).Bold(true).Padding(0, 1)
	statusGitStyle     = lipgloss.NewStyle().Foreground(fg).Background(bgLight).Padding(0, 1)
	statusMessageStyle = lipgloss.NewStyle().Foreground(yellow).Background(bgDark).Padding(0, 1)
	
	logoStyle = lipgloss.NewStyle().Bold(true).Foreground(blue).Align(lipgloss.Center)
	welcomeSubStyle = lipgloss.NewStyle().Foreground(fg).Align(lipgloss.Center)
)

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.textarea.SetHeight(m.height - 2)
		m.textarea.SetWidth(m.width)
		m.commandInput.Width = m.width - 4
		return m, nil
	case clearStatusMsg:
		m.statusMessage = ""
		return m, nil
	}

	if m.isWelcome {
		if _, ok := msg.(tea.KeyMsg); ok { m.isWelcome = false; return m, nil }
	}

	switch m.mode {
	// ================= NORMAL MODE =================
	case modeNormal:
		if k, ok := msg.(tea.KeyMsg); ok {
			key := k.String()
			
			// --- ОДИНОЧНЫЕ КОМАНДЫ (сразу сбрасываем буфер) ---
			switch key {
			case "i": m.mode = modeInsert; m.textarea.Focus(); m.keyBuffer = ""; return m, textarea.Blink
			case "a": m.mode = modeInsert; m.textarea.Focus(); m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRight}); m.keyBuffer = ""; return m, textarea.Blink
			case ":": m.mode = modeCommand; m.commandInput.Focus(); m.commandInput.SetValue(""); m.keyBuffer = ""; return m, textinput.Blink
			
			case "h": m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyLeft}); m.keyBuffer = ""; return m, nil
			case "j": m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyDown}); m.keyBuffer = ""; return m, nil
			case "k": m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyUp}); m.keyBuffer = ""; return m, nil
			case "l": m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRight}); m.keyBuffer = ""; return m, nil
			
			// Прыжок в начало/конец СТРОКИ (эмулируем Ctrl+A и Ctrl+E, это стандарт терминала)
			case "0": m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyCtrlA}); m.keyBuffer = ""; return m, nil
			case "$": m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyCtrlE}); m.keyBuffer = ""; return m, nil

			// Прыжок в конец ФАЙЛА (G)
			case "G":
				linesCount := len(strings.Split(m.textarea.Value(), "\n"))
				// Просто "зажимаем" кнопку вниз до упора
				for i := 0; i < linesCount; i++ {
					m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyDown})
				}
				// И ставим курсор в конец строки
				m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
				m.keyBuffer = ""
				return m, nil

			case "u": m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyCtrlZ}); m.keyBuffer = ""; return m, nil
			case "p": m = m.pasteFromClipboard(); m.keyBuffer = ""; return m, clearStatusCmd()
			}

			// --- СЛОЖНЫЕ КОМАНДЫ (накапливаем в буфер) ---
			m.keyBuffer += key

			if strings.HasSuffix(m.keyBuffer, "dd") {
				m = m.deleteCurrentLine()
				m.keyBuffer = ""
				return m, clearStatusCmd()
			}
			if strings.HasSuffix(m.keyBuffer, "yy") {
				m = m.copyLineToClipboard()
				m.keyBuffer = ""
				return m, clearStatusCmd()
			}
			
			// Прыжок в начало ФАЙЛА (gg)
			if strings.HasSuffix(m.keyBuffer, "gg") {
				linesCount := len(strings.Split(m.textarea.Value(), "\n"))
				// "Зажимаем" кнопку вверх до упора
				for i := 0; i < linesCount; i++ {
					m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyUp})
				}
				// И ставим курсор в начало строки
				m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
				m.keyBuffer = ""
				return m, nil
			}

			// Очистка буфера от мусора (если нажал фигню)
			if len(m.keyBuffer) > 2 { m.keyBuffer = "" }
			return m, nil
		}

	// ================= INSERT MODE =================
	case modeInsert:
		if k, ok := msg.(tea.KeyMsg); ok {
			if k.String() == "esc" { m.mode = modeNormal; m.textarea.Blur(); return m, nil }
			
			// Автозакрытие скобок
			char := k.String()
			pairs := map[string]string{ "{": "}", "[": "]", "(": ")", "\"": "\"", "'": "'" }
			if closing, exists := pairs[char]; exists {
				m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(char + closing)})
				m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyLeft})
				m.isDirty = true
				return m, nil
			}
		}
		m.isDirty = true
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd

	// ================= COMMAND MODE =================
	case modeCommand:
		if k, ok := msg.(tea.KeyMsg); ok {
			if k.String() == "esc" { m.mode = modeNormal; m.commandInput.Blur(); return m, nil }
			if k.String() == "enter" { return m.executeCommand(m.commandInput.Value()) }
		}
		m.commandInput, cmd = m.commandInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	if m.isWelcome {
		logo := `
   _____         __   ______    ___ __ 
  / __/___ ____ / /_ / __/ /_  (_) // /_
 / _// __// __// __// _// __/ / / // __/
/_/  \__//_/   \__//___/\__/ /_//_/\__/ 
                                        
`
		fullContent := lipgloss.JoinVertical(lipgloss.Center,
			logoStyle.Render(logo),
			welcomeSubStyle.Render("Премиальный TUI Редактор • Tokyo Night"),
			"\n",
			lipgloss.NewStyle().Foreground(magenta).Render("[i] Вставка • [:] Команды • [dd] Удалить • [gg] Вверх"),
		)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, fullContent)
	}

	fname := m.filename
	if fname == "" { fname = "Новый_файл" }
	dirtyIcon := ""
	if m.isDirty { dirtyIcon = lipgloss.NewStyle().Foreground(red).Render(" ●") }
	
	fileTab := lipgloss.NewStyle().Foreground(lipgloss.Color(m.fileType.color)).Render(m.fileType.icon + " " + fname)
	topBarLeft := topBarStyle.Render(fileTab + dirtyIcon)
	topBarRight := topBarStyle.Render(m.fileType.name)
	
	topGap := strings.Repeat(" ", max(0, m.width-lipgloss.Width(topBarLeft)-lipgloss.Width(topBarRight)))
	topBar := lipgloss.NewStyle().Background(bgLight).Render(topBarLeft + topGap + topBarRight)

	var bottomBar string
	if m.statusMessage != "" {
		bottomBar = statusMessageStyle.Width(m.width).Render(" " + m.statusMessage)
	} else if m.mode == modeCommand {
		bottomBar = lipgloss.NewStyle().Background(bgLight).Width(m.width).Render(m.commandInput.View())
	} else {
		modeStr, style := " NORMAL ", statusNormalStyle
		if m.mode == modeInsert { modeStr, style = " INSERT ", statusInsertStyle }
		
		git := ""
		if m.gitBranch != "" { git = statusGitStyle.Render(" " + m.gitBranch) }
		
		pos := lipgloss.NewStyle().Foreground(blue).Background(bgLight).Padding(0, 1).Render(
			fmt.Sprintf("Ln %d", m.textarea.Line()+1),
		)
		
		leftSide := lipgloss.JoinHorizontal(lipgloss.Top, style.Render(modeStr), git)
		botGap := strings.Repeat(" ", max(0, m.width-lipgloss.Width(leftSide)-lipgloss.Width(pos)))
		bottomBar = lipgloss.NewStyle().Background(bgLight).Render(leftSide + botGap + pos)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		topBar,
		m.textarea.View(),
		bottomBar,
	)
}

func max(a, b int) int {
	if a > b { return a }
	return b
}

func main() {
	arg := ""
	if len(os.Args) > 1 { arg = os.Args[1] }
	p := tea.NewProgram(initialModel(arg), tea.WithAltScreen())
	if _, err := p.Run(); err != nil { fmt.Println(err) }
}
