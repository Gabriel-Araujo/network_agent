# BGP e FRRouting: guia prático

## Sumário

1. [O que é BGP](#1-o-que-é-bgp)
2. [Conceitos fundamentais](#2-conceitos-fundamentais)
3. [Como o BGP funciona na prática](#3-como-o-bgp-funciona-na-prática)
4. [Atributos e seleção de melhor rota](#4-atributos-e-seleção-de-melhor-rota)
5. [FRRouting: visão geral](#5-frrouting-visão-geral)
6. [Configurando BGP no FRR](#6-configurando-bgp-no-frr)
7. [Políticas de roteamento](#7-políticas-de-roteamento)
8. [Cenários comuns](#8-cenários-comuns)
9. [Operação e troubleshooting](#9-operação-e-troubleshooting)
10. [Boas práticas de segurança](#10-boas-práticas-de-segurança)
11. [FAQ](#11-faq)
12. [Referências](#12-referências)

---

## 1. O que é BGP

O **Border Gateway Protocol** (BGP, versão 4, definida na RFC 4271) é o protocolo que mantém a Internet funcionando. Enquanto protocolos como OSPF e IS-IS resolvem roteamento *dentro* de uma organização, o BGP resolve roteamento *entre* organizações: ele é o mecanismo pelo qual provedores, datacenters e empresas anunciam uns aos outros quais blocos de endereços IP conseguem alcançar.

Três características definem o BGP:

**É um protocolo de vetor de caminho (path vector).** Em vez de anunciar apenas um custo numérico até um destino, o BGP anuncia a lista completa de sistemas autônomos que a rota atravessa. Isso permite detectar loops de forma trivial — se um roteador vê o próprio número de AS na lista, descarta o anúncio — e dá visibilidade sobre por onde o tráfego passa.

**É orientado a política, não a métrica.** OSPF escolhe o caminho mais curto. BGP escolhe o caminho que você mandar escolher. A decisão leva em conta relações comerciais, contratos, custo de trânsito e preferências operacionais. Essa flexibilidade é o motivo de o protocolo existir na forma atual.

**Roda sobre TCP, na porta 179.** As sessões são explicitamente configuradas entre pares (não há descoberta automática de vizinhos como em OSPF), e o TCP cuida da entrega confiável e ordenada. Uma consequência importante: o BGP não descobre topologia sozinho, ele depende de você dizer com quem falar.

O BGP é lento e conservador por design. Ele não foi feito para convergir em milissegundos; foi feito para escalar a mais de um milhão de rotas na tabela global e para não propagar instabilidade mundo afora.

---

## 2. Conceitos fundamentais

### Sistema Autônomo (AS)

Um AS é um conjunto de redes sob uma mesma administração e política de roteamento. Cada AS tem um número (ASN) atribuído por um RIR — no Brasil, o NIC.br/LACNIC. ASNs de 16 bits vão de 1 a 65535; ASNs de 32 bits (RFC 6793) vão até 4294967295 e hoje são o padrão em novas alocações.

Faixas privadas, que não podem aparecer na tabela global:

- 64512–65534 (16 bits)
- 4200000000–4294967294 (32 bits)

O ASN 23456 (`AS_TRANS`) é reservado para compatibilidade entre roteadores que só entendem 16 bits e vizinhos com ASN de 32 bits.

### eBGP e iBGP

| | eBGP | iBGP |
|---|---|---|
| Entre | ASNs diferentes | Mesmo ASN |
| TTL padrão | 1 (vizinho diretamente conectado) | 255 |
| Next-hop | Reescrito para o IP local | Preservado por padrão |
| Propagação | Repassa rotas para todos os vizinhos | **Não** repassa rota aprendida de um iBGP para outro iBGP |
| AS-path | Prepend do próprio ASN | Inalterado |
| Local preference | Não é enviado entre ASes | Enviado entre pares iBGP |

A regra de não-propagação do iBGP (*split horizon* do iBGP) existe para evitar loops, já que o AS-path não muda dentro do AS. A consequência é que o iBGP exige **malha completa** (full mesh): com N roteadores, N(N-1)/2 sessões. Para escalar além de poucas dezenas de roteadores, usa-se **route reflectors** (RFC 4456) ou, mais raramente, confederações (RFC 5065).

### Tabelas

O BGP mantém a **Adj-RIB-In** (tudo que foi recebido de cada vizinho), a **Loc-RIB** (rotas após aplicar políticas de entrada e escolher a melhor) e a **Adj-RIB-Out** (o que é anunciado a cada vizinho após políticas de saída). Só as melhores rotas da Loc-RIB são instaladas na FIB do kernel e efetivamente usadas para encaminhar pacotes.

### Mensagens

- **OPEN** — negocia versão, ASN, hold time, router-id e capabilities (multiprotocolo, route refresh, ASN de 32 bits, graceful restart, add-path).
- **UPDATE** — anuncia prefixos (NLRI) com seus atributos ou retira prefixos (withdraw).
- **KEEPALIVE** — mantém a sessão viva; padrão a cada 60 s, com hold time de 180 s.
- **NOTIFICATION** — sinaliza erro e derruba a sessão.

### Máquina de estados

`Idle → Connect → Active → OpenSent → OpenConfirm → Established`

Só em `Established` há troca de rotas. Uma sessão que oscila entre `Connect` e `Active` normalmente indica que o TCP não está fechando: firewall, rota ausente para o vizinho, ASN errado ou IP de origem incorreto.

---

## 3. Como o BGP funciona na prática

O ciclo de vida de um anúncio:

1. Um roteador origina um prefixo (via `network`, redistribuição ou agregação) e o coloca na Loc-RIB com AS-path vazio.
2. Ao anunciar para um vizinho eBGP, prepende o próprio ASN no AS-path e define o next-hop como o próprio endereço da sessão.
3. O vizinho recebe, aplica a política de entrada (filtros, route-maps), e insere na Adj-RIB-In.
4. Se a rota passa nos filtros, entra na Loc-RIB e concorre com outras rotas para o mesmo prefixo pelo algoritmo de seleção.
5. A melhor rota é instalada na FIB e, se a política de saída permitir, repassada adiante com o AS-path acrescido.

Cada AS pelo qual a rota passa adiciona um ASN. Um AS-path como `65001 3356 15169` significa: para chegar nesse prefixo, o tráfego passará pelo AS 65001, depois pelo 3356, e o prefixo é originado pelo 15169.

### Relações comerciais

O modelo econômico da Internet se traduz diretamente em política BGP:

- **Trânsito** — você paga um provedor para alcançar toda a Internet. Recebe rota padrão ou tabela completa.
- **Peering** — troca gratuita de tráfego entre dois ASes, geralmente em um IX. Cada lado anuncia apenas suas próprias redes e as de seus clientes, nunca as de seus provedores.
- **Cliente** — quem paga você por trânsito. Rotas de cliente são anunciadas para todo mundo.

A regra fundamental é: **nunca anuncie rotas de um provedor ou peer para outro provedor ou peer**. Violar isso transforma sua rede em trânsito gratuito e é a causa clássica de vazamento de rotas (*route leak*).

---

## 4. Atributos e seleção de melhor rota

### Principais atributos

| Atributo | Tipo | Escopo | Uso típico |
|---|---|---|---|
| `AS_PATH` | Well-known mandatory | Global | Detecção de loop; influenciar tráfego de entrada via prepend |
| `NEXT_HOP` | Well-known mandatory | Global | Endereço para onde encaminhar |
| `ORIGIN` | Well-known mandatory | Global | IGP / EGP / Incomplete |
| `LOCAL_PREF` | Well-known discretionary | Dentro do AS | Escolher saída preferida (tráfego de **saída**) |
| `MED` | Optional non-transitive | Entre ASes adjacentes | Sugerir ponto de entrada preferido |
| `COMMUNITY` | Optional transitive | Configurável | Marcação e sinalização de política |
| `ATOMIC_AGGREGATE` / `AGGREGATOR` | — | Global | Indicam sumarização |

**Communities** merecem destaque. São etiquetas no formato `ASN:valor` que permitem marcar rotas e aplicar políticas de forma escalável. Existem as bem conhecidas — `no-export`, `no-advertise`, `local-AS`, `blackhole` (65535:666) — e as definidas por cada operadora. É comum um provedor publicar uma tabela de communities que o cliente pode usar para pedir prepend automático, alterar local-pref ou solicitar blackhole de um /32 sob ataque. Existem ainda *extended communities* (usadas em L3VPN e EVPN) e *large communities* (RFC 8092), necessárias quando o ASN é de 32 bits.

### Algoritmo de seleção

Na ordem, o BGP compara rotas para o mesmo prefixo e para na primeira diferença:

1. **Weight** maior (específico do FRR/Cisco, local ao roteador, não propagado)
2. **Local preference** maior
3. Rota **originada localmente**
4. **AS-path** mais curto
5. **Origin** menor (IGP < EGP < Incomplete)
6. **MED** menor (só entre rotas do mesmo AS vizinho, por padrão)
7. **eBGP** antes de iBGP
8. Menor custo IGP até o **next-hop**
9. Rota mais antiga (estabilidade em eBGP)
10. Menor **router-id** do vizinho
11. Menor **cluster-list**
12. Menor endereço IP do vizinho

Vale memorizar os quatro primeiros — a esmagadora maioria das decisões operacionais se resolve neles. Uma regra prática: **local preference controla o tráfego que sai** do seu AS, **AS-path prepend e MED tentam influenciar o tráfego que entra**. Controlar entrada é sempre mais difícil, porque a decisão final é do outro lado.

---

## 5. FRRouting: visão geral

O **FRRouting** (FRR) é uma suíte de roteamento livre para Linux e Unix, fork do Quagga (que por sua vez descende do Zebra), mantida sob a Linux Foundation. É o software de roteamento por trás de plataformas como Cumulus Linux, SONiC e VyOS, além de ser usado diretamente em servidores, roteadores de borda e no plano de controle de datacenters.

### Arquitetura

O FRR é modular: cada protocolo roda como um daemon separado, e todos conversam com o **zebra**, que é o gerenciador de RIB e o único que fala com o kernel (via netlink no Linux).

```
  bgpd    ospfd    isisd    staticd    bfdd    ...
    \       |        |        /         /
     \      |        |       /         /
      +-----+--------+------+---------+
                    |
                  zebra
                    |
              kernel FIB (netlink)
```

Daemons mais relevantes: `zebra` (obrigatório), `bgpd`, `ospfd`, `ospf6d`, `isisd`, `staticd`, `bfdd` (detecção rápida de falha), `pimd`, `vrrpd`, `watchfrr` (supervisão).

### Arquivos e comandos

| Caminho | Função |
|---|---|
| `/etc/frr/daemons` | Liga/desliga cada daemon e define suas flags |
| `/etc/frr/frr.conf` | Configuração integrada (modo padrão atual) |
| `/etc/frr/vtysh.conf` | Configuração do shell |
| `/var/log/frr/` | Logs |

A CLI é o **vtysh**, com sintaxe deliberadamente parecida com IOS:

```bash
sudo vtysh                          # entra no shell interativo
sudo vtysh -c "show bgp summary"    # comando único
sudo systemctl restart frr          # reinicia a suíte
```

Para habilitar o BGP, edite `/etc/frr/daemons`:

```
bgpd=yes
bgpd_options="  -A 127.0.0.1"
```

e reinicie o serviço.

---

## 6. Configurando BGP no FRR

### Exemplo mínimo: eBGP com um provedor

```
frr version 10.2
frr defaults traditional
hostname borda01
log syslog informational
service integrated-vtysh-config
!
router bgp 65001
 bgp router-id 10.0.0.1
 no bgp ebgp-requires-policy
 bgp log-neighbor-changes
 !
 neighbor 203.0.113.1 remote-as 64500
 neighbor 203.0.113.1 description ISP-A
 neighbor 203.0.113.1 password SenhaForte
 neighbor 203.0.113.1 timers 10 30
 !
 address-family ipv4 unicast
  network 198.51.100.0/24
  neighbor 203.0.113.1 activate
  neighbor 203.0.113.1 soft-reconfiguration inbound
  neighbor 203.0.113.1 route-map ISP-A-IN in
  neighbor 203.0.113.1 route-map ISP-A-OUT out
  neighbor 203.0.113.1 maximum-prefix 1000000 restart 5
 exit-address-family
exit
!
ip prefix-list MINHAS-REDES seq 5 permit 198.51.100.0/24
!
route-map ISP-A-OUT permit 10
 match ip address prefix-list MINHAS-REDES
exit
!
route-map ISP-A-IN permit 10
 set local-preference 150
exit
```

Pontos que merecem atenção:

- **`no bgp ebgp-requires-policy`** — desde a versão 7.4 o FRR implementa a RFC 8212: sem route-map de entrada e saída, sessões eBGP não trocam prefixo nenhum. Em produção, prefira **manter esse comportamento** e definir os route-maps de fato, em vez de desligar a proteção.
- **`network`** só anuncia o prefixo se ele existir na tabela de rotas do roteador. Uma rota estática para `Null0` (`ip route 198.51.100.0/24 blackhole`) é a forma usual de garantir isso.
- **`soft-reconfiguration inbound`** guarda os updates originais em memória, permitindo reaplicar políticas sem derrubar a sessão. Se ambos os lados suportam *route refresh* (praticamente universal hoje), isso é desnecessário e só consome RAM.
- **`maximum-prefix`** é uma proteção essencial: limita quantos prefixos você aceita antes de derrubar a sessão, evitando que um vizinho com problema encha sua memória.

### iBGP com route reflector

Em vez de malha completa, um ou dois roteadores atuam como refletores:

```
! No route reflector (AS 65001)
router bgp 65001
 bgp router-id 10.0.0.254
 bgp cluster-id 10.0.0.254
 neighbor CLIENTES peer-group
 neighbor CLIENTES remote-as 65001
 neighbor CLIENTES update-source lo
 neighbor 10.0.0.1 peer-group CLIENTES
 neighbor 10.0.0.2 peer-group CLIENTES
 neighbor 10.0.0.3 peer-group CLIENTES
 !
 address-family ipv4 unicast
  neighbor CLIENTES activate
  neighbor CLIENTES route-reflector-client
  neighbor CLIENTES next-hop-self
 exit-address-family
```

O uso de **peer-group** é fortemente recomendado: além de reduzir a configuração, o FRR agrupa o cálculo de update por grupo, o que melhora bastante o desempenho com muitos vizinhos.

`update-source lo` faz a sessão iBGP usar o loopback, para que ela sobreviva à queda de um enlace físico quando há múltiplos caminhos internos. Isso exige que o IGP (OSPF/IS-IS) distribua os loopbacks.

### BGP unnumbered

Recurso muito usado em datacenters: em vez de configurar endereços IPv4 ponto a ponto, a sessão sobe sobre o link-local IPv6 descoberto por Router Advertisement, e o próximo salto IPv4 é resolvido via RFC 5549.

```
router bgp 65001
 neighbor swp1 interface remote-as external
 neighbor swp2 interface remote-as external
 !
 address-family ipv4 unicast
  network 10.1.1.0/24
  neighbor swp1 activate
  neighbor swp2 activate
 exit-address-family
```

Uma folha de spine-leaf inteira pode ser configurada assim, sem planejar um único /31. `remote-as external` aceita qualquer ASN diferente do local — útil quando cada spine tem um ASN distinto.

### IPv6

```
router bgp 65001
 neighbor 2001:db8::1 remote-as 64500
 !
 address-family ipv6 unicast
  network 2001:db8:100::/48
  neighbor 2001:db8::1 activate
 exit-address-family
```

Note que um vizinho definido no nível global precisa de `activate` **em cada** address-family. Esquecer isso é um dos erros mais frequentes: a sessão sobe, mas nenhuma rota aparece.

---

## 7. Políticas de roteamento

### Ferramentas de filtro

| Ferramenta | Filtra por | Exemplo |
|---|---|---|
| `ip prefix-list` | Prefixo e tamanho de máscara | `ip prefix-list P seq 5 permit 10.0.0.0/8 le 24` |
| `bgp as-path access-list` | Regex sobre AS-path | `bgp as-path access-list PEERS permit ^64500_` |
| `bgp community-list` | Communities | `bgp community-list standard NAO-EXPORTAR permit 65001:100` |
| `route-map` | Combina os anteriores e aplica ações | ver abaixo |

O **route-map** é a peça central. Ele é uma sequência de cláusulas numeradas, avaliadas em ordem; a primeira que casar decide. Cláusula sem `match` casa com tudo. E há um **deny implícito no final** — se você criou um route-map, precisa terminar com uma cláusula `permit` genérica ou aceitará apenas o que casou explicitamente.

```
bgp community-list standard PREPEND-2X permit 65001:200
!
route-map CLIENTE-IN deny 10
 match ip address prefix-list BOGONS
exit
!
route-map CLIENTE-IN permit 20
 match community PREPEND-2X
 set as-path prepend 65001 65001
 set local-preference 200
 set community 65001:1000 additive
exit
!
route-map CLIENTE-IN permit 30
 set local-preference 200
exit
```

Após alterar uma política, aplique-a sem derrubar a sessão:

```
clear bgp ipv4 unicast 203.0.113.1 soft in
clear bgp * soft out
```

### Filtragem de bogons

Nunca aceite (nem anuncie) prefixos reservados ou privados:

```
ip prefix-list BOGONS seq 5  permit 0.0.0.0/8 le 32
ip prefix-list BOGONS seq 10 permit 10.0.0.0/8 le 32
ip prefix-list BOGONS seq 15 permit 100.64.0.0/10 le 32
ip prefix-list BOGONS seq 20 permit 127.0.0.0/8 le 32
ip prefix-list BOGONS seq 25 permit 169.254.0.0/16 le 32
ip prefix-list BOGONS seq 30 permit 172.16.0.0/12 le 32
ip prefix-list BOGONS seq 35 permit 192.0.2.0/24 le 32
ip prefix-list BOGONS seq 40 permit 192.168.0.0/16 le 32
ip prefix-list BOGONS seq 45 permit 198.18.0.0/15 le 32
ip prefix-list BOGONS seq 50 permit 224.0.0.0/4 le 32
ip prefix-list BOGONS seq 55 permit 240.0.0.0/4 le 32
ip prefix-list BOGONS seq 60 permit 0.0.0.0/0 ge 25
```

A última linha rejeita prefixos mais específicos que /24, que na prática não são aceitos na tabela global.

---

## 8. Cenários comuns

### Multihoming com dois provedores

Objetivo típico: usar o ISP-A como primário e o ISP-B como backup.

**Saída** — controlada por local preference:

```
route-map ISP-A-IN permit 10
 set local-preference 200
exit
route-map ISP-B-IN permit 10
 set local-preference 100
exit
```

**Entrada** — controlada por prepend (menos determinístico, porque depende da política dos outros ASes):

```
route-map ISP-B-OUT permit 10
 match ip address prefix-list MINHAS-REDES
 set as-path prepend 65001 65001 65001
exit
```

Alternativa mais eficaz: usar as communities publicadas pelo provedor para pedir que ele reduza a local-pref das suas rotas internamente.

### Datacenter spine-leaf

Topologia moderna de datacenter usa eBGP em toda a fabric: cada leaf tem um ASN privado próprio, cada spine tem outro, e o ECMP faz o balanceamento.

```
router bgp 65101
 bgp router-id 10.0.0.101
 bgp bestpath as-path multipath-relax
 timers bgp 3 9
 neighbor FABRIC peer-group
 neighbor FABRIC remote-as external
 neighbor swp1 interface peer-group FABRIC
 neighbor swp2 interface peer-group FABRIC
 !
 address-family ipv4 unicast
  redistribute connected route-map SERVIDORES
  maximum-paths 8
 exit-address-family
```

`bestpath as-path multipath-relax` é indispensável aqui: sem ele, o BGP só faz ECMP entre caminhos com AS-paths idênticos, e como cada spine tem ASN diferente, não haveria balanceamento.

### Blackhole de destino sob DDoS

```
ip route 198.51.100.42/32 blackhole
!
route-map ISP-A-OUT permit 5
 match ip address prefix-list BLACKHOLE
 set community 65535:666 additive
exit
```

O provedor, ao ver a community `blackhole` bem conhecida (RFC 7999), descarta o tráfego para aquele /32 na borda dele, protegendo o restante do seu link.

---

## 9. Operação e troubleshooting

### Comandos essenciais

```bash
show bgp summary                          # estado das sessões e nº de prefixos
show bgp ipv4 unicast                     # tabela BGP
show bgp ipv4 unicast 198.51.100.0/24     # detalhe de um prefixo
show bgp neighbor 203.0.113.1             # tudo sobre um vizinho
show bgp neighbor 203.0.113.1 received-routes   # exige soft-reconfiguration
show bgp neighbor 203.0.113.1 advertised-routes
show bgp ipv4 unicast regexp _64500_      # rotas que passam por um AS
show bgp ipv4 unicast community 65001:100
show ip route bgp                          # o que foi instalado na RIB
show bgp nexthop                           # resolução de next-hop
```

### Roteiro de diagnóstico

**A sessão não sobe.** Verifique conectividade IP até o vizinho (`ping`), depois a porta 179 (`ss -tnp | grep 179`, firewall, `nftables`/`iptables`). Confirme `remote-as` em ambos os lados, confira o `update-source` se a sessão é entre loopbacks, e valide o `ebgp-multihop` se os roteadores não são adjacentes. `show bgp neighbor` mostra o último erro de NOTIFICATION.

**A sessão está Established mas não recebo rotas.** Cheque nesta ordem: o vizinho está `activate` na address-family correta? Existe route-map de entrada com deny implícito? A RFC 8212 (`ebgp-requires-policy`) está bloqueando? O outro lado está mesmo anunciando (`show bgp neighbor X advertised-routes` do lado dele)?

**Recebo a rota mas ela não é instalada.** Quase sempre é next-hop irresolvível. `show bgp ipv4 unicast <prefixo>` mostrará `(inaccessible)`. Em iBGP, a causa clássica é falta de `next-hop-self` no roteador de borda. Outra possibilidade é distância administrativa: uma rota estática ou OSPF para o mesmo prefixo pode estar vencendo.

**A rota escolhida não é a que eu esperava.** `show bgp ipv4 unicast <prefixo>` lista todos os caminhos e marca o melhor com `>`, indicando o critério que decidiu. Percorra o algoritmo da seção 4 na ordem.

**Flapping.** Verifique estabilidade do enlace, ative BFD para detecção rápida e considere `neighbor X timers` mais folgados se o problema for perda esporádica de keepalive.

### BFD para convergência rápida

```
bfd
 peer 203.0.113.1
  detect-multiplier 3
  receive-interval 300
  transmit-interval 300
 exit
exit
!
router bgp 65001
 neighbor 203.0.113.1 bfd
```

Com BFD, uma falha é detectada em menos de um segundo, em vez dos 180 s do hold timer padrão.

---

## 10. Boas práticas de segurança

**Filtre sempre, nos dois sentidos.** A regra de ouro é: aceite de clientes apenas os prefixos que eles registraram, aceite de peers apenas os cones deles, e anuncie apenas o seu próprio espaço. Ferramentas como `bgpq4` geram prefix-lists automaticamente a partir de objetos IRR:

```bash
bgpq4 -F "ip prefix-list CLIENTE-X seq %n permit %A/%l %m\n" AS-CLIENTE-X
```

**Use RPKI.** O FRR suporta validação de origem via RTR. Combinado a um validador local (Routinator, rpki-client):

```
rpki
 rpki cache 192.0.2.10 3323 preference 1
 rpki polling_period 300
exit
!
route-map ISP-IN deny 5
 match rpki invalid
exit
route-map ISP-IN permit 10
 match rpki notfound
 set local-preference 90
exit
route-map ISP-IN permit 20
 match rpki valid
 set local-preference 110
exit
```

**Autentique as sessões.** `neighbor X password` (TCP-MD5) é o mínimo; TCP-AO é mais moderno. Combine com GTSM:

```
neighbor 203.0.113.1 ttl-security hops 1
```

**Limite prefixos** com `maximum-prefix`, e monitore. Um cliente que anuncia a tabela inteira por engano derruba a sessão em vez de derrubar sua rede.

**Siga o MANRS.** As quatro ações — filtragem, antispoofing, coordenação e validação global — cobrem essencialmente tudo acima.

---

## 11. FAQ

**O que exatamente diferencia BGP de OSPF? Quando uso cada um?**
OSPF é um IGP: descobre vizinhos automaticamente, tem visão completa da topologia interna e converge em segundos escolhendo o caminho de menor custo. BGP é um EGP: sessões manuais, sem visão de topologia, decisão baseada em política. Na prática você usa os dois juntos — o IGP carrega os endereços de infraestrutura e loopbacks (poucas centenas de rotas), e o BGP carrega os prefixos de cliente e da Internet (centenas de milhares). Em datacenters modernos, porém, é comum usar BGP puro até o leaf, dispensando o IGP.

**Preciso de um ASN público e de um bloco IP para usar BGP?**
Para participar da Internet global, sim: você precisa de ASN e de espaço de endereçamento alocados pelo NIC.br (no Brasil), além de contratos de trânsito ou peering. Para uso interno — datacenter, laboratório, VPN entre filiais — ASNs privados e espaço RFC 1918 funcionam perfeitamente.

**Devo pedir tabela completa (full table) ou rota padrão ao meu provedor?**
Rota padrão basta se você tem um único provedor: consome quase nada de memória e o resultado de encaminhamento é idêntico. Tabela completa faz sentido em multihoming, quando você quer escolher a melhor saída por destino. Hoje a tabela IPv4 passa de 900 mil rotas e exige alguns GB de RAM no FRR — planeje com folga. Uma opção intermediária é receber rota padrão + rotas dos clientes do provedor.

**Quanta memória o FRR consome com a tabela completa?**
Depende de quantas sessões recebem a tabela e se `soft-reconfiguration inbound` está ativo (ele praticamente duplica o consumo por vizinho). Como ordem de grandeza, reserve algo entre 4 e 8 GB para uma borda com duas ou três sessões full table em IPv4 e IPv6. Prefira `route refresh` ao soft-reconfiguration.

**Por que a sessão eBGP subiu mas nenhum prefixo é trocado?**
Provavelmente pela RFC 8212, implementada no FRR desde a 7.4: sessões eBGP sem route-map de entrada **e** de saída não trocam prefixos. A saída correta é criar as políticas. O comando `no bgp ebgp-requires-policy` desliga a proteção e é aceitável em laboratório, mas não em produção.

**Qual a diferença entre `network`, `redistribute` e `aggregate-address`?**
`network` anuncia um prefixo exato, e só se ele já existir na tabela de rotas. `redistribute` injeta em massa rotas de outra fonte (connected, static, ospf) — poderoso e perigoso, use sempre com route-map restringindo o que entra. `aggregate-address` cria um sumário a partir de rotas mais específicas presentes na tabela BGP; com `summary-only` ele suprime as específicas.

**Meu prefixo aparece em `show bgp` mas não em `show ip route`. Por quê?**
Duas causas dominam. A primeira é next-hop irresolvível — o BGP mantém a rota na tabela mas não a instala porque não sabe como alcançar o próximo salto; em iBGP, resolve-se com `next-hop-self` no roteador que aprendeu a rota externamente. A segunda é uma rota concorrente com distância administrativa menor (estática = 1, OSPF = 110, eBGP = 20, iBGP = 200).

**Preciso mesmo de full mesh no iBGP?**
Sim, a menos que use route reflectors ou confederações. A regra de que um roteador não repassa a outro iBGP o que aprendeu de um iBGP existe para prevenir loops. Route reflector é a solução padrão e muito mais simples que confederação; use dois refletores redundantes com o mesmo `cluster-id`.

**Como faço o tráfego *entrar* pelo link que eu quero?**
Essa é a parte difícil, porque a decisão é dos outros. As alternativas, da mais fraca à mais forte: AS-path prepend (funciona parcialmente, ignorado por quem usa local-pref), MED (só vale entre ASes adjacentes e muitos ignoram), communities do provedor (a mais eficaz — o provedor ajusta a local-pref dele conforme sua marcação), e anúncio de prefixos mais específicos por um dos links (funciona quase sempre, mas polui a tabela global e é malvisto).

**O que é BGP unnumbered e vale a pena?**
É uma sessão eBGP sobre link-local IPv6, sem endereços IP configurados no enlace, com resolução de next-hop IPv4 pela RFC 5549. Em fabrics de datacenter vale muito: elimina o planejamento de /31, torna a configuração de cada switch praticamente idêntica e simplifica automação. Entre organizações diferentes, é raro — provedores costumam exigir endereçamento tradicional.

**Como aplico mudanças de política sem derrubar a sessão?**
`clear bgp <vizinho> soft in` e `clear bgp <vizinho> soft out`. O soft in requer route refresh (negociado automaticamente por praticamente todos os roteadores modernos) ou `soft-reconfiguration inbound`. Evite `clear bgp *` sem `soft` — isso derruba as sessões de verdade e provoca reconvergência completa.

**FRR é confiável em produção?**
Sim. É o plano de controle de Cumulus Linux, SONiC, VyOS e diversas distribuições de roteador, e roda em grandes datacenters e provedores. Como qualquer software, exige atenção a versões: use uma release estável, acompanhe as notas de segurança e teste upgrades antes de aplicar na borda.

**Qual a diferença entre `frr.conf` integrado e arquivos separados por daemon?**
O modo integrado (`service integrated-vtysh-config`) mantém tudo em `/etc/frr/frr.conf` e é o padrão atual: `write memory` no vtysh salva a configuração inteira de uma vez. O modo legado usava `bgpd.conf`, `ospfd.conf` etc. separados. O integrado é mais simples de versionar e automatizar.

**Como faço backup e versionamento da configuração?**
`vtysh -c "show running-config"` gera a configuração completa; jogue a saída em um repositório Git com um cron ou pipeline de CI. Para provisionamento, o FRR é bem suportado por Ansible (módulos `frr_*`) e por ferramentas como NAPALM.

**Devo usar RPKI? Ele quebra alguma coisa?**
Deve. A fração de rotas classificadas como *invalid* na tabela global é pequena e majoritariamente composta de erros de configuração ou sequestros. Comece marcando com local-pref reduzida em vez de descartar, observe por algumas semanas e então passe a rejeitar. Rode o validador localmente e monitore-o — se o cache cair, o FRR passa a tratar tudo como *notfound*, o que é o comportamento seguro.

**O que causa route leak e como evito?**
Um vazamento acontece quando você anuncia para um provedor ou peer rotas que aprendeu de outro provedor ou peer, oferecendo trânsito acidental. A defesa é filtro de saída explícito: anuncie apenas seu próprio espaço e o de seus clientes, usando prefix-lists geradas a partir de IRR ou communities internas que marquem a origem de cada rota. `no bgp ebgp-requires-policy` desativado ajuda a impor essa disciplina.

**Consigo simular tudo isso sem hardware?**
Sim, e é o recomendado antes de tocar em produção. Containerlab com imagens FRR, GNS3, EVE-NG ou simplesmente múltiplos containers Docker com namespaces de rede permitem montar topologias completas de spine-leaf ou multihoming em um notebook.

**BGP funciona com VRFs e MPLS L3VPN no FRR?**
Sim. O FRR suporta VRFs (`vrf` no zebra, `router bgp <asn> vrf <nome>`), a address-family `l2vpn evpn` para VXLAN/EVPN — muito usada em datacenter — e `ipv4 vpn` para L3VPN com MPLS. EVPN no FRR é maduro e é a base das implementações de Cumulus e SONiC.

---

## 12. Referências

- RFC 4271 — A Border Gateway Protocol 4 (BGP-4)
- RFC 4456 — BGP Route Reflection
- RFC 5549 — Advertising IPv4 NLRI with an IPv6 Next Hop
- RFC 6793 — BGP Support for Four-Octet AS Numbers
- RFC 7999 — BLACKHOLE Community
- RFC 8092 — BGP Large Communities
- RFC 8212 — Default External BGP Route Propagation Behavior
- RFC 8893 / RFC 6811 — BGP Prefix Origin Validation (RPKI)
- Documentação oficial do FRRouting — https://docs.frrouting.org
- MANRS — https://www.manrs.org
- Iljitsch van Beijnum, *BGP* (O'Reilly)
- Dinesh Dutta, *BGP in the Data Center* (O'Reilly)