/** The interface modules, each cruised from its own folder so its own tsconfig answers for what `@/` means. */
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

export const here = dirname(fileURLToPath(import.meta.url))
export const root = resolve(here, '../../..')
export const config = join(here, 'rules.cjs')
export const depcruise = join(here, 'node_modules/.bin/depcruise')

export const modules = [
  {
    name: '@numen/ui',
    at: 'modules/libs/ui',
    sources: ['src'],
    says: 'The shared components. Reaches nothing of ours.',
  },
  {
    name: '@numen/wire',
    at: 'modules/libs/wire',
    sources: ['index.ts'],
    says: "What a window's ports are answered with.",
  },
  {
    name: '@numen/editor',
    at: 'modules/apps/desktop/editor',
    sources: ['src'],
    says: 'The notes window: reaches the components, the wire and the schema.',
  },
  {
    name: '@numen/flashcards',
    at: 'modules/apps/desktop/flashcards',
    sources: ['src'],
    says: 'The review window: the same three, and never the other window.',
  },
]
