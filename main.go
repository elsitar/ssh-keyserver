package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Hosts  map[string]HostConfig  `yaml:"hosts"`
	Groups map[string]GroupConfig `yaml:"groups"`
}

type HostConfig struct {
	Token  string   `yaml:"token"`
	Users  []string `yaml:"users"`
	Groups []string `yaml:"groups"`
}

type GroupConfig struct {
	Users []string `yaml:"users"`
}

type Server struct {
	config     Config
	configLock sync.RWMutex
	configPath string
}

func NewServer(configPath string) (*Server, error) {
	s := &Server{
		configPath: configPath,
	}

	if err := s.loadConfig(); err != nil {
		return nil, err
	}

	// Setup config file watcher
	if err := s.watchConfig(); err != nil {
		return nil, fmt.Errorf("failed to setup config watcher: %v", err)
	}

	return s, nil
}

func (s *Server) loadConfig() error {
	data, err := os.ReadFile(s.configPath)
	if err != nil {
		return fmt.Errorf("error reading config file: %v", err)
	}

	var newConfig Config
	if err := yaml.Unmarshal(data, &newConfig); err != nil {
		return fmt.Errorf("error parsing config file: %v", err)
	}

	s.configLock.Lock()
	s.config = newConfig
	s.configLock.Unlock()

	log.Printf("Config loaded successfully from %s", s.configPath)
	return nil
}

func (s *Server) watchConfig() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	go func() {
		// Use a timer to debounce rapid file changes
		var debounceTimer *time.Timer
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					if debounceTimer != nil {
						debounceTimer.Stop()
					}
					debounceTimer = time.AfterFunc(100*time.Millisecond, func() {
						if err := s.loadConfig(); err != nil {
							log.Printf("Error reloading config: %v", err)
						} else {
							log.Printf("Config reloaded successfully")
						}
					})
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("Config watcher error: %v", err)
			}
		}
	}()

	return watcher.Add(s.configPath)
}

func (s *Server) validateToken(hostname, token string) bool {
	s.configLock.RLock()
	defer s.configLock.RUnlock()

	hostConfig, exists := s.config.Hosts[hostname]
	if !exists {
		return false
	}
	return hostConfig.Token == token
}

func (s *Server) getUsersForHost(hostname string) []string {
	s.configLock.RLock()
	defer s.configLock.RUnlock()

	hostConfig, exists := s.config.Hosts[hostname]
	if !exists {
		return nil
	}

	uniqueUsers := make(map[string]bool)

	// Add direct users
	for _, user := range hostConfig.Users {
		uniqueUsers[user] = true
	}

	// Add users from groups
	for _, groupName := range hostConfig.Groups {
		if groupConfig, exists := s.config.Groups[groupName]; exists {
			for _, user := range groupConfig.Users {
				uniqueUsers[user] = true
			}
		}
	}

	users := make([]string, 0, len(uniqueUsers))
	for user := range uniqueUsers {
		_, err := os.ReadDir(filepath.Join("keyring", user))
		if err != nil {
			log.Printf("Error reading keyring for user %s: %v", user, err)
			continue
		}
		users = append(users, user)
	}
	log.Printf("Found %d users for %s: %v", len(users), hostname, users)
	return users
}

func (s *Server) getKeysForUsers(users []string) string {
	s.configLock.RLock()
	defer s.configLock.RUnlock()

	var keys strings.Builder
	for _, username := range users {
		userKeyDir := filepath.Join("keyring", username)
		files, err := os.ReadDir(userKeyDir)
		if err != nil {
			continue
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

			// Check if key is valid
			_, _, _, _, err = ssh.ParseAuthorizedKey(keyData)
			if err != nil {
				log.Printf("Invalid key found in %s", keyPath)
				continue
			}
			keys.Write(keyData)
			if !strings.HasSuffix(string(keyData), "\n") {
				keys.WriteString("\n")
			}
		}
	}

	return keys.String()
}

func (s *Server) getKeysHandler(w http.ResponseWriter, r *http.Request) {
	// Check HTTP method
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract hostname from path
	path := strings.TrimPrefix(r.URL.Path, "/keys/")
	if path == "" {
		http.Error(w, "Missing hostname", http.StatusBadRequest)
		return
	}
	hostname := path

	// Validate Hostname
	_, exists := s.config.Hosts[hostname]
	if !exists {
		http.Error(w, "Host not found", http.StatusNotFound)
		return
	}

	// Validate Authorization header
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Token ") {
		http.Error(w, "Invalid Authorization header", http.StatusUnauthorized)
		return
	}
	token := strings.TrimPrefix(authHeader, "Token ")

	// Validate Authorization token
	if !s.validateToken(hostname, token) {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Get list of authorized users for this host
	users := s.getUsersForHost(hostname)
	if len(users) == 0 {
		http.Error(w, "Host has no valid users", http.StatusNotFound)
		return
	}

	// Collect all public keys for authorized users
	keys := s.getKeysForUsers(users)
	if len(strings.Split(keys, "\n")) <= 1 {
		http.Error(w, "Host has no valid keys", http.StatusNotFound)
		return
	}

	log.Printf("Serving %d keys for %s and users %s", len(strings.Split(keys, "\n")), hostname, users)
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, keys)
}

func main() {
	server, err := NewServer("config.yaml")
	if err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}

	// Use default ServeMux
	mux := http.NewServeMux()
	mux.HandleFunc("/keys/", server.getKeysHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
