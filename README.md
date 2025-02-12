# SSH Key Server

A lightweight HTTP server that manages SSH public keys for multiple hosts and users. It provides a centralized way to distribute SSH public keys to authorized hosts, with support for user groups and host-specific access tokens.

## Features

- HTTP API for retrieving authorized SSH keys
- Host-based authentication using tokens
- User grouping support
- Hot-reload of configuration
- File-based key storage
- Caching support for offline operation

## Directory Structure

```
ssh-keyserver/
├── keyring/           # Directory containing user SSH public keys
│   ├── alice/        # Each user has their own directory
│   │   └── id_rsa.pub
│   └── bob/
│       └── id_rsa.pub
├── config.yaml       # Server configuration file
├── main.go          # Server implementation
└── fetch-authorized-keys.sh  # Client script for fetching keys
```

## Configuration

Create a `config.yaml` file with your hosts and groups:

```yaml
hosts:
  webserver1:
    token: "foo"
    users: ["alice", "bob"]
    groups: ["devops"]
  dbserver1:
    token: "bar"
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

## Client Setup

1. Copy the `fetch-authorized-keys.sh` script to your client machines
2. Configure your sshd to use the script

Add to `/etc/ssh/sshd_config`:
```
AuthorizedKeysCommand /usr/local/bin/fetch-authorized-keys.sh
AuthorizedKeysCommandUser nobody
```

3. Restart sshd:
```bash
sudo systemctl restart sshd
```

## API Usage

Get keys for a specific host:
```bash
curl -H "Authorization: Token foo" http://localhost:8080/keys/webserver1
```

## Security Considerations

- Use HTTPS in production
- Keep tokens secure
- Place the server behind a firewall
- Regularly audit the keyring directory
- Monitor access logs

## Environment Variables

- `PORT`: Server port (default: 8080)

## License

MIT