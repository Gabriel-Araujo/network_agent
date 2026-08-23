# Exemplos BGP

Coletânea de cenários em que o BGP é o protocolo principal, mas trabalha junto com outra tecnologia (IGP, MPLS, VXLAN, IPsec, VRRP, RPKI, PBR, Kubernetes). Sintaxe FRRouting, salvo indicação em contrário. Os trechos são ilustrativos e não foram testados em equipamento.

## Indice

1. eBGP com originação por agregado e rota estática
2. iBGP sobre OSPF como underlay
3. Route reflector iBGP
4. Confederações BGP
5. eBGP com BFD
6. BGP na WAN com VRRP na LAN
7. Política de rotas: prefix-list, as-path e communities
8. Hardening de sessão: MD5, GTSM, maximum-prefix e dampening
9. Validação de origem com RPKI
10. Anúncio condicional para trânsito de backup
11. Multihoming com engenharia de tráfego
12. MP-BGP dual-stack IPv4/IPv6 em sessão única
13. MPLS L3VPN com VPNv4 e VRF
14. Vazamento de rotas entre VRFs
15. BGP labeled-unicast (BGP-LU)
16. EVPN/VXLAN sobre fabric eBGP
17. BGP sobre túnel GRE com IPsec
18. Mitigação com RTBH e FlowSpec
19. BGP com PBR para saída seletiva
20. Anycast com ExaBGP e health check
21. MetalLB em modo BGP no Kubernetes
22. BGP unnumbered com ECMP em host Linux
23. Manutenção com graceful restart e graceful shutdown

---

## Exemplo 1

**eBGP com originação por agregado e rota estática.** O agregado só é anunciado se existir na RIB local, então a rota estática para descarte serve de âncora: garante o anúncio permanente do bloco e evita que tráfego para endereços não utilizados fique em laço.

```
! âncora do agregado - sem ela, o prefixo pode sumir do anúncio
ip route 198.51.100.0/24 blackhole
!
router bgp 65001
 bgp router-id 203.0.113.2
 no bgp default ipv4-unicast
 neighbor 203.0.113.1 remote-as 64512
 neighbor 203.0.113.1 description upstream
 !
 address-family ipv4 unicast
  network 198.51.100.0/24
  ! anuncia só o agregado, suprimindo os mais específicos internos
  aggregate-address 198.51.100.0/24 summary-only
  neighbor 203.0.113.1 activate
  neighbor 203.0.113.1 route-map RM-OUT out
  neighbor 203.0.113.1 route-map RM-IN in
 exit-address-family
exit
!
ip prefix-list PL-MEUS seq 10 permit 198.51.100.0/24
route-map RM-OUT permit 10
 match ip address prefix-list PL-MEUS
exit
route-map RM-IN permit 10
exit
```

---

## Exemplo 2

**iBGP sobre OSPF como underlay.** Divisão clássica de papéis: o IGP carrega apenas loopbacks e enlaces internos, garantindo a resolução de next-hop; o iBGP carrega as rotas externas. `update-source` e `next-hop-self` são o que faz a combinação funcionar.

```
! ---- underlay: OSPF só com infraestrutura ----
interface lo
 ip address 10.0.0.1/32
 ip ospf area 0
exit
interface eth1
 ip ospf area 0
 ip ospf network point-to-point
exit
!
router ospf
 ospf router-id 10.0.0.1
 passive-interface default
 no passive-interface eth1
exit
!
! ---- overlay: iBGP entre loopbacks ----
router bgp 65000
 bgp router-id 10.0.0.1
 no bgp default ipv4-unicast
 neighbor IBGP peer-group
 neighbor IBGP remote-as 65000
 neighbor IBGP update-source lo
 neighbor 10.0.0.2 peer-group IBGP
 neighbor 10.0.0.3 peer-group IBGP
 !
 address-family ipv4 unicast
  neighbor IBGP activate
  ! obrigatório na borda: o next-hop externo não é alcançável pelo OSPF
  neighbor IBGP next-hop-self
 exit-address-family
exit
```

---

## Exemplo 3

**Route reflector iBGP.** Elimina a malha completa de N×(N-1)/2 sessões. Os clientes só falam com os refletores; os refletores refletem entre clientes preservando `originator-id` e `cluster-list` para evitar laços.

