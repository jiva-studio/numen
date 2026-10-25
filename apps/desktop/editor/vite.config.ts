import { fileURLToPath, URL } from 'node:url'
import { defineConfig, type Plugin } from 'vite'
import vue from '@vitejs/plugin-vue'
import { HOST_MODULES } from '@numen/plugin-sdk/host-modules'

/** The id of the entry that serves one host module to plugin bundles. */
const HOST_MODULE_ENTRY = 'host-module:'

/** Each host module as an entry that re-exports it, built to its path in `HOST_MODULES`. */
function createHostModuleEntries(): Plugin {
  return {
    name: 'host-module-entries',
    resolveId: (id) => (id.startsWith(HOST_MODULE_ENTRY) ? `\0${id}` : undefined),
    load: (id) =>
      id.startsWith(`\0${HOST_MODULE_ENTRY}`)
        ? `export * from '${id.slice(HOST_MODULE_ENTRY.length + 1)}'`
        : undefined,
  }
}

/** The name of each host module's entry: its path without the leading slash and `.js`. */
const hostModuleInputs = Object.fromEntries(
  Object.entries(HOST_MODULES).map(([name, path]) => [
    path.slice(1, -'.js'.length),
    HOST_MODULE_ENTRY + name,
  ]),
)

export default defineConfig({
  plugins: [vue(), createHostModuleEntries()],
  // `@/` is the window's own source. A path leaving its own folder is written
  // through it, so what layer is reached is read off the import itself.
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      '@numen/plugin-sdk': fileURLToPath(
        new URL('../../../libs/plugin-sdk-ts/src/index.ts', import.meta.url),
      ),
      '@numen/protocol': fileURLToPath(new URL('../../../libs/protocol/index.ts', import.meta.url)),
      '@numen/wire': fileURLToPath(new URL('../../../libs/wire/index.ts', import.meta.url)),
      '@numen/ui/styles.css': fileURLToPath(
        new URL('../../../libs/ui/dist/numen-ui.css', import.meta.url),
      ),
      '@numen/ui': fileURLToPath(new URL('../../../libs/ui/dist/index.js', import.meta.url)),
    },
  },
  // Built into the Go package that serves it: what is embedded has to sit
  // beside the code that embeds it.
  // `assets` is where the vault's own files are asked for, so the window's
  // built pieces are filed apart from them.
  build: {
    outDir: '../../../libs/core/adapter/window/editor/pages/app',
    emptyOutDir: true,
    assetsDir: 'built',
    rollupOptions: {
      input: {
        index: fileURLToPath(new URL('./index.html', import.meta.url)),
        ...hostModuleInputs,
      },
      // A host module keeps every export, and the page and the plugins share its one instance.
      preserveEntrySignatures: 'strict',
      output: {
        entryFileNames: (chunk) =>
          chunk.name in hostModuleInputs ? '[name].js' : 'built/[name]-[hash].js',
      },
    },
  },
})
