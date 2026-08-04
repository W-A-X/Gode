package filesystem

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

type FileType int

const (
	FileTypeUnknown      FileType = 0
	FileTypeFile         FileType = 1
	FileTypeDirectory    FileType = 2
	FileTypeSymbolicLink FileType = 64
)

type Stat struct {
	Type        FileType `json:"type"`
	Size        int64    `json:"size"`
	Mtime       int64    `json:"mtime"`
	Permissions int32    `json:"permissions"`
}

type DirEntry struct {
	Name string   `json:"name"`
	Type FileType `json:"type"`
}

type FileSystemService struct {
	openFiles   map[uint32]*os.File
	nextFd      uint32
	mu          sync.RWMutex
	watchers    map[string][]chan FileEvent
	nextWatchId uint32
	watchMu     sync.RWMutex
}

type FileEvent struct {
	Type int    `json:"type"`
	Path string `json:"path"`
}

func NewFileSystemService() *FileSystemService {
	return &FileSystemService{
		openFiles: make(map[uint32]*os.File),
		nextFd:    1,
		watchers:  make(map[string][]chan FileEvent),
	}
}

func (s *FileSystemService) Stat(path string) (*Stat, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	return fileInfoToStat(info), nil
}

func (s *FileSystemService) Realpath(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}

func (s *FileSystemService) Readdir(path string) ([]DirEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	result := make([]DirEntry, len(entries))
	for i, e := range entries {
		result[i] = DirEntry{
			Name: e.Name(),
			Type: fileTypeFromInfo(e.Type()),
		}
	}
	return result, nil
}

func (s *FileSystemService) Open(path string, mode string) (uint32, error) {
	flag := parseOpenMode(mode)
	f, err := os.OpenFile(path, flag, 0666)
	if err != nil {
		return 0, err
	}
	s.mu.Lock()
	fd := s.nextFd
	s.nextFd++
	s.openFiles[fd] = f
	s.mu.Unlock()
	return fd, nil
}

func (s *FileSystemService) Close(fd uint32) error {
	s.mu.Lock()
	f, ok := s.openFiles[fd]
	if ok {
		delete(s.openFiles, fd)
	}
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("file descriptor %d not found", fd)
	}
	return f.Close()
}

func (s *FileSystemService) Read(fd uint32, pos int64, length int) ([]byte, int, error) {
	s.mu.RLock()
	f, ok := s.openFiles[fd]
	s.mu.RUnlock()
	if !ok {
		return nil, 0, fmt.Errorf("file descriptor %d not found", fd)
	}
	buf := make([]byte, length)
	n, err := f.ReadAt(buf, pos)
	if err != nil && err != io.EOF {
		return nil, n, err
	}
	return buf[:n], n, nil
}

func (s *FileSystemService) Write(fd uint32, pos int64, data []byte, offset, length int) (int, error) {
	s.mu.RLock()
	f, ok := s.openFiles[fd]
	s.mu.RUnlock()
	if !ok {
		return 0, fmt.Errorf("file descriptor %d not found", fd)
	}
	n, err := f.WriteAt(data[offset:offset+length], pos)
	return n, err
}

func (s *FileSystemService) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (s *FileSystemService) WriteFile(path string, data []byte, mode string) error {
	return os.WriteFile(path, data, 0666)
}

func (s *FileSystemService) Mkdir(path string) error {
	return os.MkdirAll(path, 0755)
}

func (s *FileSystemService) Delete(path string, recursive bool) error {
	if recursive {
		return os.RemoveAll(path)
	}
	return os.Remove(path)
}

func (s *FileSystemService) Rename(src, dst string, overwrite bool) error {
	if !overwrite {
		if _, err := os.Stat(dst); err == nil {
			return fmt.Errorf("target already exists: %s", dst)
		}
	}
	return os.Rename(src, dst)
}

func (s *FileSystemService) Copy(src, dst string, overwrite bool) error {
	if !overwrite {
		if _, err := os.Stat(dst); err == nil {
			return fmt.Errorf("target already exists: %s", dst)
		}
	}
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	if srcInfo.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst)
}

func (s *FileSystemService) CloneFile(src, dst string) error {
	return copyFile(src, dst)
}

func (s *FileSystemService) Watch(path string) (uint32, <-chan FileEvent, func()) {
	s.watchMu.Lock()
	id := s.nextWatchId
	s.nextWatchId++
	ch := make(chan FileEvent, 100)
	key := path
	s.watchers[key] = append(s.watchers[key], ch)
	s.watchMu.Unlock()

	go s.pollWatcher(path, ch)

	dispose := func() {
		s.watchMu.Lock()
		watchers := s.watchers[key]
		for i, w := range watchers {
			if w == ch {
				s.watchers[key] = append(watchers[:i], watchers[i+1:]...)
				break
			}
		}
		if len(s.watchers[key]) == 0 {
			delete(s.watchers, key)
		}
		s.watchMu.Unlock()
		close(ch)
	}

	return id, ch, dispose
}

func (s *FileSystemService) Unwatch(id uint32) {
}

func (s *FileSystemService) pollWatcher(path string, ch chan<- FileEvent) {
	var lastModTime time.Time
	for {
		time.Sleep(2 * time.Second)
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if !info.ModTime().Equal(lastModTime) {
			if !lastModTime.IsZero() {
				select {
				case ch <- FileEvent{Type: 2, Path: path}:
				default:
				}
			}
			lastModTime = info.ModTime()
		}
		if info.IsDir() {
			entries, err := os.ReadDir(path)
			if err != nil {
				continue
			}
			for _, e := range entries {
				childPath := filepath.Join(path, e.Name())
				childInfo, err := os.Stat(childPath)
				if err != nil {
					continue
				}
				if !childInfo.ModTime().Equal(lastModTime) {
					select {
					case ch <- FileEvent{Type: 2, Path: childPath}:
					default:
					}
				}
			}
		}
	}
}

func fileInfoToStat(info os.FileInfo) *Stat {
	mode := info.Mode()
	fileType := FileTypeFile
	if mode.IsDir() {
		fileType = FileTypeDirectory
	} else if mode&os.ModeSymlink != 0 {
		fileType = FileTypeSymbolicLink
	}
	return &Stat{
		Type:        fileType,
		Size:        info.Size(),
		Mtime:       info.ModTime().UnixMilli(),
		Permissions: int32(mode.Perm()),
	}
}

func fileTypeFromInfo(mode os.FileMode) FileType {
	switch {
	case mode&os.ModeSymlink != 0:
		return FileTypeSymbolicLink
	case mode.IsDir():
		return FileTypeDirectory
	default:
		return FileTypeFile
	}
}

func parseOpenMode(mode string) int {
	switch mode {
	case "r":
		return os.O_RDONLY
	case "w":
		return os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	case "a":
		return os.O_WRONLY | os.O_CREATE | os.O_APPEND
	case "rw":
		return os.O_RDWR | os.O_CREATE
	default:
		return os.O_RDONLY
	}
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func init() {
	_ = strings.Contains
	_ = binary.LittleEndian
	_ = syscall.Statfs_t{}
}