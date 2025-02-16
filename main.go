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
	"net/http"
	"os"
)

func main() {
	configPath := os.Getenv("KEYSERVER_CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	keyrinPath := os.Getenv("KEYSERVER_KEYRING_PATH")
	if keyrinPath == "" {
		keyrinPath = "keyring"
	}

	server, err := NewServer(configPath, keyrinPath)
	if err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/keys/", server.getKeysHandler)

	port := os.Getenv("KEYSERVER_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