```
! ---- no refletor ----
router bgp 65000
 bgp router-id 10.0.0.254
 bgp cluster-id 10.0.0.254
 no bgp default ipv4-unicast
 !
 neighbor CLIENTES peer-group
 neighbor CLIENTES remote-as 65000
 neighbor CLIENTES update-source lo
 neighbor 10.0.0.1 peer-group CLIENTES
 neighbor 10.0.0.2 peer-group CLIENTES
 neighbor 10.0.0.3 peer-group CLIENTES
 !
 address-family ipv4 unicast
  neighbor CLIENTES activate
  neighbor CLIENTES route-reflector-client
  ! preserva múltiplos caminhos para os clientes (RFC 7911)
  neighbor CLIENTES addpath-tx-all-paths
 exit-address-family
exit
!
! ---- no cliente: sessão iBGP normal, sem nada de especial ----
router bgp 65000
 neighbor 10.0.0.254 remote-as 65000
 neighbor 10.0.0.254 update-source lo
 address-family ipv4 unicast
  neighbor 10.0.0.254 activate
 exit-address-family
exit
```

---

## Exemplo 4

**Confederações BGP.** Alternativa ao route reflector: o AS é dividido em sub-ASes privados que se enxergam como eBGP internamente, mas aparecem como um único AS para o mundo externo.

```
router bgp 65001
 ! identidade vista de fora
 bgp confederation identifier 65000
 ! sub-ASes que fazem parte da mesma confederação
 bgp confederation peers 65002 65003
 bgp router-id 10.0.0.1
 no bgp default ipv4-unicast
 !
 ! vizinho dentro do mesmo sub-AS = iBGP
 neighbor 10.0.0.2 remote-as 65001
 neighbor 10.0.0.2 update-source lo
 !
 ! vizinho em outro sub-AS = eBGP de confederação
 neighbor 10.0.1.1 remote-as 65002
 neighbor 10.0.1.1 update-source lo
 neighbor 10.0.1.1 ebgp-multihop 2
 !
 ! trânsito externo enxerga apenas AS 65000
 neighbor 203.0.113.1 remote-as 64512
 !
 address-family ipv4 unicast
  neighbor 10.0.0.2 activate
  neighbor 10.0.1.1 activate
  neighbor 10.0.1.1 next-hop-self
  neighbor 203.0.113.1 activate
 exit-address-family
exit
```

---

## Exemplo 5

**eBGP com BFD.** O BGP detecta falha por hold timer (dezenas de segundos). O BFD reduz isso para menos de um segundo sem exigir keepalives BGP agressivos. Os dois protocolos se complementam: BFD detecta, BGP reage.

```
bfd
 profile FABRIC
  detect-multiplier 3
  receive-interval 300
  transmit-interval 300
 exit
exit
!
router bgp 65001
 bgp router-id 10.0.0.1
 no bgp default ipv4-unicast
 !
 neighbor FABRIC peer-group
 neighbor FABRIC remote-as external
 ! associa a sessão BGP ao perfil BFD
 neighbor FABRIC bfd profile FABRIC
 neighbor eth1 interface peer-group FABRIC
 neighbor eth2 interface peer-group FABRIC
 !
 address-family ipv4 unicast
  network 10.0.0.1/32
  neighbor FABRIC activate
  maximum-paths 8
 exit-address-family
exit
!
! verificação: show bfd peers brief / show bgp neighbor
```

---

## Exemplo 6

**BGP na WAN com VRRP na LAN.** Dois roteadores de borda com sessões eBGP independentes para provedores diferentes, mas um único gateway virtual para os hosts. O VRRP resolve o lado do host; o BGP resolve o lado do mundo.

```
! ---- LAN: gateway virtual 10.10.10.1 ----
interface eth0
 ip address 10.10.10.2/24
 vrrp 10
 vrrp 10 ip 10.10.10.1
 vrrp 10 priority 200
 vrrp 10 advertisement-interval 1000
exit
!
! ---- WAN: sessão eBGP com o provedor A ----
interface eth1
 ip address 203.0.113.2/30
exit
!
router bgp 65001
 bgp router-id 10.0.0.1
 no bgp default ipv4-unicast
 neighbor 203.0.113.1 remote-as 64512
 !
 ! iBGP com o roteador par para sincronizar rotas externas
 neighbor 10.10.10.3 remote-as 65001
 !
 address-family ipv4 unicast
  network 198.51.100.0/24
  neighbor 203.0.113.1 activate
  neighbor 203.0.113.1 route-map RM-UP-OUT out
  neighbor 10.10.10.3 activate
  neighbor 10.10.10.3 next-hop-self
 exit-address-family
exit
!
! no par: mesma config com "vrrp 10 priority 100" e provedor B
! habilitar vrrpd em /etc/frr/daemons
```

