## Ingester

O `ingester.exe` carrega os chunks JSON gerados pelo `chucker.exe` e os
persiste na tabela `frr_docs` do PostgreSQL para uso pelo retriever RAG.

### Uso

No PowerShell, a partir da raiz do projeto:

```powershell
.\ingester.exe -dir ".\resources\frrouting\docs\chunks"
```

Parâmetros:

- `-dir`: diretório com os arquivos JSON de chunks; obrigatório;
- `-dsn`: string de conexão PostgreSQL; opcional, com padrão
  `postgres://postgres:postgres@localhost:5433/frr_rag`.

Quando o `ingester.exe` é executado no host e o Compose está ativo, o padrão
usa a porta publicada `localhost:5433`, que é encaminhada para a porta interna
`5432` do serviço `db`.

Se o processo for executado dentro da rede do Compose, use o nome do serviço e
a porta interna:

```text
postgres://postgres:postgres@db:5432/frr_rag
```

Nesse contexto, não use `localhost:5433`: `localhost` aponta para o próprio
container e `5433` é uma porta publicada no host, não a porta do PostgreSQL na
rede interna.

Para informar outro banco:

```powershell
.\ingester.exe -dsn "postgres://usuario:senha@localhost:5432/frr_rag" -dir ".\resources\frrouting\docs\chunks"
```

Uma DSN explícita em `-dsn` substitui o padrão e permite usar uma instalação
PostgreSQL local ou outra porta publicada. O PostgreSQL deve estar disponível
antes da execução. O programa lê todos os arquivos do diretório informado e
faz upsert dos chunks na tabela `frr_docs`.