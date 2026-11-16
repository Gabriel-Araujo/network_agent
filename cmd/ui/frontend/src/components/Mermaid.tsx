import { render } from 'pele'
import { peleErrorMessage } from '../mermaidBlocks'

// Renderiza um Bloco de diagrama (sintaxe Mermaid) como SVG inline, pelo pele.
//
// O render do pele é síncrono, não toca no DOM e não emite `id` no SVG, então o
// componente é função pura: sem estado, sem efeito, sem contador de id. O SVG
// existe já no primeiro render, logo o elemento nunca aparece vazio — era isso
// que o render assíncrono do mermaid (mais o chunk do tipo de diagrama carregado
// por demanda) deixava em branco por segundos.
//
// Segurança: o fonte é escrito pelo modelo. O pele escapa o texto de rótulo e só
// deixa passar link de esquema conhecido (`linkSchemes` default: http, https,
// mailto, tel), com `rel="noopener"` — um `click ... href "javascript:..."` não
// vira link. O tema sai das variáveis `--pele-*` definidas no App.css.
// Tudo medido em `.agent/ui/assets/03-fallback-diagrama/probe-pele.txt`.
export default function Mermaid({ code }: { code: string }) {
  let svg: string
  try {
    svg = render(code).svg
  } catch (e) {
    // o objeto inteiro fica no devtools do `wails dev`; só a mensagem vai para a
    // tela, que é onde o usuário precisa dela.
    console.error('diagrama: render falhou', e)
    // Bloco não renderizado: o aviso usa o mesmo triplo o-quê/por-quê/próximo
    // dos outros erros do app, e o fonte vai como texto — em JSX, para o React
    // escapar, e não por innerHTML.
    return (
      <div className="mermaid mermaid--failed" role="alert">
        <p className="error__title">O diagrama não renderizou</p>
        <p className="error__why">{peleErrorMessage(e)}</p>
        <p className="error__next">
          O fonte está abaixo; peça o diagrama de novo se quiser tentar outra forma.
        </p>
        <pre className="mermaid-source">{code}</pre>
      </div>
    )
  }

  return <div className="mermaid" dangerouslySetInnerHTML={{ __html: svg }} />
}