---

## Exemplo 7

**Política de rotas.** Combinação dos três filtros que sustentam qualquer borda: prefix-list para o que, as-path access-list para de onde, community para o porquê. O `deny` implícito ao final do route-map é a proteção real contra vazamento.

```
! ---- o que aceito ----
ip prefix-list PL-CLIENTE seq 10 permit 198.51.100.0/24 le 24
ip prefix-list PL-BOGONS seq 10 permit 10.0.0.0/8 le 32
ip prefix-list PL-BOGONS seq 20 permit 172.16.0.0/12 le 32
ip prefix-list PL-BOGONS seq 30 permit 192.168.0.0/16 le 32
ip prefix-list PL-BOGONS seq 40 permit 0.0.0.0/0 ge 25
!
! ---- de onde aceito ----
bgp as-path access-list AS-CLIENTE permit ^65010$
!
! ---- rótulos semânticos ----
bgp community-list standard CL-CLIENTE permit 65000:100
bgp community-list standard CL-PEER    permit 65000:200
bgp community-list standard CL-TRANSIT permit 65000:300
!
route-map RM-CLIENTE-IN deny 10
 match ip address prefix-list PL-BOGONS
exit
route-map RM-CLIENTE-IN permit 20
 match ip address prefix-list PL-CLIENTE
 match as-path AS-CLIENTE
 set community 65000:100 additive
 set local-preference 300
exit
! deny implícito no seq 99: nada mais entra
!
route-map RM-TRANSIT-OUT permit 10
 ! ao trânsito, anuncia clientes e prefixos próprios - nunca rotas de peers
 match community CL-CLIENTE
exit
route-map RM-TRANSIT-OUT permit 20
 match ip address prefix-list PL-MEUS
exit
```

---

## Exemplo 8

**Hardening de sessão.** Quatro camadas independentes: autenticação do TCP, restrição de TTL, limite de prefixos e amortecimento de instabilidade. Nenhuma delas substitui as outras.

```
router bgp 65001
 neighbor 203.0.113.1 remote-as 64512
 ! autenticação TCP-MD5 (referência a cofre, nunca literal)
 neighbor 203.0.113.1 password {{BGP_SECRET}}
 ! GTSM: só aceita pacotes com TTL compatível com 1 salto
 neighbor 203.0.113.1 ttl-security hops 1
 neighbor 203.0.113.1 timers 10 30
 !
 address-family ipv4 unicast
  neighbor 203.0.113.1 activate
  ! derruba a sessão acima de 1M de prefixos, alerta em 90%, religa em 30 min
  neighbor 203.0.113.1 maximum-prefix 1000000 90 restart 30
  ! penaliza prefixos instáveis: half-life 15min, reuse 750, suppress 2000, max 60min
  bgp dampening 15 750 2000 60
 exit-address-family
exit
!
! dampening é controverso em trânsito full-table: avalie antes de habilitar
! verificação: show bgp ipv4 unicast dampening dampened-paths
```

---

## Exemplo 9

**Validação de origem com RPKI.** O BGP não autentica quem originou um prefixo. O RPKI acrescenta essa camada via cache validador externo (routinator, rpki-client, fort), e o resultado vira condição de route-map.

```
! daemon: bgpd compilado com suporte a RPKI + habilitar em /etc/frr/daemons
rpki
 rpki polling_period 300
 rpki cache 192.0.2.10 3323 preference 1
 rpki cache 192.0.2.11 3323 preference 2
exit
!
route-map RM-TRANSIT-IN deny 10
 ! origem inválida: descarta
 match rpki invalid
exit
route-map RM-TRANSIT-IN permit 20
 match rpki valid
 set local-preference 200
exit
route-map RM-TRANSIT-IN permit 30
 ! sem ROA publicado: aceita, mas com preferência menor
 match rpki notfound
 set local-preference 100
exit
!
router bgp 65001
 address-family ipv4 unicast
  neighbor 203.0.113.1 route-map RM-TRANSIT-IN in
 exit-address-family
exit
!
! a sintaxe do nó rpki mudou entre versões do FRR: confirme na sua
! verificação: show rpki prefix-table / show bgp ipv4 unicast rpki invalid
```

