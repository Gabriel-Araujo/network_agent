## Chucker

O `chucker.exe` divide um documento FRRouting em chunks JSON para ingestão no
retriever RAG.

### Uso

No PowerShell, a partir da raiz do projeto:

```powershell
.\chucker.exe -path ".\resources\frrouting\docs\bgp.rst" -protocol bgp -daemon bgpd
```

Parâmetros obrigatórios:

- `-path`: caminho do arquivo `.rst` de entrada;
- `-protocol`: protocolo documentado, por exemplo `bgp`;
- `-daemon`: daemon FRR correspondente, por exemplo `bgpd`.

O comando grava o resultado em:

```text
resources\frrouting\docs\chunks\bgp.json
```

Para outro documento, altere os três parâmetros. Por exemplo:

```powershell
.\chucker.exe -path ".\resources\frrouting\docs\ospf.rst" -protocol ospf -daemon ospfd
```