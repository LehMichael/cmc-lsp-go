package workspace

import (
	"net/url"
	"os"
	"path/filepath"
	"sync"
)

type Overlay struct {
	mu   sync.RWMutex
	docs map[string]Document
}

type Document struct {
	contents string
	version  int
}

func NewOverlay() *Overlay {
	return &Overlay{
		docs: map[string]Document{},
	}
}

func (o *Overlay) Open(uri string, text string, version int) error {
	path, err := uriToPath(uri)
	if err != nil {
		return err
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	o.docs[path] = Document{
		contents: text,
		version:  version,
	}

	return nil
}

func (o *Overlay) Update(uri string, text string, version int) error {
	path, err := uriToPath(uri)
	if err != nil {
		return err
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	o.docs[path] = Document{
		contents: text,
		version:  version,
	}

	return nil
}

func (o *Overlay) Close(uri string) error {
	path, err := uriToPath(uri)
	if err != nil {
		return err
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	delete(o.docs, path)

	return nil
}

func (o *Overlay) Read(path string) (string, error) {
	o.mu.RLock()

	if d, ok := o.docs[path]; ok {
		o.mu.RUnlock()
		return d.contents, nil
	}

	o.mu.RUnlock()

	if d, err := os.ReadFile(path); err != nil {
		return "", err
	} else {
		return string(d), nil
	}
}

func uriToPath(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	p := u.Path // decoded: %20 -> space, etc.

	// Windows: u.Path is "/C:/Users/foo" — strip the leading slash
	if len(p) >= 3 && p[0] == '/' && p[2] == ':' {
		p = p[1:]
	}
	p = filepath.Clean(p)

	// On case-insensitive filesystems (Windows, default macOS), fold case
	// so c:\foo and C:\Foo compare equal. On Linux, do NOT.
	// Easiest portable hack: keep a separate compare-key, e.g. strings.ToLower(p),
	// but store the original-cased path for actual os.ReadFile.
	return p, nil
}
