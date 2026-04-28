package i18n

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"path"
	"sync"
	"text/template"
)

var (
	messagesMu sync.RWMutex
	messages   = map[string]string{}
)

func RegisterLocales(fsys fs.FS, dir string) {
	if fsys == nil {
		return
	}
	if dir == "" {
		dir = "."
	}
	_ = fs.WalkDir(fsys, dir, func(file string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || path.Ext(file) != ".json" {
			return nil
		}
		data, err := fs.ReadFile(fsys, file)
		if err != nil {
			return nil
		}
		var decoded map[string]any
		if err := json.Unmarshal(data, &decoded); err != nil {
			return nil
		}
		flat := map[string]string{}
		flatten("", decoded, flat)

		messagesMu.Lock()
		for key, value := range flat {
			messages[key] = value
		}
		messagesMu.Unlock()
		return nil
	})
}

func T(messageID string, args ...any) string {
	messagesMu.RLock()
	message, ok := messages[messageID]
	messagesMu.RUnlock()
	if !ok {
		return messageID
	}
	if len(args) == 0 {
		return message
	}
	tmpl, err := template.New(messageID).Parse(message)
	if err != nil {
		return message
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, args[0]); err != nil {
		return message
	}
	return out.String()
}

func Label(word string) string {
	key := "common.label." + word
	value := T(key)
	if value == key {
		return word
	}
	return value
}

func flatten(prefix string, value any, out map[string]string) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			next := key
			if prefix != "" {
				next = prefix + "." + key
			}
			flatten(next, child, out)
		}
	case string:
		out[prefix] = typed
	}
}
