package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/internal/config"
	"github.com/Gabriel-Araujo/network_agent/internal/logger"
	"github.com/Gabriel-Araujo/network_agent/internal/skills"
	filetools "github.com/Gabriel-Araujo/network_agent/internal/tools/file"
	"github.com/Gabriel-Araujo/network_agent/pkg/util"
	"github.com/openai/openai-go/v3/responses"
)

// log é o handle do pacote (uma declaração por pacote, não por arquivo).
var log = logger.Named("TOOLS")

var Tools = FileTools

// GetFileAndSkillTools returns the toolset of a dispatched Subagente: the file
// tools plus read_skill_file, and no use_skill — so no skill catalog reaches
// it. Dropping the catalog is the cut, not filtering it: every skill in the
// registry is either the generic's pipeline (frr-intent-analyzer, the
// frr-file-builder stub, frr-rag-retriever, which no live prompt even names)
// or "mermaid", already *mounted* in the prompt of the two Subagentes who draw
// (promptSkills). So the catalog held nothing a dispatched Subagente could
// use, while its descriptions ordered activation anyway: 12 of the 30 calls in
// the ticket 28 batch went to the two pipeline skills, and t11 spent both its
// calls there and never reached mermaid. read_skill_file stays, so the mermaid
// references remain reachable — the Illustrator and Teacher configs name both
// the tool and the skill, and HandleSkillCall resolves against the full
// registry.
func GetFileAndSkillTools() []responses.ToolUnionParam {
	return append([]responses.ToolUnionParam{skills.ReadSkillFileTool()}, Tools...)
}

// GetSkillAndTools is the generic agent's toolset: file + the whole skills
// catalog, plus the subagent tool.
func GetSkillAndTools() []responses.ToolUnionParam {
	skillsTools := skills.LoadSkills().BuildTools()

	return append(append(skillsTools, Tools...), SubAgentTool)
}

// Failure é uma chamada de tool que falhou. É o registro que sobe até o Chat
// (ticket 18) para o app poder dizer ao usuário **o que ele fez** em vez de
// deixar a falha morrer num log.Error que ninguém lê.
//
// Carrega os quatro fatos que o Aviso do ticket 19 formata: a tool, o caminho
// **como o modelo pediu** (não o absoluto resolvido, que já está no
// log.Error), a **raiz** contra a qual ele resolveu — o knob que o ticket 02
// tornou efetivo e o ticket 10 desdobrou em duas — e a causa tipada.
//
// Path e Root ficam vazios quando a falha é anterior a eles: argumentos
// ilegíveis, tool sem caminho, raiz de material que não resolveu.
type Failure struct {
	Tool string
	Path string
	Root string
	Err  error
}

// failure embrulha o erro de uma tool no registro, ou devolve nil quando a
// chamada foi bem: é o que mantém "sucesso não avisa" em um lugar só.
func failure(tool, path, root string, err error) *Failure {
	if err == nil {
		return nil
	}
	return &Failure{Tool: tool, Path: path, Root: root, Err: err}
}

// pathArg nomeia o argumento de caminho de cada file tool: o search chama o
// dele "path", as outras três "filePath". A raiz se resolve contra ele **antes**
// do switch, porque a regra das duas raízes mora num lugar só — se cada tool
// resolvesse a sua, as quatro poderiam discordar.
var pathArg = map[string]string{
	filetools.READ_TOOL_NAME:   "filePath",
	filetools.EDIT_TOOL_NAME:   "filePath",
	filetools.WRITE_TOOL_NAME:  "filePath",
	filetools.SEARCH_TOOL_NAME: "path",
}

// writesToDisk diz se a tool grava. É a lista da regra 2 do ticket 10: o
// EditFile entra porque lê **e** escreve, e é a escrita que a raiz só-leitura
// recusa.
func writesToDisk(toolName string) bool {
	return toolName == filetools.WRITE_TOOL_NAME || toolName == filetools.EDIT_TOOL_NAME
}

