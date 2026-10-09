import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import dts from 'vite-plugin-dts'

export default defineConfig({
  plugins: [vue(), tailwindcss(), dts({ include: ['src'], rollupTypes: true })],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  build: {
    lib: {
      entry: fileURLToPath(new URL('./src/index.ts', import.meta.url)),
      formats: ['es'],
      cssFileName: 'numen-ui',
    },
    // Vue is the host application's, not ours; two copies of it in one page
    // is a broken application, not a large bundle.
    rollupOptions: {
      external: ['vue'],
      // One file per module, so a window linking a handful of components is
      // not handed the editor and its grammars along with them.
      output: {
        preserveModules: true,
        preserveModulesRoot: fileURLToPath(new URL('./src', import.meta.url)),
        // A dependency kept as its own file is filed under `vendor/`. Left
        // where rollup puts it, the path says `node_modules`, and a test
        // runner takes that as a package to load outside the bundle — which
        // hands the page a second Vue and breaks every render.
        entryFileNames: (chunk) => `${chunk.name.replaceAll('node_modules/', 'vendor/')}.js`,
      },
    },
  },
})
