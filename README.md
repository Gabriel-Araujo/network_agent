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

## Build from source
just exec the go build command

```bash
go build
```

