// A ponte com o Go no Wails v3. O v3 não injeta `window.go` nem `window.runtime`:
// as chamadas vão pelo `@wailsio/runtime`, que também liga o arrasto da titlebar
// (`--wails-draggable`) só por ser importado. As bindings geradas não entram: os
// tipos do front já são os de `types.ts`, e `Call.ByName` dispensa o gerador no
// build. Ver docs/WAILS_V3.md, §2.
import { Call } from '@wailsio/runtime'

export { Application, Clipboard, Events, Window } from '@wailsio/runtime'

// `_wails.environment` é escrito pelo script que o Go injeta na página. No
// browser puro ele não existe, e a ponte também não.
const inWails = () => !!(window as any)._wails?.environment

// Os métodos do `App` de `cmd/ui`, pelo nome que o v3 dá a eles
// (`main.App.<Método>`). Fora do app devolve `undefined`, como o
// `window.go?.main?.App` do v2.
const proxy: any = new Proxy(
  {},
  { get: (_, method) => (...args: unknown[]) => Call.ByName(`main.App.${String(method)}`, ...args) },
)
export const goApp = (): any => (inWails() ? proxy : undefined)

/** O texto do erro de uma chamada: o v3 rejeita com `RuntimeError`, e `String(e)` traria o prefixo. */
export const errorText = (e: unknown) => (e instanceof Error ? e.message : String(e))