---

## Exemplo 10

**Anúncio condicional para trânsito de backup.** O prefixo só é anunciado ao provedor secundário quando uma rota-âncora do provedor primário desaparece. Evita pagar trânsito de backup enquanto o link principal está saudável.

```
! rota-âncora que só existe enquanto o primário está vivo
ip prefix-list PL-ANCORA seq 10 permit 192.0.2.0/24
route-map RM-CHECK-PRIMARIO permit 10
 match ip address prefix-list PL-ANCORA
exit
!
! o que anunciar quando a âncora sumir
ip prefix-list PL-MEUS seq 10 permit 198.51.100.0/24
route-map RM-ANUNCIA-BACKUP permit 10
 match ip address prefix-list PL-MEUS
exit
!
router bgp 65001
 neighbor 203.0.113.5 remote-as 64513
 neighbor 203.0.113.5 description trânsito-backup
 !
 address-family ipv4 unicast
  neighbor 203.0.113.5 activate
  ! anuncia RM-ANUNCIA-BACKUP somente se RM-CHECK-PRIMARIO não casar nada
  neighbor 203.0.113.5 advertise-map RM-ANUNCIA-BACKUP non-exist-map RM-CHECK-PRIMARIO
 exit-address-family
exit
!
! verificação: show bgp ipv4 unicast neighbors 203.0.113.5 advertised-routes
```

---

## Exemplo 11

**Multihoming com engenharia de tráfego.** Saída é decisão local (`local-preference`). Entrada é persuasão (`as-path prepend`, `MED`, communities do provedor). Os dois lados exigem mecanismos diferentes.

```
! ---- SAÍDA: prefiro sair pelo provedor A ----
route-map RM-A-IN permit 10
 set local-preference 200
exit
route-map RM-B-IN permit 10
 set local-preference 100
exit
!
! ---- ENTRADA: desencorajo entrada pelo provedor B ----
route-map RM-B-OUT permit 10
 match ip address prefix-list PL-MEUS
 set as-path prepend 65001 65001 65001
 ! muitos provedores publicam communities de controle; exemplo genérico:
 set community 64513:120 additive
exit
route-map RM-A-OUT permit 10
 match ip address prefix-list PL-MEUS
exit
!
router bgp 65001
 neighbor 203.0.113.1 remote-as 64512
 neighbor 203.0.113.5 remote-as 64513
 !
 address-family ipv4 unicast
  network 198.51.100.0/24
  neighbor 203.0.113.1 activate
  neighbor 203.0.113.1 route-map RM-A-IN in
  neighbor 203.0.113.1 route-map RM-A-OUT out
  neighbor 203.0.113.5 activate
  neighbor 203.0.113.5 route-map RM-B-IN in
  neighbor 203.0.113.5 route-map RM-B-OUT out
 exit-address-family
exit
```

---

## Exemplo 12

**MP-BGP dual-stack em sessão única.** Uma sessão BGP sobre transporte IPv6 carrega famílias IPv4 e IPv6. Reduz pela metade o número de sessões a operar, ao custo de acoplar as duas famílias ao mesmo destino de falha.

```
router bgp 65001
 bgp router-id 10.0.0.1
 no bgp default ipv4-unicast
 !
 neighbor 2001:db8::1 remote-as 64512
 neighbor 2001:db8::1 description upstream-dual-stack
 ! permite next-hop IPv4 sobre transporte IPv6 (RFC 5549)
 neighbor 2001:db8::1 capability extended-nexthop
 !
 address-family ipv4 unicast
  network 198.51.100.0/24
  neighbor 2001:db8::1 activate
  neighbor 2001:db8::1 route-map RM-V4-IN in
  neighbor 2001:db8::1 route-map RM-V4-OUT out
 exit-address-family
 !
 address-family ipv6 unicast
  network 2001:db8:100::/48
  neighbor 2001:db8::1 activate
  neighbor 2001:db8::1 route-map RM-V6-IN in
  neighbor 2001:db8::1 route-map RM-V6-OUT out
 exit-address-family
exit
```

---

## Exemplo 13

**MPLS L3VPN com VPNv4 e VRF.** BGP transporta as rotas de cliente com route distinguisher e route target; o MPLS transporta os pacotes. O IGP do core não conhece nenhuma rota de cliente.

