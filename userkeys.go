/*
SSH Key Server - A lightweight HTTP server that manages SSH public keys
Copyright (C) 2024 elsitar

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	rfsnotify "github.com/elsitar/ssh-keyserver/utils"
	"github.com/fsnotify/fsnotify"
	"golang.org/x/crypto/ssh"
)

type UserKeys struct {
	keyring     map[string][]string // username -> array of public keys
	keyringPath string
	keyringLock sync.RWMutex
}

func NewUserKeys(keyringPath string) (*UserKeys, error) {
	uk := &UserKeys{
		keyring:     make(map[string][]string),
		keyringPath: keyringPath,
	}

	// Load initial keys
	if err := uk.loadAllKeys(); err != nil {
		return nil, err
	}

	// Start watching the keyring directory
	if err := uk.watchKeyring(); err != nil {
		return nil, err
	}

	return uk, nil
}

func (uk *UserKeys) watchKeyring() error {
	watcher, err := rfsnotify.NewWatcher()
	if err != nil {
		return err
	}

	go func() {
		var (
			debounceTimer    *time.Timer
			debounceInterval = 1 * time.Second
			pendingReload    bool
			mu               sync.Mutex
		)

		reload := func() {
			mu.Lock()
			defer mu.Unlock()

			if !pendingReload {
				return
			}
			pendingReload = false

			if err := uk.loadAllKeys(); err != nil {
				log.Printf("Error reloading keyring: %v", err)
			} else {
				log.Printf("Keyring reloaded successfully")
			}
		}

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if event.Has(fsnotify.Create) || event.Has(fsnotify.Write) || event.Has(fsnotify.Remove) {
					mu.Lock()
					pendingReload = true
					if debounceTimer != nil {
						debounceTimer.Stop()
					}
					debounceTimer = time.AfterFunc(debounceInterval, reload)
					mu.Unlock()
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("Keyring watcher error: %v", err)
			}
		}
	}()

	return watcher.AddRecursive(uk.keyringPath)
}

func (uk *UserKeys) loadAllKeys() error {
	newKeyring := make(map[string][]string)

	entries, err := os.ReadDir(uk.keyringPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		username := entry.Name()
		keys, err := uk.loadUserKeys(username)
		if err != nil {
			log.Printf("Error loading keys for user %s: %v", username, err)
			continue
		}
		if len(keys) > 0 {
			newKeyring[username] = keys
		}
	}

	uk.keyringLock.Lock()
	uk.keyring = newKeyring
	uk.keyringLock.Unlock()

	log.Printf("Loaded keys for %d users", len(newKeyring))
	return nil
}

func (uk *UserKeys) loadUserKeys(username string) ([]string, error) {
	var keys []string
	userKeyDir := filepath.Join(uk.keyringPath, username)

	files, err := os.ReadDir(userKeyDir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".pub") {
			continue
		}

		keyPath := filepath.Join(userKeyDir, file.Name())
		keyData, err := os.ReadFile(keyPath)
		if err != nil {
			log.Printf("Error reading key file %s: %v", keyPath, err)
			continue
		}

		// Validate the key
		_, _, _, _, err = ssh.ParseAuthorizedKey(keyData)
		if err != nil {
			log.Printf("Invalid key found in %s", keyPath)
			continue
		}

		keyStr := string(keyData)
		if !strings.HasSuffix(keyStr, "\n") {
			keyStr += "\n"
		}
		keys = append(keys, keyStr)
	}

	return keys, nil
}

func (uk *UserKeys) GetUserKeys(username string) []string {
	uk.keyringLock.RLock()
	defer uk.keyringLock.RUnlock()
	return uk.keyring[username]
}
