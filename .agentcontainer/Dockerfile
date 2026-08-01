FROM docker.io/timescale/timescaledb-ha:pg18

COPY init-extensions.sql /docker-entrypoint-initdb.d/001-extensions.sql
COPY docs.sql /docker-entrypoint-initdb.d/002-docs.sql

EXPOSE 5432
