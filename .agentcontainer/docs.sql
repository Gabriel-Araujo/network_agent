CREATE TABLE documents (
                            id          BIGSERIAL PRIMARY KEY,
                            question    TEXT NOT NULL,            -- chunk original
                            answer      TEXT NOT NULL,            -- chunk original
                            metadata    JSONB,                    -- fonte, página, data etc.
                            embedding   VECTOR(4096)             -- dimensão depende do modelo

);

CREATE INDEX ON documents USING diskann (embedding);

CREATE TABLE IF NOT EXISTS frr_docs (
    id              BIGSERIAL PRIMARY KEY,
    chunk_id        TEXT UNIQUE NOT NULL,          -- ex: "bgp:cmd:bgp-router-id-a-b-c-d"
    chunk_type      TEXT NOT NULL
                    CHECK (chunk_type IN ('command_reference', 'concept')),
    daemon          TEXT NOT NULL,                 -- bgpd | ospfd | isisd | zebra | ...
    protocol        TEXT NOT NULL,                 -- bgp | ospf | isis | static | ...
    section_path    TEXT NOT NULL,                 -- "BGP > BGP Router Configuration > ..."
    command         TEXT,                          -- só em command_reference
    content         TEXT NOT NULL,                  -- texto que vira embedding
    parent_content  TEXT NOT NULL,                  -- seção completa, pra expandir contexto
    source_url      TEXT NOT NULL,
    token_count     INT NOT NULL,
    embedding       VECTOR(4096) NOT NULL,
    content_tsv     TSVECTOR GENERATED ALWAYS AS (to_tsvector('english', content)) STORED
);

CREATE INDEX IF NOT EXISTS frr_docs_diskann_idx
    ON frr_docs USING diskann (embedding vector_cosine_ops);

CREATE INDEX IF NOT EXISTS frr_docs_tsv_idx
    ON frr_docs USING gin (content_tsv);

CREATE INDEX IF NOT EXISTS frr_docs_filter_idx
    ON frr_docs (daemon, protocol, chunk_type);
