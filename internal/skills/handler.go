package skills

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/openai/openai-go/v3/responses"
)

// HandleSkillCall processa uma chamada de função relacionada a skills
// (use_skill / read_skill_file / read_skill_directory) e devolve o resultado para o modelo
// como uma saída de função (function call output).
//
// Devolve também a falha, quando houve: o erro é o canal que leva a falha de
// tool até o Chat (ticket 18), e sem ele a única notícia de que a skill não
// carregou é a frase seca que o modelo recebe.
//
// turn só pesa na ativação da frr-file-builder (ver activation).
func HandleSkillCall(call responses.ResponseFunctionToolCall, turn Turn) (responses.ResponseInputItemUnionParam, error) {
	reg := LoadSkills()

	switch call.Name {
	case ToolUseSkill:
		var args struct {
			SkillName string `json:"skill_name"`
		}
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return responses.ResponseInputItemParamOfFunctionCallOutput(fmt.Sprintf("erro: argumentos inválidos: %v", err)),
				fmt.Errorf("argumentos ilegíveis na chamada de skill: %w", err)
		}

		sk, ok := reg.Get(args.SkillName)
		if !ok {
			return responses.ResponseInputItemParamOfFunctionCallOutput(fmt.Sprintf("erro: skill %q não encontrada", args.SkillName)),
				fmt.Errorf("skill %q não encontrada no registry", args.SkillName)
		}
		body, err := reg.activation(sk, turn)
		if err != nil {
			return responses.ResponseInputItemParamOfFunctionCallOutput(fmt.Sprintf("erro: %v", err)), err
		}
		return responses.ResponseInputItemParamOfFunctionCallOutput(body), nil

	case ToolReadSkillFile:
		var args struct {
			SkillName    string `json:"skill_name"`
			RelativePath string `json:"relative_path"`
		}
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return responses.ResponseInputItemParamOfFunctionCallOutput(fmt.Sprintf("erro: argumentos inválidos: %v", err)),
				fmt.Errorf("argumentos ilegíveis na chamada de skill: %w", err)
		}

		content, err := reg.ReadFile(args.SkillName, args.RelativePath)
		if err != nil {
			return responses.ResponseInputItemParamOfFunctionCallOutput(fmt.Sprintf("erro: %v", err)),
				fmt.Errorf("lendo o arquivo %q da skill %q: %w", args.RelativePath, args.SkillName, err)
		}
		return responses.ResponseInputItemParamOfFunctionCallOutput(content), nil

	case ToolReadSkillDirectory:
		var args struct {
			SkillName    string `json:"skill_name"`
			RelativePath string `json:"relative_path"`
		}
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return responses.ResponseInputItemParamOfFunctionCallOutput(fmt.Sprintf("erro: argumentos inválidos: %v", err)),
				fmt.Errorf("argumentos ilegíveis na chamada de skill: %w", err)
		}

		items, err := reg.ListDir(args.SkillName, args.RelativePath)
		if err != nil {
			return responses.ResponseInputItemParamOfFunctionCallOutput(fmt.Sprintf("erro: %v", err)),
				fmt.Errorf("listando a pasta %q da skill %q: %w", args.RelativePath, args.SkillName, err)
		}
		return responses.ResponseInputItemParamOfFunctionCallOutput(strings.Join(items, "\n")), nil

	default:
		return responses.ResponseInputItemParamOfFunctionCallOutput("Error: Called invalid skill function"),
			errors.New("não é uma tool de skill")
	}
}

// Turn é o que a ativação da frr-file-builder precisa saber do turno em que
// roda: o protocol classificado, que nomeia o Tech pack, e se o prompt leva
// um bloco <recovered_context>. O zero é "nada classificado, nada buscado".
type Turn struct {
	Protocol  string
	Retrieved bool
}

// packSkill é a skill cuja ativação entrega o Tech pack do turno junto do
// corpo; qualquer outra ativa só com o corpo
// (.agent/frr-file-builder/issues/06-decidir-como-o-pack-chega-ao-modelo.md).
const packSkill = "frr-file-builder"

// packFiles são os arquivos do pack que a ativação entrega, nesta ordem.
// Exemplo nenhum: esses o modelo escolhe pelo índice do TECH.md e lê com
// read_skill_file.
var packFiles = []string{"TECH.md", "syntax.md", "doctrine.md"}

// activation é o resultado do use_skill: o corpo e, só para a packSkill, o
// pack de references/<protocol>/ (casado por nome exato) ou a linha que diz
// por que ele não vem, e a linha do alcance do RAG quando o turno buscou para
// um protocol. Pasta do pack sem um dos packFiles é erro: entrega parcial
// calada é o modelo construindo sem a regra achando que a tem.
func (r *Registry) activation(sk *Skill, turn Turn) (string, error) {
	if sk.Name != packSkill {
		return sk.Body, nil
	}
	var out strings.Builder
	out.WriteString(sk.Body)

	dir := "references/" + turn.Protocol
	info, statErr := os.Stat(filepath.Join(sk.Dir, filepath.FromSlash(dir)))
	switch {
	case turn.Protocol == "":
		fmt.Fprintf(&out, "\n\nThis turn has no classified technology, so no pack follows this body: "+
			"read the pack of each technology of the request that has a row in the table with %s.", ToolReadSkillFile)
	case statErr != nil || !info.IsDir():
		fmt.Fprintf(&out, "\n\nNo pack covers %[1]s, the technology of this turn. Above its first block write:\n"+
			"! WARNING: no pack covers %[1]s; built from the retrieved FRR documentation only", turn.Protocol)
	default:
		for _, name := range packFiles {
			rel := dir + "/" + name
			text, err := r.ReadFile(sk.Name, rel)
			if err != nil {
				return "", fmt.Errorf("pack %q da skill %q incompleto: %w", turn.Protocol, sk.Name, err)
			}
			fmt.Fprintf(&out, "\n\n<file path=%q>\n%s\n</file>", rel, strings.TrimRight(text, "\n"))
		}
	}

	if turn.Retrieved && turn.Protocol != "" {
		fmt.Fprintf(&out, "\n\nThe <recovered_context> of this turn was retrieved for %s only: "+
			"a part of the request in another technology with no pack has no source, so say which part and stop.", turn.Protocol)
	}
	return out.String(), nil
}

// ActivatedSkill devolve o nome da skill que uma chamada use_skill ordena
// ativar, e "" quando a chamada não é de ativação ou os argumentos não são
// legíveis. É o fato que o turno registra: sem ele, "o agente desenhou sem a
// skill" só se descobre perguntando ao modelo, que é como o dono descobriu
// (.agent/ui/issues/32-o-generico-desenha-sem-a-skill.md).
//
// O parse é o mesmo do HandleSkillCall, no mesmo encoding/json: dois parsers
// de strictness diferente sobre os mesmos bytes diriam coisas diferentes sobre
// a mesma chamada, e o nome registrado tem de ser o nome que carregou o corpo.
func ActivatedSkill(call responses.ResponseFunctionToolCall) string {
	if call.Name != ToolUseSkill {
		return ""
	}
	var args struct {
		SkillName string `json:"skill_name"`
	}
	if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
		return ""
	}
	return args.SkillName
}
