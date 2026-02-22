package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
)

type editorMode int

const (
	modeNormal editorMode = iota
	modeInsert
	modeCommand
)

type clearStatusMsg struct{}

// Структура для верхней панели (вкладок)
type fileInfo struct {
	icon  string
	color string
	name  string
}

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
	
	// Для сложных Vim-команд (типа 'dd', 'gg')
	keyBuffer    string 
	
	// Данные для красоты
	fileType     fileInfo
	startupTime  time.Time
}

func getGitBranch() string {
	cmd := exec.Command("git", "branch", "--show-current")
	out, _ := cmd.Output()
	return strings.TrimSpace(string(out))
}

// Умное определение типа файла
func getFileType(filename string) fileInfo {
	if filename == "" {
		return fileInfo{icon: "📝", color: "#a9b1d6", name: "Text"}
	}
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".go": return fileInfo{icon: "🐹", color: "#7dcfff", name: "Go"}
	case ".js", ".ts": return fileInfo{icon: "🟨", color: "#e0af68", name: "JS/TS"}
	case ".py": return fileInfo{icon: "🐍", color: "#9ece6a", name: "Python"}
	case ".html", ".css": return fileInfo{icon: "🌐", color: "#f7768e", name: "Web"}
	case ".json": return fileInfo{icon: "📋", color: "#b4f9f8", name: "JSON"}
	case ".md": return fileInfo{icon: "📘", color: "#bb9af7", name: "Markdown"}
	default: return fileInfo{icon: "📄", color: "#a9b1d6", name: strings.ToUpper(strings.TrimPrefix(ext, "."))}
	}
}

func initialModel(filename string) model {
	ti := textarea.New()
	ti.ShowLineNumbers = true
	ti.Placeholder = "// Напиши что-нибудь гениальное..."
	
	// Настройка курсора
	ti.Cursor.Style = cursorStyle

	ci := textinput.New()
	ci.Prompt = "🚀 :"
	ci.Cursor.Style = cursorStyle

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
		fileType:     getFileType(filename),
		startupTime:  time.Now(),
	}
}