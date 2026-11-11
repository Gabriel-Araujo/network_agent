package filetools

import "errors"

// AS FALHAS TIPADAS
//
// Toda file tool devolve (string, error). A string é o que o **modelo** lê e
// não muda — o erro é adicional, não substituto (ticket 18, §1). O erro é o que
// o **app** lê: é o canal que leva a falha até o Chat, de onde o Aviso ao
// usuário sai (ticket 19).
//
// Os sentinelas abaixo existem para quem formata a falha escolher o caso sem
// farejar prefixo de string. Prefixo não é contrato: são quatro grafias em duas
// línguas ("Error: ", "Error %s", "erro: ", "erro no sub-agente: "), e o
// ReadFile devolve conteúdo cru — um arquivo que começasse com "Error:" viraria
// falha num turno que leu bem.
//
// O caso inesperado (sem permissão, disco cheio, JSON que não serializa) não
// tem sentinela: vai o erro cru do sistema, embrulhado com o que a tool estava
// fazendo. Quem formata trata tipado o que conhece e mostra o cru no resto.
//
// O texto de cada sentinela é a **frase curta que o usuário lê** no Aviso
// (ticket 19, §3, via Reason): por isso é português, e por isso é curto — a
// ação, o caminho e a pasta são do Aviso, e repeti-los aqui faria a frase
// dizer duas vezes onde a tool olhou.
var (
	// ErrOutsideRoot: o caminho pedido resolve fora da raiz dele.
	ErrOutsideRoot = errors.New("caminho fora da raiz")

	// ErrNotFound: o arquivo alvo não existe. A tool sabe onde olhou; não
	// sabe se o arquivo existe em outro lugar.
	ErrNotFound = errors.New("arquivo não encontrado")

	// ErrNoWorkspace: nenhuma pasta foi escolhida para os arquivos do
	// usuário, logo não há onde ler nem gravar. Só o usuário conserta.
	ErrNoWorkspace = errors.New("nenhuma pasta escolhida para os arquivos do usuário")

	// ErrWorkspaceGone: a pasta escolhida não está mais lá — renomeada,
	// desmontada, num drive que saiu.
	ErrWorkspaceGone = errors.New("a pasta escolhida não existe mais")

	// ErrReadOnlyRoot: a escrita cai dentro do material do agente ou do
	// diretório do app ao lado dele, que são só de leitura.
	ErrReadOnlyRoot = errors.New("essa pasta é só de leitura")

	// ErrTooLarge: o conteúdo a gravar, ou o arquivo a editar, passa do
	// limite.
	ErrTooLarge = errors.New("tamanho acima do limite")

	// ErrTextNotFound: o old_text do EditFile não está no arquivo.
	ErrTextNotFound = errors.New("old_text não encontrado no arquivo")

	// ErrTextNotUnique: o old_text do EditFile casa mais de uma vez, e por
	// isso nada foi trocado.
	ErrTextNotUnique = errors.New("old_text não é único no arquivo")

	// ErrStale: o arquivo mudou entre a leitura e a escrita do EditFile.
	ErrStale = errors.New("arquivo modificado por outro processo")
)

// sentinelas são os casos tipados acima. A lista existe para o Reason abaixo,
// e um sentinela novo entra nela junto da declaração — quem esquecer cai no
// caso inesperado, que mostra o erro cru.
var sentinelas = []error{
	ErrOutsideRoot,
	ErrNotFound,
	ErrNoWorkspace,
	ErrWorkspaceGone,
	ErrReadOnlyRoot,
	ErrTooLarge,
	ErrTextNotFound,
	ErrTextNotUnique,
	ErrStale,
}

// Reason devolve o motivo curto da falha: a frase do caso tipado que err
// carrega, ou o err cru quando o caso é inesperado. É a última metade do Aviso
// que o usuário lê (ticket 19, §3).
//
// A frase é o texto do próprio sentinela, e não um segundo catálogo em quem
// formata: o par caso ↔ frase mora num lugar só, aqui. O cru só no caso
// inesperado porque o conhecido já vem embrulhado — o ErrOutsideRoot carrega
// o erro do SafePath, que é uma frase em inglês, e o Aviso é em português.
func Reason(err error) string {
	for _, sentinela := range sentinelas {
		if errors.Is(err, sentinela) {
			return sentinela.Error()
		}
	}
	return err.Error()
}
