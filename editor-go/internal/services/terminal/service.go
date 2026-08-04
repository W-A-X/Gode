package terminal

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

type TerminalService struct {
	terminals map[string]*Terminal
	mu        sync.RWMutex
}

type Terminal struct {
	ID        string
	Command   string
	Args      []string
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	cols      int
	rows      int
	processID int
	exitCode  int
	running   bool
	mu        sync.Mutex
}

type ProcessInfo struct {
	ID        string `json:"id"`
	ProcessID int    `json:"processId"`
	Command   string `json:"command"`
	Title     string `json:"title"`
}

func NewTerminalService() *TerminalService {
	return &TerminalService{
		terminals: make(map[string]*Terminal),
	}
}

func (s *TerminalService) CreateTerminal(id, shellPath string, args []string, cols, rows int, workDir string, env map[string]string) (*ProcessInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cmd := exec.Command(shellPath, args...)
	cmd.Dir = workDir
	if env != nil {
		cmd.Env = os.Environ()
		for k, v := range env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}
	cmd.Env = append(cmd.Env, fmt.Sprintf("TERM=xterm-256color"))

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	cmd.Stderr = cmd.Stdout

	term := &Terminal{
		ID:      id,
		Command: shellPath,
		Args:    args,
		cmd:     cmd,
		stdin:   stdinPipe,
		stdout:  stdoutPipe,
		cols:    cols,
		rows:    rows,
		running: false,
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start terminal: %w", err)
	}

	term.processID = cmd.Process.Pid
	term.running = true

	go func() {
		err := cmd.Wait()
		term.mu.Lock()
		term.running = false
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				term.exitCode = exitErr.ExitCode()
			} else {
				term.exitCode = -1
			}
		} else {
			term.exitCode = 0
		}
		term.mu.Unlock()
	}()

	s.terminals[id] = term

	return &ProcessInfo{
		ID:        id,
		ProcessID: term.processID,
		Command:   shellPath,
		Title:     shellPath,
	}, nil
}

func (s *TerminalService) KillTerminal(id string) error {
	s.mu.RLock()
	term, ok := s.terminals[id]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("terminal %s not found", id)
	}

	term.mu.Lock()
	defer term.mu.Unlock()

	if !term.running {
		return nil
	}

	if term.cmd.Process != nil {
		return term.cmd.Process.Signal(syscall.SIGTERM)
	}
	return nil
}

func (s *TerminalService) SendInput(id string, data string) error {
	s.mu.RLock()
	term, ok := s.terminals[id]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("terminal %s not found", id)
	}

	term.mu.Lock()
	defer term.mu.Unlock()

	if !term.running {
		return fmt.Errorf("terminal %s is not running", id)
	}

	_, err := term.stdin.Write([]byte(data))
	return err
}

func (s *TerminalService) ResizeTerminal(id string, cols, rows int) error {
	s.mu.RLock()
	term, ok := s.terminals[id]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("terminal %s not found", id)
	}

	term.mu.Lock()
	defer term.mu.Unlock()

	term.cols = cols
	term.rows = rows
	return nil
}

func (s *TerminalService) GetProcessInfo(id string) (*ProcessInfo, error) {
	s.mu.RLock()
	term, ok := s.terminals[id]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("terminal %s not found", id)
	}

	term.mu.Lock()
	defer term.mu.Unlock()

	return &ProcessInfo{
		ID:        term.ID,
		ProcessID: term.processID,
		Command:   term.Command,
		Title:     term.Command,
	}, nil
}

func (s *TerminalService) GetStdoutReader(id string) (io.Reader, error) {
	s.mu.RLock()
	term, ok := s.terminals[id]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("terminal %s not found", id)
	}
	return term.stdout, nil
}

func (s *TerminalService) IsRunning(id string) bool {
	s.mu.RLock()
	term, ok := s.terminals[id]
	s.mu.RUnlock()
	if !ok {
		return false
	}
	term.mu.Lock()
	defer term.mu.Unlock()
	return term.running
}

func (s *TerminalService) GetExitCode(id string) int {
	s.mu.RLock()
	term, ok := s.terminals[id]
	s.mu.RUnlock()
	if !ok {
		return -1
	}
	term.mu.Lock()
	defer term.mu.Unlock()
	return term.exitCode
}

func (s *TerminalService) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, term := range s.terminals {
		if term.cmd.Process != nil {
			term.cmd.Process.Kill()
		}
	}
	s.terminals = make(map[string]*Terminal)
}

func getDefaultShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}
	return shell
}

func init() {
	_ = time.Now()
}