/**
 * The build every plugin frontend shares: the one ES module and stylesheet it is packed into.
 *
 * A plugin resolves `@numen/plugin-sdk` and `@numen/ui` as installed packages, by their `exports`.
 */
import { existsSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
import type { Plugin, PluginOption, UserConfig } from 'vite'
import { HOST_MODULES } from './hostModules.ts'
import { LIGHTNING_CSS } from '@numen/ui/vite'
import { createUtilitiesPlugin } from './viteStyles.ts'

function createStylesPlugin(root: string): Plugin {
  return {
    name: 'ensure-styles',
    apply: 'build',
    closeBundle() {
      const cssPath = resolve(root, 'dist', 'styles.css')
      if (!existsSync(cssPath)) {
        writeFileSync(cssPath, '')
      }
    },
  }
}

export interface PluginBuildOptions {
  readonly root: string
  readonly plugins: readonly PluginOption[]
}

const CSS_OPTIONS = { lightningcss: LIGHTNING_CSS }

/**
 * The build of one plugin frontend into `dist/index.js` and `dist/styles.css`. The modules the
 * window serves are left out of the bundle and imported from their paths in `HOST_MODULES`.
 */
export function createPluginBuildConfig(options: PluginBuildOptions): UserConfig {
  return {
    plugins: [
      ...options.plugins,
      createUtilitiesPlugin(options.root),
      createStylesPlugin(options.root),
    ],
    css: CSS_OPTIONS,
    build: {
      outDir: 'dist',
      emptyOutDir: true,
      lib: {
        entry: resolve(options.root, 'frontend/index.ts'),
        formats: ['es'],
        fileName: () => 'index.js',
        cssFileName: 'styles',
      },
      rollupOptions: {
        external: Object.keys(HOST_MODULES),
        output: {
          paths: HOST_MODULES,
          assetFileNames: (assetInfo) => {
            if (assetInfo.name?.endsWith('.css')) return 'styles.css'
            return assetInfo.name || 'asset'
          },
        },
      },
    },
  }
}