```
! VRF do cliente no plano de dados
vrf CLIENTE-A
exit-vrf
!
interface eth2
 ! interface voltada ao CE
 ip address 10.20.0.1/30
exit
!
! ---- instância BGP do VRF ----
router bgp 65000 vrf CLIENTE-A
 no bgp default ipv4-unicast
 neighbor 10.20.0.2 remote-as 65100
 !
 address-family ipv4 unicast
  neighbor 10.20.0.2 activate
  ! exporta para a tabela VPN com RD/RT e rótulo
  rd vpn export 65000:100
  rt vpn both 65000:100
  label vpn export auto
  export vpn
  import vpn
 exit-address-family
exit
!
! ---- instância global: sessão VPNv4 com o PE remoto ----
router bgp 65000
 bgp router-id 10.0.0.1
 no bgp default ipv4-unicast
 neighbor 10.0.0.2 remote-as 65000
 neighbor 10.0.0.2 update-source lo
 !
 address-family ipv4 vpn
  neighbor 10.0.0.2 activate
 exit-address-family
exit
```

---

## Exemplo 14

**Vazamento de rotas entre VRFs.** Compartilhar serviços comuns (DNS, monitoração, saída para internet) entre VRFs isolados, sem passar por firewall externo nem duplicar tabela.

```
vrf CLIENTE-A
exit-vrf
vrf SERVICOS
exit-vrf
!
router bgp 65000 vrf CLIENTE-A
 address-family ipv4 unicast
  ! importa as rotas do VRF de serviços compartilhados
  import vrf SERVICOS
  ! filtra o que entra: só o bloco de serviços, nada mais
  import vrf route-map RM-IMPORT-SERVICOS
 exit-address-family
exit
!
router bgp 65000 vrf SERVICOS
 address-family ipv4 unicast
  redistribute connected
 exit-address-family
exit
!
ip prefix-list PL-SERVICOS seq 10 permit 10.99.0.0/24
route-map RM-IMPORT-SERVICOS permit 10
 match ip address prefix-list PL-SERVICOS
exit
!
! verificação: show ip route vrf CLIENTE-A
```

---

## Exemplo 15

**BGP labeled-unicast.** O BGP distribui prefixo e rótulo MPLS na mesma NLRI, permitindo LSP fim-a-fim entre domínios sem estender o IGP nem o LDP através da fronteira.

```
router bgp 65001
 bgp router-id 10.0.0.1
 no bgp default ipv4-unicast
 !
 neighbor 10.0.0.2 remote-as 65001
 neighbor 10.0.0.2 update-source lo
 neighbor 203.0.113.1 remote-as 65002
 !
 ! família rotulada: cada prefixo carrega seu rótulo
 address-family ipv4 labeled-unicast
  network 10.0.0.1/32
  neighbor 10.0.0.2 activate
  neighbor 10.0.0.2 next-hop-self
  neighbor 203.0.113.1 activate
 exit-address-family
exit
!
! requer suporte MPLS no kernel: modprobe mpls_router
! sysctl net.mpls.platform_labels=100000
! sysctl net.mpls.conf.eth1.input=1
```

---

## Exemplo 16

**EVPN/VXLAN sobre fabric eBGP.** O BGP distribui endereços MAC (tipo 2) e prefixos IP (tipo 5) como rotas; o VXLAN encapsula. Substitui flood-and-learn e elimina STP do fabric.

```
! interface VXLAN e bridge são criadas fora do FRR (ip link / netplan)
! ip link add vxlan100 type vxlan id 100 dstport 4789 local 10.0.0.1 nolearning
!
router bgp 65001
 bgp router-id 10.0.0.1
 no bgp default ipv4-unicast
 bgp bestpath as-path multipath-relax
 !
 neighbor FABRIC peer-group
 neighbor FABRIC remote-as external
 neighbor eth1 interface peer-group FABRIC
 neighbor eth2 interface peer-group FABRIC
 !
 address-family ipv4 unicast
  network 10.0.0.1/32
  neighbor FABRIC activate
 exit-address-family
 !
 address-family l2vpn evpn
  neighbor FABRIC activate
  ! anuncia automaticamente todas as VNIs locais
  advertise-all-vni
  advertise-default-gw
 exit-address-family
exit
!
! verificação: show bgp l2vpn evpn / show evpn vni / show evpn mac vni 100
```

---

## Exemplo 17

