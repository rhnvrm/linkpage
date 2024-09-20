package main

import (
	"bytes"
	"io"
	"sync"
	"text/template"
)

type Templates struct {
	Home  *cachedTemplate
	Admin *template.Template
}

type cachedTemplate struct {
	*template.Template
	rawData []byte
	sync.RWMutex
}

func (ct *cachedTemplate) Save(data Page) error {
	var out = bytes.NewBuffer([]byte{})
	if err := ct.Execute(out, data); err != nil {
		return err
	}

	ct.Lock()
	ct.rawData = out.Bytes()
	ct.Unlock()
	return nil
}

func (ct *cachedTemplate) Write(w io.Writer) error {
	ct.RLock()
	defer ct.RUnlock()

	_, err := io.Copy(w, bytes.NewBuffer(ct.rawData))
	if err != nil {
		return err
	}

	return nil
}
