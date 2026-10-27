/**
 * The build every plugin frontend shares: the paths its imports resolve to,
 * and the one ES module and stylesheet it is packed into.
 *
 * A plugin's `vite.config.ts` and `vitest.config.ts` name the aliases and
 * externals they use; the paths and the shape of the bundle are written here.
 */
import { existsSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
import type { Plugin, PluginOption, UserConfig } from 'vite'
import { HOST_MODULES } from '@numen/plugin-sdk/host-modules'
import { LIGHTNING_CSS } from '@numen/ui/vite'
import { createUtilitiesPlugin } from './vite.styles.ts'

const modules = resolve(import.meta.dirname, '..')

/** Every alias a plugin may resolve, in the order Vite tries them. */
const ALIASES = {
  '@plugins': import.meta.dirname,
  '@numen/plugin-sdk/testing': resolve(modules, 'libs/plugin-sdk-ts/src/testing/index.ts'),
  '@numen/plugin-sdk': resolve(modules, 'libs/plugin-sdk-ts/src/index.ts'),
  '@numen/protocol/numen/v1': resolve(modules, 'libs/protocol/src/numen/v1'),
  '@numen/protocol': resolve(modules, 'libs/protocol/index.ts'),
  '@numen/ui/styles.css': resolve(modules, 'libs/ui/core/dist/numen-ui.css'),
  '@numen/ui': resolve(modules, 'libs/ui/core/dist/index.js'),
} as const

export type AliasName = keyof typeof ALIASES

/** The named aliases and their paths, in the order of `ALIASES`. */
export function getPluginAliases(names: readonly AliasName[]): Record<string, string> {
  const wanted = new Set(names)
  return Object.fromEntries(
    Object.entries(ALIASES).filter(([name]) => wanted.has(name as AliasName)),
  )
}

/** Writes an empty `dist/styles.css` when the bundle carries no stylesheet. */
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
  /** The plugin's directory, which holds `frontend/index.ts` and receives `dist/`. */
  readonly root: string
  /** The plugins of the build ahead of the stylesheet one, such as `vue()`. */
  readonly plugins: readonly PluginOption[]
  readonly aliases: readonly AliasName[]
}

/**
 * The build of one plugin frontend into `dist/index.js` and `dist/styles.css`. The modules the
 * window serves are left out of the bundle and imported from their paths in `HOST_MODULES`.
 */
export function createPluginBuildConfig(options: PluginBuildOptions): UserConfig {
  return {
    plugins: [...options.plugins, createUtilitiesPlugin(), createStylesPlugin(options.root)],
    resolve: { alias: getPluginAliases(options.aliases) },
    css: { lightningcss: LIGHTNING_CSS },
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
