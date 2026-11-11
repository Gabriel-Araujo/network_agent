package config

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	resources "github.com/Gabriel-Araujo/network_agent"
)

// Dir devolve o diretório de dados do aplicativo no home (~/.network_agent):
// o diretório que contém o config.json e, ao lado dele, o history/, o logs/ e
// o espelho do material do agente. O literal do nome mora em constants.go e
// não se repete: todo caminho do app no home se deriva daqui.
func Dir() string {
	return filepath.Dir(fileLocation)
}

// SetDir aponta o diretório de dados do app para dir e devolve a função que
// restaura o anterior (molde do SetCreator, para t.Cleanup). É a costura de
// teste do home, exportada porque quem grava nele mora fora deste pacote — o
// espelho do retrieval e o do briefing —, e um teste que não possa trocar o
// home escreveria no ~/.network_agent da máquina que roda o teste.
func SetDir(dir string) (restore func()) {
	saved := fileLocation
	fileLocation = filepath.Join(dir, configFileName)
	materialRoot = sync.OnceValues(seedMaterial)
	return func() {
		fileLocation = saved
		materialRoot = sync.OnceValues(seedMaterial)
	}
}

// materialRoot projeta o espelho na primeira chamada e devolve a raiz dele.
// sync.OnceValues e não sync.Once: o erro da semeadura é memoizado junto da
// raiz, então a segunda leitura de material recebe a mesma resposta sem
// reprojetar — e sem engolir a falha.
var materialRoot = sync.OnceValues(seedMaterial)

// MaterialRoot semeia o espelho na primeira chamada e devolve a raiz contra a
// qual um caminho "resources/..." resolve — o mesmo diretório que Dir(),
// porque a projeção preserva o caminho embutido (ADR 0004).
//
// Existe para quem precisa da raiz em vez de um arquivo: a regra de prefixo
// das file tools escolhe entre esta raiz e o Workspace e depois deixa o
// SafePath juntar o relativo, então ela pede a raiz, não o join.
func MaterialRoot() (string, error) {
	return materialRoot()
}

// MaterialFile devolve o caminho absoluto de um arquivo de material do agente
// dentro do espelho, semeando-o na primeira chamada.
//
// O relativo é o mesmo que a árvore de fonte usa — MaterialFile("resources/agents")
// —, porque o espelho preserva o caminho embutido (ADR 0004): é o que mantém o
// "resources/agents/teacher/tech-packs/" que o AGENT.md do Teacher ensina
// literalmente verdadeiro, agora resolvendo contra o home.
//
// Substitui paths.RepoFile no caminho de material do agente: a origem é o
// binário, não a árvore de fonte, logo nada aqui depende de o repo existir na
// máquina que roda o app.
func MaterialFile(rel string) (string, error) {
	root, err := materialRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, filepath.FromSlash(rel)), nil
}

// seedMaterial escreve o material embutido no binário em Dir(), preservando o
// caminho: resources/agents/x no embed.FS vira <Dir()>/resources/agents/x no
// disco.
//
// É projeção, não cópia-se-faltar: cada arquivo é sobrescrito quando o
// conteúdo embutido difere do que está no disco, e por isso uma edição do
// usuário no espelho não sobrevive à próxima abertura. Material de agente se
// customiza no repo. A comparação existe só para não reescrever 82 arquivos a
// cada processo que lê material.
//
// ponytail: a escrita não é atômica — dois processos semeando ao mesmo tempo
// podem deixar um arquivo lido pela metade por um terceiro. Os dois escrevem
// os mesmos bytes e a janela é de microssegundos; se isso aparecer num
// transcript, o conserto é o tmp+rename do internal/history/filestore.go.
func seedMaterial() (string, error) {
	root := Dir()

	err := fs.WalkDir(resources.Material, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		embedded, err := resources.Material.ReadFile(path)
		if err != nil {
			return err
		}
		dst := filepath.Join(root, filepath.FromSlash(path))
		if onDisk, rErr := os.ReadFile(dst); rErr == nil && bytes.Equal(onDisk, embedded) {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
			return err
		}
		return os.WriteFile(dst, embedded, 0o600)
	})
	if err != nil {
		return "", fmt.Errorf("config: falha ao projetar o material do agente em %s: %w", root, err)
	}

	log.Debugf("material do agente projetado em %s\n", root)
	return root, nil
}