**BGP sobre túnel GRE com IPsec.** O IPsec provê confidencialidade e o GRE provê uma interface roteável; o BGP faz o resto. Combinação típica de interconexão site-a-site e multi-cloud.

```
! ---- túnel GRE (fora do FRR) ----
! ip link add gre1 type gre local 203.0.113.2 remote 198.51.100.2 ttl 255
! ip link set gre1 up mtu 1400
! ip addr add 169.254.10.1/30 dev gre1
! IPsec (strongswan) protege o tráfego GRE entre os endpoints públicos
!
router bgp 65001
 bgp router-id 10.0.0.1
 no bgp default ipv4-unicast
 !
 neighbor 169.254.10.2 remote-as 65002
 neighbor 169.254.10.2 description site-remoto
 neighbor 169.254.10.2 bfd
 ! túnel instável derruba a sessão rápido demais sem isto
 neighbor 169.254.10.2 timers 10 30
 !
 address-family ipv4 unicast
  network 10.10.0.0/16
  neighbor 169.254.10.2 activate
  neighbor 169.254.10.2 route-map RM-SITE-IN in
  neighbor 169.254.10.2 route-map RM-SITE-OUT out
  neighbor 169.254.10.2 maximum-prefix 5000 80 restart 15
 exit-address-family
exit
!
! MTU do túnel exige clamping de MSS no firewall para evitar PMTUD quebrado
```

---

## Exemplo 18

**Mitigação com RTBH e FlowSpec.** RTBH usa a community BLACKHOLE (RFC 7999) para propagar descarte por destino; FlowSpec (RFC 8955) propaga regras por 5-tupla. Ambos usam o BGP como barramento de política de segurança.

```
! ---- RTBH: no nó gatilho ----
ip route 198.51.100.66/32 blackhole
ip prefix-list PL-BLACKHOLE seq 10 permit 198.51.100.66/32
!
route-map RM-RTBH-OUT permit 10
 match ip address prefix-list PL-BLACKHOLE
 set community 65535:666 no-export additive
exit
!
! ---- RTBH: nas bordas ----
ip route 192.0.2.1/32 blackhole
bgp community-list standard CL-BLACKHOLE permit 65535:666
!
route-map RM-EDGE-IN permit 10
 match community CL-BLACKHOLE
 set ip next-hop 192.0.2.1
exit
route-map RM-EDGE-IN permit 20
exit
!
! ---- FlowSpec: recepção e instalação local ----
router bgp 65001
 address-family ipv4 flowspec
  neighbor 10.0.0.9 activate
  local-install eth1
 exit-address-family
exit
!
! o FRR recebe e instala FlowSpec; a originação costuma vir de ExaBGP/GoBGP
! verificação: show bgp ipv4 flowspec detail
```

---

## Exemplo 19

**BGP com PBR para saída seletiva.** O BGP escolhe o melhor caminho por prefixo de destino. O PBR sobrepõe essa decisão para tráfego que precisa sair por outro lugar — por origem, porta ou marcação.

```
! habilitar pbrd em /etc/frr/daemons
!
! grupo de próximos saltos alternativo
nexthop-group NHG-PROVEDOR-B
 nexthop 203.0.113.5
exit
!
pbr-map PBR-SAIDA seq 10
 ! tráfego desta origem sai pelo provedor B, independente do BGP
 match src-ip 10.10.50.0/24
 set nexthop-group NHG-PROVEDOR-B
exit
!
pbr-map PBR-SAIDA seq 20
 match dst-ip 192.0.2.0/24
 match dst-port 443
 set nexthop 203.0.113.1
exit
!
interface eth0
 pbr-policy PBR-SAIDA
exit
!
! verificação: show pbr map / show pbr interface
```

---

## Exemplo 20

**Anycast com ExaBGP e health check.** O anúncio fica atrelado ao resultado de um teste de saúde do serviço: se o health check falha, o processo retira o prefixo e o tráfego reflui para as outras réplicas em menos de um segundo.

```
# /etc/exabgp/exabgp.conf
process healthcheck {
    run /usr/bin/python3 -m exabgp healthcheck --cmd "curl -sf http://localhost:8080/health" --label anycast --interval 2 --fast-interval 1 --rise 3 --fall 2;
    encoder text;
}

neighbor 10.0.0.100 {
    router-id 10.1.0.7;
    local-address 10.1.0.7;
    local-as 65107;
    peer-as 65100;
    hold-time 9;

    family {
        ipv4 unicast;
    }

    api {
        processes [ healthcheck ];
    }
}

# o VIP anycast 10.1.0.53/32 fica em lo; o healthcheck anuncia/retira o /32
# equivalente em FRR: "network 10.1.0.53/32" + ip addr add/del 10.1.0.53/32 dev lo
```

