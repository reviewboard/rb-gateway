# rb-gateway

A repository API server for [Review Board](https://www.reviewboard.org/).
rb-gateway exposes a REST API for repository operations (branches, commits,
file contents) and manages webhooks for SCM push events. It supports Git and
Mercurial repositories.


## Requirements

- [Go](https://go.dev/) 1.22 or newer
- Git (for serving Git repositories)
- Mercurial (for serving Mercurial repositories)


## Building

```sh
$ git clone https://github.com/reviewboard/rb-gateway.git
$ cd rb-gateway
$ make build
```

Or build directly with `go`:

```sh
$ go build -ldflags "-X main.version=$(cat VERSION)"
```


## Configuration

Copy the sample configuration and edit it:

```sh
$ cp sample_config.json config.json
```

The configuration file is JSON with the following fields:

| Field              | Required | Default          | Description                                               |
|--------------------|----------|------------------|-----------------------------------------------------------|
| `port`             | No       | `8888`           | Port the server listens on.                               |
| `repositories`     | Yes      |                  | Array of repository definitions.                          |
| `htpasswdPath`     | No       | `htpasswd`       | Path to the htpasswd file for basic auth.                 |
| `tokenStorePath`   | No       | `tokens.dat`     | Path to the API token store.                              |
| `webhookStorePath` | No       | `webhooks.json`  | Path to the webhook store.                                |
| `useTLS`           | No       | `false`          | Enable TLS.                                               |
| `sslCertificate`   | No       |                  | Path to TLS certificate (required if `useTLS` is `true`). |
| `sslKey`           | No       |                  | Path to TLS private key (required if `useTLS` is `true`). |

Each repository in the `repositories` array has:

| Field  | Description                            |
|--------|----------------------------------------|
| `name` | A unique name used in the API URL.     |
| `path` | Absolute path to the repository.       |
| `scm`  | Repository type: `"git"` or `"hg"`.    |

Relative paths in the configuration are resolved relative to the directory
containing the configuration file.

Example (`sample_config.json`):

```json
{
    "htpasswdPath": "htpasswd",
    "port": 8888,
    "tokenStorePath": "tokens.dat",
    "webhookStorePath": "webhooks.json",
    "repositories": [
        {"name": "repo1", "path": "/path/to/repo1.git", "scm": "git"},
        {"name": "repo2", "path": "/path/to/repo2.hg", "scm": "hg"}
    ]
}
```


## Authentication

rb-gateway uses an [htpasswd](https://httpd.apache.org/docs/current/programs/htpasswd.html)
file for basic authentication. Passwords may be stored as bcrypt hashes
(recommended) or in plain text.

Create an htpasswd file with the `htpasswd` utility:

```sh
$ htpasswd -Bc htpasswd myuser
```


## Running

Start the server:

```sh
$ ./rb-gateway serve
```

By default, this reads `config.json` from the current directory. Use
`--config` to specify a different path:

```sh
$ ./rb-gateway --config /etc/rb-gateway/config.json serve
```

The server reloads its configuration automatically when the config file
changes or when it receives `SIGHUP`. Send `SIGINT` or `SIGTERM` to shut
down gracefully.


### Commands

| Command              | Description                                           |
|----------------------|-------------------------------------------------------|
| `serve`              | Start the API server (default).                       |
| `trigger-webhooks`   | Trigger matching webhooks for a repository and event. |
| `reinstall-hooks`    | Re-install hook scripts for all repositories.         |

```sh
$ ./rb-gateway --version          # Print version
$ ./rb-gateway trigger-webhooks <repository> <event>
$ ./rb-gateway reinstall-hooks
```


## Testing

Run unit tests:

```sh
$ go test ./...
```

Or using the Makefile (uses [gotestsum](https://github.com/gotestyourself/gotestsum)
for nicer output):

```sh
$ make test
```

### Integration tests

Integration tests are gated behind the `integration` build tag and require
a built `rb-gateway` binary:

```sh
$ make integration-tests
```

This builds the binary, sets `RBGATEWAY_PATH`, and runs the integration
tests with `-tags integration`.


## License

MIT License. See [LICENSE](LICENSE) for details.
