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

Or alternatively, you can use the docker-compose file in the docker folder (.agentcontainer) to set up a docker container with postgres and pgvector using the following command:

```bash
docker compose -f ./.agentcontainer/docker-compose.yml up -d 
```

Or run the Dockerfile:
```bash
docker build --tag network-agent --file Dockerfile .
docker run -p 5433:5432 --name network-agent --detach network-agent 
```
#### Observations

- check the [.agentcontainer/.env](.agentcontainer/.env) file for database configuration params.
- in the container the database is reachable at localhost:5433.

## Build from source
just exec the go build command

```bash
go build
```