---

## Exemplo 21

**MetalLB em modo BGP no Kubernetes.** Os nós do cluster estabelecem sessão BGP com os ToRs e anunciam os IPs de `Service` do tipo LoadBalancer como `/32`, apenas a partir dos nós com endpoints saudáveis.

```yaml
apiVersion: metallb.io/v1beta1
kind: IPAddressPool
metadata:
  name: pool-servicos
  namespace: metallb-system
spec:
  addresses:
    - 10.1.0.0/24
---
apiVersion: metallb.io/v1beta2
kind: BGPPeer
metadata:
  name: tor1
  namespace: metallb-system
spec:
  myASN: 65200
  peerASN: 65100
  peerAddress: 10.0.0.101
  # BFD acelera a detecção de falha do nó
  bfdProfile: fabric
---
apiVersion: metallb.io/v1beta1
kind: BGPAdvertisement
metadata:
  name: adv-servicos
  namespace: metallb-system
spec:
  ipAddressPools:
    - pool-servicos
  aggregationLength: 32
  communities:
    - 65000:100
# no ToR: maximum-prefix e filtro de entrada restrito ao bloco 10.1.0.0/24
```

---

## Exemplo 22

**BGP unnumbered com ECMP em host Linux.** Sessões por interface usando IPv6 link-local e next-hop IPv4 (RFC 5549), sem endereçamento ponto-a-ponto. O kernel precisa ser ajustado para que o multipath e o encaminhamento funcionem de fato.

```
! ---- /etc/sysctl.d/90-frr.conf ----
! net.ipv4.conf.all.forwarding=1
! net.ipv6.conf.all.forwarding=1
! net.ipv4.fib_multipath_hash_policy=1   (hash por 5-tupla, não só por IP)
! net.ipv4.conf.all.rp_filter=0
!
interface lo
 ip address 10.0.0.7/32
exit
!
router bgp 65107
 bgp router-id 10.0.0.7
 no bgp default ipv4-unicast
 bgp bestpath as-path multipath-relax
 !
 neighbor TOR peer-group
 neighbor TOR remote-as external
 neighbor TOR bfd
 neighbor eth0 interface peer-group TOR
 neighbor eth1 interface peer-group TOR
 !
 address-family ipv4 unicast
  network 10.0.0.7/32
  neighbor TOR activate
  neighbor TOR route-map RM-OUT out
  neighbor TOR route-map RM-IN in
  maximum-paths 2
 exit-address-family
exit
!
ip prefix-list PL-ME seq 10 permit 10.0.0.7/32
route-map RM-OUT permit 10
 match ip address prefix-list PL-ME
exit
route-map RM-IN permit 10
exit
```

---

## Exemplo 23

**Manutenção com graceful restart e graceful shutdown.** Dois mecanismos com propósitos distintos: o restart preserva o encaminhamento durante reinício do daemon; o shutdown (RFC 8326) drena tráfego antes de uma parada planejada, evitando perda de conexões em curso.

```
router bgp 65001
 ! preserva a FIB enquanto o bgpd reinicia
 bgp graceful-restart
 bgp graceful-restart preserve-fw-state
 bgp graceful-restart restart-time 120
 bgp graceful-restart stalepath-time 360
exit
!
! ---- procedimento de manutenção ----
! 1) drenar: marca as rotas com GRACEFUL_SHUTDOWN e local-pref 0
!    vtysh -c "conf t" -c "router bgp 65001" -c "bgp graceful-shutdown"
! 2) aguardar o esvaziamento das conexões existentes
! 3) parar o serviço e executar a manutenção
! 4) reverter:
!    vtysh -c "conf t" -c "router bgp 65001" -c "no bgp graceful-shutdown"
! 5) verificar: show bgp ipv4 unicast neighbors <peer> advertised-routes
!
! honrar a community na entrada, quando o vizinho a envia:
bgp community-list standard CL-GSHUT permit graceful-shutdown
!
route-map RM-PEER-IN permit 10
 match community CL-GSHUT
 set local-preference 0
exit
route-map RM-PEER-IN permit 20
exit
```