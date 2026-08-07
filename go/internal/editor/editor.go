/*
editor implements the text editor model.

Replaces src/vs/editor/common/model/* (text model, buffer, document).
This is the in-memory representation of open text documents.
*/

package editor

import (
	"fmt"
	"sync"
	"time"
)

// TextDocument represents an open text document.
type TextDocument struct {
	mu       sync.RWMutex
	uri      string
	version  int
	content  []rune
	dirty    bool
	path     string
	modified time.Time
}

func NewTextDocument(uri, path string) *TextDocument {
	return &TextDocument{
		uri:      uri,
		path:     path,
		version:  1,
		content:  []rune{},
	}
}

func (d *TextDocument) URI() string {
	return d.uri
}

func (d *TextDocument) Path() string {
	return d.path
}

func (d *TextDocument) Version() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.version
}

func (d *TextDocument) Text() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return string(d.content)
}

func (d *TextDocument) SetText(text string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.content = []rune(text)
	d.version++
	d.dirty = true
	d.modified = time.Now()
}

func (d *TextDocument) IsDirty() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.dirty
}

func (d *TextDocument) Save() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	// TODO: write to disk
	d.dirty = false
	return nil
}

// TextEditorManager manages open text documents.
type TextEditorManager struct {
	mu    sync.RWMutex
	docs  map[string]*TextDocument
}

func NewTextEditorManager() *TextEditorManager {
	return &TextEditorManager{
		docs: make(map[string]*TextDocument),
	}
}

func (m *TextEditorManager) OpenDocument(uri, path string) (*TextDocument, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if doc, ok := m.docs[uri]; ok {
		return doc, nil
	}

	doc := NewTextDocument(uri, path)
	m.docs[uri] = doc
	return doc, nil
}

func (m *TextEditorManager) CloseDocument(uri string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.docs, uri)
}

func (m *TextEditorManager) GetDocument(uri string) *TextDocument {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.docs[uri]
}

func (m *TextEditorManager) AllDocuments() []*TextDocument {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*TextDocument, 0, len(m.docs))
	for _, doc := range m.docs {
		result = append(result, doc)
	}
	return result
}

// ActiveEditor tracks the currently active text editor.
type ActiveEditor struct {
	mu   sync.RWMutex
	doc  *TextDocument
}

func NewActiveEditor() *ActiveEditor {
	return &ActiveEditor{}
}

func (a *ActiveEditor) SetDocument(doc *TextDocument) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.doc = doc
}

func (a *ActiveEditor) Document() *TextDocument {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.doc
}

func (a *ActiveEditor) ShowMessage(msg string) {
	fmt.Println("[editor]", msg)
}