// rootFor devolve a raiz contra a qual filePath resolve, ou a recusa que vai ao
// modelo no lugar do resultado da tool. É o único lugar onde essa escolha mora.
//
// São duas raízes (ticket 05, ADR 0004): caminho começando com MIRROR_PREFIX
// resolve no espelho de material do agente, só leitura; todo o resto resolve no
// Workspace, a pasta que o usuário escolheu e a única raiz de escrita. É
// **prefixo** e não ordem de busca porque ordem esconde a raiz dentro de um
// os.Stat: com o arquivo nas duas raízes o modelo lê um e não sabe qual.
//
// appRoot é a raiz só-leitura — config.Dir(), o diretório onde o espelho é
// projetado e onde o config.json e o history/ moram ao lado dele. É parâmetro,
// e não chamada direta, para o teste poder pôr um t.TempDir() no lugar do home
// da máquina: o mesmo molde do forSubagentAt.
//
// A recusa e o registro andam juntos: refusal != "" se e somente se fail !=
// nil — a string é o que o modelo lê, o registro é o que o app lê.
func rootFor(appRoot, toolName, filePath, workspace string) (root string, refusal string, fail *Failure) {
	// Um caminho chega ao material do agente por duas formas: o prefixo, que é
	// como os prompts o ensinam, e o absoluto, que é como ele **volta** — as
	// saídas das tools são absolutas por decisão (ticket 05, Q9), e a descrição
	// manda o modelo reusar o caminho que recebeu. As duas formas têm de cair na
	// mesma raiz, senão reusar um caminho do search faria o arquivo ser
	// procurado no Workspace, onde ele não está.
	inMirror := strings.HasPrefix(filepath.ToSlash(filePath), filetools.MIRROR_PREFIX)
	if !inMirror && (filepath.IsAbs(filePath) || filepath.VolumeName(filePath) != "") {
		_, err := util.SafePath(appRoot, filePath)
		inMirror = err == nil
	}

	if inMirror {
		if writesToDisk(toolName) {
			log.Warn("escrita recusada na raiz só-leitura", "tool", toolName, "path", filePath, "root", appRoot)
			return "", filetools.READ_ONLY_ROOT_NOTICE + filePath, failure(toolName, filePath, appRoot, filetools.ErrReadOnlyRoot)
		}
		return appRoot, "", nil
	}

	// A checagem é a cada chamada, não no startup: um startup envelhece no
	// segundo em que um drive sai no meio da sessão.
	if strings.TrimSpace(workspace) == "" {
		log.Warn("caminho recusado: nenhum Workspace escolhido", "tool", toolName, "path", filePath)
		return "", filetools.NO_WORKSPACE_NOTICE, failure(toolName, filePath, "", filetools.ErrNoWorkspace)
	}
	if info, err := os.Stat(workspace); err != nil || !info.IsDir() {
		log.Warn("caminho recusado: o Workspace não existe mais", "tool", toolName, "path", filePath, "workspace", workspace, "err", err)
		return "", filetools.WORKSPACE_GONE_NOTICE + workspace, failure(toolName, filePath, workspace, filetools.ErrWorkspaceGone)
	}

	// O prefixo não fecha a regra 2 sozinho: com o Workspace **acima** do
	// diretório do app (o home escolhido como pasta, ou o próprio
	// ~/.network_agent), um caminho sem prefixo ainda cai dentro dele —
	// "config.json" ou ".network_agent/history/" pedidos ao WriteFile. O agente
	// gravando ali é a falha que aviso nenhum recupera, então a recusa olha o
	// alvo resolvido, não só o prefixo. SafePath é o predicado de contenção que
	// já existe: erro nil significa "está dentro".
	if writesToDisk(toolName) {
		if target, err := util.SafePath(workspace, filePath); err == nil {
			if _, inside := util.SafePath(appRoot, target); inside == nil {
				log.Warn("escrita recusada: o alvo cai no diretório do app", "tool", toolName, "path", filePath, "target", target, "root", appRoot)
				return "", filetools.READ_ONLY_ROOT_NOTICE + filePath, failure(toolName, filePath, appRoot, filetools.ErrReadOnlyRoot)
			}
		}
	}

	return workspace, "", nil
}

