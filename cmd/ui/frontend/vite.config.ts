import {defineConfig} from 'vitest/config'
import react from '@vitejs/plugin-react'
import macros from 'unplugin-parcel-macros'
import wails from '@wailsio/runtime/plugins/vite'
import optimizeLocales from '@react-aria/optimize-locales-plugin'

// Config do S2 (docs/REACT_SPECTRUM_S2.md, §2): o macro `style` antes de tudo,
// lightningcss para deduplicar o CSS atômico e todo o CSS do S2 num chunk só.
export default defineConfig({
  plugins: [
    macros.vite(),
    // O app é só pt-BR: os outros locales do S2 saem do bundle (docs/FRONTEND.md, Idioma).
    {...optimizeLocales.vite({locales: ['pt-BR']}), enforce: 'pre'},
    react(),
    wails('./bindings'),
  ],
  // O `./style` do S2 só exporta a condição `node` (roda no build, via macro); sem isto o
  // scan de dependências do `vite dev` falha ao resolvê-lo e recarrega a página na 1ª vez.
  optimizeDeps: {exclude: ['@react-spectrum/s2/style']},
  // O bloco do template do Wails v3. O `host` também é necessário: o proxy do dev disca só em IPv4
  // (`tcp4 127.0.0.1`), e no Windows o `localhost` padrão do Vite escuta em `::1`.
  server: {
    host: '127.0.0.1',
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
  },
  // O Markdown (nos testes) importa componentes do S2, que importam `.css`: o Vite os processa em
  // vez de deixar o Node carregá-los como módulo. Sem o `fsModuleCache`: um módulo lido do cache não
  // passa pelo macro, que então não registra o `macro-<hash>.css` que o módulo importa.
  test: {fsModuleCache: false, server: {deps: {inline: [/@react-spectrum\/s2/]}}},
  build: {
    target: ['es2022'],
    cssMinify: 'lightningcss',
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (/macro-(.*)\.css$/.test(id) || /@react-spectrum\/s2\/.*\.css$/.test(id)) {
            return 's2-styles'
          }
        },
      },
    },
  },
})
