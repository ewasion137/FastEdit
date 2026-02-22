package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- СТИЛИ ---
var (
	statusNormalStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFF")).Background(lipgloss.Color("#5A56E0")).Bold(true)
	statusInsertStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFF")).Background(lipgloss.Color("#04B575")).Bold(true)
	statusGitStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFF")).Background(lipgloss.Color("#FF5F87")).Padding(0, 1)
	statusMessageStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00")).Background(lipgloss.Color("#5A56E0"))
	logoStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575")).Align(lipgloss.Center)
)

type editorMode int
const (
	modeNormal editorMode = iota
	modeInsert
	modeCommand
)

type clearStatusMsg struct{}

// --- МОДЕЛЬ ---
type model struct {
	textarea     textarea.Model
	commandInput textinput.Model
	mode         editorMode
	isDirty      bool
	isWelcome    bool
	statusMessage string
	filename     string
	gitBranch    string
	width        int
	height       int
}

func getGitBranch() string {
	cmd := exec.Command("git", "branch", "--show-current")
	out, _ := cmd.Output()
	return strings.TrimSpace(string(out))
}

func initialModel(filename string) model {
	ti := textarea.New()
	ti.ShowLineNumbers = true
	ti.Placeholder = "Начни путь..."

	ci := textinput.New()
	ci.Prompt = ":"

	isWelcome := filename == ""
	if !isWelcome {
		content, err := os.ReadFile(filename)
		if err == nil {
			ti.SetValue(string(content))
		}
	}

	return model{
		textarea:     ti,
		commandInput: ci,
		mode:         modeNormal,
		isWelcome:    isWelcome,
		filename:     filename,
		gitBranch:    getGitBranch(),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

// --- UPDATE ---
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.textarea.SetHeight(m.height - 1)
		m.textarea.SetWidth(m.width)
		m.commandInput.Width = m.width - 2
		return m, nil
	case clearStatusMsg:
		m.statusMessage = ""
		return m, nil
	}

	// Если экран приветствия - любая кнопка убирает его
	if m.isWelcome {
		if _, ok := msg.(tea.KeyMsg); ok {
			m.isWelcome = false
			return m, nil
		}
	}

	switch m.mode {
	case modeNormal:
		if k, ok := msg.(tea.KeyMsg); ok {
			switch k.String() {
			case "i":
				m.mode = modeInsert
				m.textarea.Focus()
				return m, textarea.Blink
			case "a":
				m.mode = modeInsert
				m.textarea.Focus()
				// Эмулируем нажатие "вправо" через Update самого textarea
				m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRight})
				return m, textarea.Blink
			
			// VIM НАВИГАЦИЯ через эмуляцию клавиш стрелок
			case "h": m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyLeft})
			case "j": m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyDown})
			case "k": m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyUp})
			case "l": m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRight})

			case ":":
				m.mode = modeCommand
				m.commandInput.Focus()
				m.commandInput.SetValue("")
				return m, nil
			}
		}

	case modeInsert:
		if k, ok := msg.(tea.KeyMsg); ok && k.String() == "esc" {
			m.mode = modeNormal
			m.textarea.Blur()
			return m, nil
		}
		m.isDirty = true
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd

	case modeCommand:
		if k, ok := msg.(tea.KeyMsg); ok {
			if k.String() == "esc" {
				m.mode = modeNormal
				m.commandInput.Blur()
				return m, nil
			}
			if k.String() == "enter" {
				command := m.commandInput.Value()
				m, cmd = m.executeCommand(command)
				return m, cmd
			}
		}
		m.commandInput, cmd = m.commandInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

// --- VIEW ---
func (m model) View() string {
	if m.isWelcome {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, 
			logoStyle.Render("FAST EDIT\n\n[i] Вставка | [:] Команды"))
	}

	var bottomBar string
	if m.statusMessage != "" {
		bottomBar = statusMessageStyle.Width(m.width).Render(" " + m.statusMessage)
	} else if m.mode == modeCommand {
		bottomBar = m.commandInput.View()
	} else {
		modeStr := " NORMAL "
		style := statusNormalStyle
		if m.mode == modeInsert {
			modeStr = " INSERT "
			style = statusInsertStyle
		}
		
		gitBlock := ""
		if m.gitBranch != "" {
			gitBlock = statusGitStyle.Render(" " + m.gitBranch)
		}

		dirty := ""
		if m.isDirty { dirty = "*" }
		fname := m.filename
		if fname == "" { fname = "[No Name]" }
		
		left := lipgloss.JoinHorizontal(lipgloss.Top, style.Render(modeStr), gitBlock, " "+fname+dirty)
		pos := fmt.Sprintf("Ln %d ", m.textarea.Line()+1)
		
		gap := strings.Repeat(" ", max(0, m.width-lipgloss.Width(left)-lipgloss.Width(pos)))
		bottomBar = style.Render(left + gap + pos)
	}

	return fmt.Sprintf("%s\n%s", m.textarea.View(), bottomBar)
}

func (m model) executeCommand(cmdStr string) (model, tea.Cmd) {
	parts := strings.Fields(cmdStr)
	m.mode = modeNormal
	m.commandInput.Blur()

	if len(parts) == 0 { return m, nil }

	switch parts[0] {
	case "q", "quit":
		if m.isDirty {
			m.statusMessage = "Не сохранено! Добавь !"
			return m, clearStatusCmd()
		}
		return m, tea.Quit
	case "q!":
		return m, tea.Quit
	case "w", "write":
		if len(parts) > 1 { m.filename = parts[1] }
		return m.saveFile()
	case "commit":
		if len(parts) < 2 {
			m.statusMessage = "Error: Нужно сообщение"
		} else {
			m, _ = m.saveFile()
			exec.Command("git", "add", m.filename).Run()
			msg := strings.Join(parts[1:], " ")
			exec.Command("git", "commit", "-m", msg).Run()
			m.statusMessage = "Коммит создан!"
		}
		return m, clearStatusCmd()
	}
	return m, nil
}

func (m model) saveFile() (model, tea.Cmd) {
	if m.filename == "" {
		m.statusMessage = "Error: Нет имени"
		return m, clearStatusCmd()
	}
	_ = os.WriteFile(m.filename, []byte(m.textarea.Value()), 0644)
	m.statusMessage = "Сохранено!"
	m.isDirty = false
	m.gitBranch = getGitBranch()
	return m, clearStatusCmd()
}

func clearStatusCmd() tea.Cmd {
	return tea.Tick(time.Second*3, func(t time.Time) tea.Msg { return clearStatusMsg{} })
}

func max(a, b int) int { if a > b { return a }; return b }

func main() {
	arg := ""
	if len(os.Args) > 1 { arg = os.Args[1] }
	p := tea.NewProgram(initialModel(arg), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println(err)
	}
}
