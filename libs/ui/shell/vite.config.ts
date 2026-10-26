import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import dts from 'vite-plugin-dts'
import { Features } from 'lightningcss'

const src = fileURLToPath(new URL('./src', import.meta.url))

export default defineConfig({
  plugins: [
    vue(),
    tailwindcss(),
    dts({ include: ['src'], exclude: ['src/vite.ts'], entryRoot: 'src' }),
  ],
  resolve: { alias: { '@': src } },
  // `light-dark()` stays as written: the window chooses a half with `color-scheme`, which the
  // lowered form does not read.
  css: { lightningcss: { exclude: Features.LightDark } },
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
