# SSH Key Server

A lightweight HTTP server that manages SSH public keys for multiple hosts and users. It provides a centralized way to distribute SSH public keys to authorized hosts, with support for user groups and host-specific access tokens.

## Features

- HTTP API for retrieving authorized SSH keys
- Host-based authentication using tokens
- User grouping support
- Live configuration reloading via file watching
- File-based public key storage with automatic reload
- Memory-efficient key caching
- Docker support

## Directory Structure

```
ssh-keyserver/
├── keyring/             # Directory containing user SSH public keys
│   ├── alice/           # Each user has their own directory
│   │   ├── id_rsa.pub   # Users can have multiple key files
│   │   └── id_ed25519.pub
│   └── bob/
│       └── id_rsa.pub
└── config.yaml          # Server configuration file
```

## Configuration

The `config.yaml` file supports hosts and groups:

```yaml
hosts:
  webserver1:
    token: "secret-token-1"
    users: ["alice", "bob"]    # Direct user assignments
    groups: ["devops"]         # Group assignments
  dbserver1:
    token: "secret-token-2"
    users: ["carol"]
    groups: ["dba"]

groups:
  devops:
    users: ["dave", "eve"]
  dba:
    users: ["frank", "grace"]
```

## Setup

1. Create the keyring directory structure:
```bash
mkdir -p keyring/{alice,bob,carol,dave,eve,frank,grace}
```

2. Add public keys to each user's directory:
```bash
cp /path/to/alice/id_rsa.pub keyring/alice/
```

3. Build and run the server:
```bash
go build
./ssh-keyserver
```

## Environment Variables

- `KEYSERVER_CONFIG_PATH`: Path to config.yaml (default: "config.yaml")
- `KEYSERVER_KEYRING_PATH`: Path to keyring directory (default: "keyring")
- `KEYSERVER_PORT`: Server port (default: "8080")

## Running with Docker

Build and run using Docker:

```bash
docker build -t ssh-keyserver .
docker run -d \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  -v $(pwd)/keyring:/app/keyring:ro \
  ssh-keyserver
```

## API Usage

Retrieve authorized keys for a host:
```bash
curl -H "Authorization: Token secret-token-1" http://localhost:8080/keys/webserver1
```

The server responds with the concatenated SSH public keys of all authorized users.

## Security Considerations

- Use HTTPS in production
- Store tokens securely and rotate them regularly
- Place the server behind a reverse proxy
- Mount config and keyring as read-only in Docker
- Monitor access logs for unauthorized attempts
- Regularly audit the keyring directory and user access
- Run the server as a non-root user

## Client Integration

1. Copy the provided `fetch-authorized-keys.sh` script to your client machines:
```bash
sudo cp fetch-authorized-keys.sh /usr/local/bin/
sudo chmod +x /usr/local/bin/fetch-authorized-keys.sh
```

2. Configure the script by editing these variables:
```bash
CACHE_DIR="/var/cache/ssh-keys"     # Local cache directory
SERVER_URL="http://keyserver:8080"  # Your keyserver URL
AUTH_TOKEN="secret-token-1"         # Host-specific token
```

3. Add to your sshd configuration (`/etc/ssh/sshd_config`):
```
AuthorizedKeysCommand /usr/local/bin/fetch-authorized-keys.sh
AuthorizedKeysCommandUser nobody
```

4. Create the cache directory:
```bash
sudo mkdir -p /var/cache/ssh-keys
sudo chown nobody:nobody /var/cache/ssh-keys
```

5. Restart the SSH daemon:
```bash
sudo systemctl restart sshd
```

The script includes automatic caching for offline operation. If the key server is unreachable, it will fall back to the last successfully retrieved keys.

## License

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.