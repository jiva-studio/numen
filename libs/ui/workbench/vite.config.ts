import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import dts from 'vite-plugin-dts'
import { LIGHTNING_CSS, createLayeredStylesPlugin } from '@numen/ui/vite'

const src = fileURLToPath(new URL('./src', import.meta.url))

export default defineConfig({
  plugins: [
    vue(),
    tailwindcss(),
    createLayeredStylesPlugin(`${src}/styles.css`),
    dts({ include: ['src'], exclude: ['src/vite.ts'], entryRoot: 'src' }),
  ],
  resolve: { alias: { '@': src } },
  css: { lightningcss: LIGHTNING_CSS },
  build: {
    lib: {
      entry: fileURLToPath(new URL('./src/index.ts', import.meta.url)),
      formats: ['es'],
      cssFileName: 'numen-shell',
    },
    rollupOptions: {
      // The window's Vue and the window's components: one copy of each in a page.
      external: ['vue', '@numen/ui'],
      output: {
        preserveModules: true,
        preserveModulesRoot: src,
        entryFileNames: (chunk) => `${chunk.name.replaceAll('node_modules/', 'vendor/')}.js`,
      },
    },
  },
})
