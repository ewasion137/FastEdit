package main

import (
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

// --- VIM МАГИЯ ---

// Удаление текущей строки (dd)
func (m model) deleteCurrentLine() model {
	lines := strings.Split(m.textarea.Value(), "\n")
	curLine := m.textarea.Line()
	
	if len(lines) == 0 || curLine >= len(lines) { return m }

	// Вырезаем строку
	lines = append(lines[:curLine], lines[curLine+1:]...)
	m.textarea.SetValue(strings.Join(lines, "\n"))
	m.isDirty = true
	m.statusMessage = "Строка удалена"
	
	// Поправляем курсор
	if curLine >= len(lines) && curLine > 0 {
		m.textarea.SetCursor(len(strings.Join(lines[:curLine-1], "\n")))
	}
	return m
}

// --- БУФЕР ОБМЕНА ---
func (m model) copyLineToClipboard() model {
	lines := strings.Split(m.textarea.Value(), "\n")
	curLine := m.textarea.Line()
	if curLine >= 0 && curLine < len(lines) {
		clipboard.WriteAll(lines[curLine] + "\n")
		m.statusMessage = "Строка скопирована (yy)"
	}
	return m
}

func (m model) pasteFromClipboard() model {
	str, err := clipboard.ReadAll()
	if err == nil && str != "" {
		m.textarea.InsertString(str)
		m.isDirty = true
		m.statusMessage = "Вставлено!"
	}
	return m
}

// --- ФАЙЛОВЫЕ ОПЕРАЦИИ ---
func (m model) openFile(filename string) (model, tea.Cmd) {
	m.filename = filename
	m.fileType = getFileType(filename)
	content, err := os.ReadFile(filename)
	if err == nil {
		m.textarea.SetValue(string(content))
		m.statusMessage = "Открыт: " + filename
	} else {
		m.textarea.SetValue("")
		m.statusMessage = "Новый файл: " + filename
	}
	m.isDirty = false
	m.gitBranch = getGitBranch()
	return m, clearStatusCmd()
}

func (m model) saveFile() (model, tea.Cmd) {
	if m.filename == "" {
		m.statusMessage = "Ошибка: Нет имени файла"
		return m, clearStatusCmd()
	}
	m.fileType = getFileType(m.filename)
	err := os.WriteFile(m.filename, []byte(m.textarea.Value()), 0644)
	if err != nil {
		m.statusMessage = "Ошибка: " + err.Error()
	} else {
		m.statusMessage = "💾 Сохранено успешно!"
		m.isDirty = false
		m.gitBranch = getGitBranch()
	}
	return m, clearStatusCmd()
}

// --- ВЫПОЛНЕНИЕ КОМАНД ---
func (m model) executeCommand(cmdStr string) (model, tea.Cmd) {
	parts := strings.Fields(cmdStr)
	m.mode = modeNormal
	m.commandInput.Blur()

	if len(parts) == 0 { return m, nil }

	switch parts[0] {
	case "q", "quit":
		if m.isDirty {
			m.statusMessage = "⚠ Не сохранено! Используй :q!"
			return m, clearStatusCmd()
		}
		return m, tea.Quit
	case "q!": return m, tea.Quit
	case "w", "write":
		if len(parts) > 1 { m.filename = parts[1] }
		return m.saveFile()
	case "o", "open":
		if len(parts) < 2 {
			m.statusMessage = "Укажи имя файла (:o main.go)"
			return m, clearStatusCmd()
		}
		return m.openFile(parts[1])
	case "commit":
		if len(parts) < 2 {
			m.statusMessage = "Ошибка: Нужно сообщение"
		} else {
			m, _ = m.saveFile()
			exec.Command("git", "add", m.filename).Run()
			msg := strings.Join(parts[1:], " ")
			exec.Command("git", "commit", "-m", msg).Run()
			m.statusMessage = "✨ Коммит отправлен!"
		}
		return m, clearStatusCmd()
	default:
		m.statusMessage = "Неизвестная команда"
		return m, clearStatusCmd()
	}
}

func clearStatusCmd() tea.Cmd {
	return tea.Tick(time.Second*3, func(t time.Time) tea.Msg { return clearStatusMsg{} })
}