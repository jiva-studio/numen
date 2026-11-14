import { readFileSync, readdirSync } from 'node:fs'
import { resolve, sep } from 'node:path'
import type { Plugin } from 'vite'
import { UI_STYLES, removePresentRules } from '@numen/ui/vite'
import { getCandidates, getUtilityCss } from '../../src/utilityCss.ts'

/** The id the stories import the utilities of every plugin frontend by. */
const STORY_UTILITIES = 'virtual:plugin-utilities.css'

function getFrontendFiles(pluginsDir: string): string[] {
  return readdirSync(pluginsDir, { recursive: true, encoding: 'utf8' })
    .filter((path) => /^[^/]+\/frontend\/.+\.(vue|ts)$/.test(path.split(sep).join('/')))
    .filter((path) => !/\.(stories|test)\.ts$/.test(path))
    .map((path) => resolve(pluginsDir, path))
}

/**
 * Serves `virtual:plugin-utilities.css` to the stories: the utilities every plugin frontend uses
 * that the build of `@numen/ui` does not carry, as each plugin's `dist/styles.css` carries them.
 * `pluginsDir` holds one directory per plugin.
 */
export function createStoryUtilitiesPlugin(pluginsDir: string): Plugin {
  return {
    name: 'numen-story-utilities',
    resolveId: (id) => (id === STORY_UTILITIES ? `\0${STORY_UTILITIES}` : undefined),
    async load(id) {
      if (id !== `\0${STORY_UTILITIES}`) return undefined
      const css = await getUtilityCss(getCandidates(getFrontendFiles(pluginsDir)))
      return removePresentRules(css, readFileSync(UI_STYLES, 'utf8'))
    },
  }
}
