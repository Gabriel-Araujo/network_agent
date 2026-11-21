# Network Agent

# Usage
The first running the agent will ask for a llm setup.

## Required
to run the agent, you must have:
- An OpenAI compatible server running.
- A postgres database running with pgvector installed.

## RAG Set up
To use RAG, you must set up postgres and pgvector. Postgres can be installed using the instructions for your system in [here](https://www.postgresql.org/download/),
and pgvector can be installed using the instructions in [here](https://github.com/pgvector/pgvector#installation-notes---linux-and-mac).

### Using Docker

Alternatively, use the Compose file in `.dockercontainer` to start PostgreSQL with
pgvector:

```bash
docker compose -f ./.dockercontainer/docker-compose.yml up -d
```

#### Observations

- Check the [.dockercontainer/.env](.dockercontainer/.env) file for database
  configuration parameters.
- A process running on the host connects through the published port with
  `localhost:5433`.
- A process running on the Compose network connects to the database service at
  `db:5432`. Inside a container, `localhost` refers to that container itself,
  not to the `db` service.
- Verify the published port with
  `docker compose -f ./.dockercontainer/docker-compose.yml port db 5432`.

#### Windows with Podman

- Start the Podman machine before using Compose:

  ```bash
  podman machine start
  ```

  If GoLand reports `connect ENOENT \\.\pipe\podman-machine-default`, the
  machine is stopped or unavailable. Start it, then retry the Compose command.
- If the image build fails with
  `lookup registry-1.docker.io: Temporary failure in name resolution`, the
  Podman machine cannot resolve Docker Hub. Check its resolver with
  `podman machine ssh` and `cat /etc/resolv.conf`; configure a nameserver
  reachable from your network or restart the machine, then retry the build.
  Compose `dns:` settings do not affect DNS used while pulling the base image.
- Start the stack with:

  ```bash
  docker compose -f ./.dockercontainer/docker-compose.yml up -d --build
  ```

  Podman may print a message that it is using the external Windows
  `docker-compose.exe` provider; that message is informational.
- Verify the published port with
  `docker compose -f ./.dockercontainer/docker-compose.yml port db 5432`; it
  should report host port `5433`. Host tools connect to `localhost:5433`, while
  containers on the Compose network connect to `db:5432`. `localhost:5433` is
  not the way to reach the host from inside the database container.

## Build from source
just exec the go build command

```bash
go build ./cmd/cli
```

To build de GUI

```bash
cd cmd/ui
wails3 build
```
