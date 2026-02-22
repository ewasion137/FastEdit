package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- СТИЛИ ---
var (
	// Стили для статус-бара в зависимости от режима
	statusNormalStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFF")).Background(lipgloss.Color("#5A56E0"))
	statusInsertStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFF")).Background(lipgloss.Color("#04B575"))
	statusCmdStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFF")).Background(lipgloss.Color("#B45EEA"))
	
	statusMessageStyle = lipgloss.NewStyle().Inherit(statusNormalStyle).Foreground(lipgloss.Color("#FFFF00"))
	logoStyle          = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575")).Align(lipgloss.Center)
)

// --- РЕЖИМЫ РЕДАКТОРА ---
type editorMode int
const (
	modeNormal editorMode = iota
	modeInsert
	modeCommand
)

// Сообщение для сброса статуса
type clearStatusMsg struct{}

// --- ГЛАВНАЯ МОДЕЛЬ ---
type model struct {
	textarea     textarea.Model
	commandInput textinput.Model
	mode         editorMode
	isDirty      bool
	isWelcome    bool
	statusMessage string
	filename     string
	width        int
	height       int
}

func initialModel(filename string) model {
	ti := textarea.New()
	// ВАЖНО: Редактор стартует в Normal режиме, поэтому поле НЕ в фокусе.
	// ti.Focus() -> УБРАЛИ
	ti.ShowLineNumbers = true
	ti.Placeholder = "..."

	ci := textinput.New()
	ci.Prompt = ":"
	ci.CharLimit = 100

	isWelcome := false
	if filename != "" {
		content, err := os.ReadFile(filename)
		if err == nil {
			ti.SetValue(string(content))
		}
	} else {
		isWelcome = true
	}

	return model{
		textarea:     ti,
		commandInput: ci,
		mode:         modeNormal, // Начинаем в NORMAL режиме!
		isWelcome:    isWelcome,
		filename:     filename,
		isDirty:      false,
	}
}

func (m model) Init() tea.Cmd {
	return nil // Убрали Blink, так как курсора в Normal режиме нет
}

// --- UPDATE ---
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
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
	case tea.KeyMsg:
		if m.isWelcome {
			m.isWelcome = false
		}
	}

	// ЛОГИКА РЕЖИМОВ
	switch m.mode {
	// --- NORMAL MODE --- (команды одной кнопкой)
	case modeNormal:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			// Переход в Insert режим
			case "i":
				m.mode = modeInsert
				m.textarea.Focus()
				return m, textarea.Blink // Запускаем мигание курсора
			// Переход в Command режим
			case ":":
				m.mode = modeCommand
				m.commandInput.Focus()
				m.commandInput.SetValue("")
				return m, nil
			case "ctrl+s":
				m, cmd = m.saveFile()
				return m, cmd
			// В будущем здесь будут команды 'dd', 'yy', 'p' и т.д.
			}
		}

	// --- INSERT MODE --- (ввод текста)
	case modeInsert:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			// Возврат в Normal режим
			case "esc":
				m.mode = modeNormal
				m.textarea.Blur() // Снимаем фокус с поля ввода
				return m, nil
			// Любое другое нажатие - это ввод текста
			default:
				m.isDirty = true
			}
		}

	// --- COMMAND MODE --- (ввод команд в строке)
	case modeCommand:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				m.mode = modeNormal
				m.commandInput.Blur()
				return m, nil
			case "enter":
				command := m.commandInput.Value()
				m, cmd = m.executeCommand(command)
				return m, cmd
			}
		}
	}

	// Передаем сообщение активному компоненту
	if m.mode == modeInsert {
		m.textarea, cmd = m.textarea.Update(msg)
	} else if m.mode == modeCommand {
		m.commandInput, cmd = m.commandInput.Update(msg)
	}
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// --- VIEW ---
func (m model) View() string {
	if m.isWelcome {
		// ASCII art...
		logo := `... Твой ASCII арт ...`
		info := "\nНажми 'i', чтобы начать печатать."
		fullContent := lipgloss.JoinVertical(lipgloss.Center, logoStyle.Render(logo), info)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, fullContent)
	}

	var bottomBar string
	if m.statusMessage != "" {
		bottomBar = statusMessageStyle.Width(m.width).Render(m.statusMessage)
	} else if m.mode == modeCommand {
		bottomBar = m.commandInput.View()
	} else {
		// --- СТАТУС-БАР В NORMAL/INSERT РЕЖИМАХ ---
		var modeText string
		var style lipgloss.Style

		if m.mode == modeNormal {
			modeText = "-- NORMAL --"
			style = statusNormalStyle
		} else {
			modeText = "-- INSERT --"
			style = statusInsertStyle
		}
		
		dirtyFlag := ""
		if m.isDirty {
			dirtyFlag = "*"
		}
		
		filename := m.filename
		if filename == "" {
			filename = "[No Name]"
		}
		
		modeBlock := style.Render(modeText)
		fileBlock := style.Padding(0, 1).Render(filename + dirtyFlag)
		posBlock := style.Padding(0, 1).Render(fmt.Sprintf("Ln %d", m.textarea.Line()+1))
		
		w := m.width - lipgloss.Width(modeBlock) - lipgloss.Width(fileBlock) - lipgloss.Width(posBlock)
		if w < 0 { w = 0 }
		gap := style.Render(strings.Repeat(" ", w))

		bottomBar = lipgloss.JoinHorizontal(lipgloss.Top, modeBlock, fileBlock, gap, posBlock)
	}

	return fmt.Sprintf("%s\n%s", m.textarea.View(), bottomBar)
}


// --- ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ ---
func clearStatusCmd() tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

func (m model) executeCommand(cmdStr string) (model, tea.Cmd) {
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		m.mode = modeNormal
		return m, nil
	}

	command := parts[0]
	switch command {
	case "q", "quit":
		if m.isDirty {
			m.statusMessage = "Есть несохраненные изменения! (используй :q!)"
			m.mode = modeNormal
			return m, clearStatusCmd()
		}
		return m, tea.Quit
	case "q!", "quit!":
		return m, tea.Quit
	// Команды :w, :write, :save, :saveas делают одно и то же
	case "w", "write", "save", "saveas":
		if len(parts) > 1 {
			m.filename = parts[1] // ":w new.txt" -> сохраняем как new.txt
		}
		return m.saveFile()
	case "wq":
		newModel, cmd := m.saveFile()
		if newModel.statusMessage != "" && strings.HasPrefix(newModel.statusMessage, "Error") {
			return newModel, cmd
		}
		return newModel, tea.Batch(cmd, tea.Quit)
	default:
		m.statusMessage = fmt.Sprintf("Неизвестная команда: %s", command)
		m.mode = modeNormal
		return m, clearStatusCmd()
	}
}

func (m model) saveFile() (model, tea.Cmd) {
	if m.filename == "" {
		m.statusMessage = "Error: Имя файла не задано. Используй :w <имя файла>"
		m.mode = modeNormal
		return m, clearStatusCmd()
	}
	
	err := os.WriteFile(m.filename, []byte(m.textarea.Value()), 0644)
	if err != nil {
		m.statusMessage = fmt.Sprintf("Error: %v", err)
	} else {
		m.statusMessage = fmt.Sprintf("Файл '%s' сохранен.", m.filename)
		m.isDirty = false
	}
	
	m.mode = modeNormal
	return m, clearStatusCmd()
}

func main() {
	// ... код main без изменений ...
	filename := ""
	if len(os.Args) > 1 {
		filename = os.Args[1]
	}

	p := tea.NewProgram(initialModel(filename), tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Fatal error: %v", err)
		os.Exit(1)
	}
}