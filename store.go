// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"io/ioutil"
	"os"
	"sync"

	"cloud.google.com/go/datastore"
)

type store interface {
	PutSnippet(ctx context.Context, id string, snip *snippet) error
	GetSnippet(ctx context.Context, id string, snip *snippet) error
}

type cloudDatastore struct {
	client *datastore.Client
}

func (s cloudDatastore) PutSnippet(ctx context.Context, id string, snip *snippet) error {
	key := datastore.NameKey("Snippet", id, nil)
	_, err := s.client.Put(ctx, key, snip)
	return err
}

func (s cloudDatastore) GetSnippet(ctx context.Context, id string, snip *snippet) error {
	key := datastore.NameKey("Snippet", id, nil)
	return s.client.Get(ctx, key, snip)
}

// inMemStore is a store backed by a map that should only be used for testing.
type inMemStore struct {
	sync.RWMutex
	m map[string]*snippet // key -> snippet
}

func (s *inMemStore) PutSnippet(_ context.Context, id string, snip *snippet) error {
	s.Lock()
	if s.m == nil {
		s.m = map[string]*snippet{}
	}
	b := make([]byte, len(snip.Body))
	copy(b, snip.Body)
	s.m[id] = &snippet{Body: b}
	s.Unlock()
	return nil
}

func (s *inMemStore) GetSnippet(_ context.Context, id string, snip *snippet) error {
	s.RLock()
	defer s.RUnlock()
	v, ok := s.m[id]
	if !ok {
		return datastore.ErrNoSuchEntity
	}
	*snip = *v
	return nil
}

// filesystemStore is a store backed by a map that should only be used for filesystem.
// add by polairs, at 2020-03-04
type filesystemStore struct {
	storePath string
	sync.RWMutex
	m map[string]*snippet // key -> snippet
}

func newFilesystemStore() *filesystemStore {
	storePath := "./snippet"
	if err := os.MkdirAll(storePath, 0755); err != nil {
		panic(err)
	}

	return &filesystemStore{storePath: storePath}
}

func (f *filesystemStore) PutSnippet(_ context.Context, id string, snip *snippet) error {
	filePath := f.storePath + "/" + id[:2]
	os.Mkdir(filePath, 0755)
	file, err := os.Create(filePath + "/" + id)
	if err != nil {
		log.Errorf("create snip file error:%#v", err)
		return err
	}
	defer file.Close()
	file.Write(snip.Body)

	return nil
}

func (f *filesystemStore) GetSnippet(_ context.Context, id string, snip *snippet) error {
	body, err := ioutil.ReadFile(f.storePath + "/" + id[:2] + "/" + id)
	if err != nil {
		return datastore.ErrNoSuchEntity
	}

	snip.Body = body

	return nil
}
