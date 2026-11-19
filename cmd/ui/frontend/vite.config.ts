import {defineConfig} from 'vite'
import react from '@vitejs/plugin-react'
import macros from 'unplugin-parcel-macros'

// Config do S2 (docs/REACT_SPECTRUM_S2.md, §2): o macro `style` antes de tudo,
// lightningcss para deduplicar o CSS atômico e todo o CSS do S2 num chunk só.
export default defineConfig({
  plugins: [macros.vite(), react()],
  // O `./style` do S2 só exporta a condição `node` (roda no build, via macro); sem isto o
  // scan de dependências do `vite dev` falha ao resolvê-lo e recarrega a página na 1ª vez.
  optimizeDeps: {exclude: ['@react-spectrum/s2/style']},
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