// CallFunction executa a file tool que o modelo chamou, resolvendo antes a raiz
// contra a qual o caminho vale. workspace é a pasta do usuário, fornecida pelo
// front-end — a UI dá a escolhida, o cmd/cli dá o CWD —, e vazia significa
// "nenhuma escolhida", que é um estado tratado, não um valor a consertar.
//
// Devolve o resultado que vai ao modelo e, quando a chamada falhou, o registro
// da falha (nil no sucesso). Os dois saem juntos de propósito: o resultado é
// uma string de erro que o modelo lê, e o registro é a mesma falha num canal
// que o app lê — antes do ticket 18 só existia o primeiro, e por isso a recusa
// chegava ao usuário no máximo pela boca do modelo.
func CallFunction(
	ctx context.Context,
	call responses.ResponseFunctionToolCall,
	workspace string,
) (responses.ResponseInputItemUnionParam, *Failure) {
	log.Debugf("function '%s' called with arguments: [%s]\n", call.Name, call.Arguments)

	args := make(map[string]string)

	err := json.Unmarshal([]byte(call.Arguments), &args)
	if err != nil {
		log.Error("argumentos inválidos na chamada de função", "name", call.Name, "err", err)
		return responses.ResponseInputItemParamOfFunctionCallOutput("Error: Failed to get args from function call object."),
			failure(call.Name, "", "", fmt.Errorf("argumentos ilegíveis na chamada da tool: %w", err))
	}

	key, ok := pathArg[call.Name]
	if !ok {
		return responses.ResponseInputItemParamOfFunctionCallOutput("Error: Called invalid function"),
			failure(call.Name, "", "", errors.New("não é uma file tool"))
	}

	// MaterialRoot semeia o espelho na primeira chamada do processo (memoizada
	// por sync.OnceValues). Vem antes do switch porque é a raiz só-leitura dos
	// dois usos: o destino de um caminho com prefixo e o diretório que a escrita
	// não pode alcançar.
	appRoot, err := config.MaterialRoot()
	if err != nil {
		log.Error("falha ao resolver a raiz do material do agente", "name", call.Name, "err", err)
		return responses.ResponseInputItemParamOfFunctionCallOutput("Error: failed to read File content"),
			failure(call.Name, args[key], "", fmt.Errorf("resolvendo a raiz do material do agente: %w", err))
	}

	root, refusal, fail := rootFor(appRoot, call.Name, args[key], workspace)
	if refusal != "" {
		return responses.ResponseInputItemParamOfFunctionCallOutput(refusal), fail
	}

	// O caminho do registro é o que o modelo pediu (args[key]), não o absoluto
	// que a tool resolveu: o absoluto já está no log.Error, e quem lê o Aviso
	// precisa reconhecer o que pediu.
	switch call.Name {
	case filetools.READ_TOOL_NAME:
		out, err := filetools.ReadFile(root, args["filePath"])
		return responses.ResponseInputItemParamOfFunctionCallOutput(out), failure(call.Name, args[key], root, err)
	case filetools.EDIT_TOOL_NAME:
		out, err := filetools.EditFile(root, args["filePath"], args["oldText"], args["newText"])
		return responses.ResponseInputItemParamOfFunctionCallOutput(out), failure(call.Name, args[key], root, err)
	case filetools.WRITE_TOOL_NAME:
		out, err := filetools.WriteFile(root, args["filePath"], args["content"])
		return responses.ResponseInputItemParamOfFunctionCallOutput(out), failure(call.Name, args[key], root, err)
	case filetools.SEARCH_TOOL_NAME:
		out, err := filetools.SearchPattern(root, args["path"], args["pattern"])
		return responses.ResponseInputItemParamOfFunctionCallOutput(out), failure(call.Name, args[key], root, err)

	default:
		return responses.ResponseInputItemParamOfFunctionCallOutput("Error: Called invalid function"),
			failure(call.Name, args[key], root, errors.New("não é uma file tool"))
	}
}
