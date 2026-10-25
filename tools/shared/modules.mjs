/**
 * The interface modules, once, for every tool that reads them.
 *
 * There were two of these lists — the linter's and the graph's — and they had
 * drifted a module apart, which is how a rule stops covering something without
 * anybody being told. A module left off either list is a rule that stops at its
 * border, so there is one list and both tools take it whole.
 */
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

export const root = resolve(dirname(fileURLToPath(import.meta.url)), '../..')

/**
 * Each module by name, with:
 *
 * - `at`, the folder it is cruised from, so its own tsconfig answers for `@/`;
 * - `sources`, what the cruise starts at, relative to `at`;
 * - `written`, where its hand-written source stands, relative to the repository;
 * - `reads`, one file of it, relative to `at`, that a cruise has to have read —
 *   a count alone cannot say which tree was walked;
 * - `says`, the one line the drawing carries.
 */
export const modules = [
  {
    name: '@numen/ui',
    at: 'libs/ui',
    sources: ['src'],
    written: 'libs/ui/src',
    reads: 'src/index.ts',
    says: 'The shared components. Reaches nothing of ours.',
  },
  {
    name: '@numen/wire',
    at: 'libs/wire',
    sources: ['index.ts'],
    written: 'libs/wire',
    reads: 'index.ts',
    says: "What a window's ports are answered with.",
  },
  {
    name: '@numen/editor',
    at: 'apps/desktop/editor',
    sources: ['src'],
    written: 'apps/desktop/editor/src',
    reads: 'src/app/App.vue',
    says: 'The notes window: reaches the components, the wire and the schema.',
  },
  {
    name: '@numen/flashcards',
    at: 'apps/desktop/flashcards',
    sources: ['src'],
    written: 'apps/desktop/flashcards/src',
    reads: 'src/app/App.vue',
    says: 'The review window: the same three, and never the other window.',
  },
  {
    name: '@numen/mobile',
    at: 'apps/mobile',
    sources: ['src'],
    written: 'apps/mobile/src',
    reads: 'src/App.vue',
    says: 'The phone: one vault, the components and the schema.',
  },
]

/**
 * The packages under `libs` and `apps` that are none of this,
 * each with the reason. A package is here or it is above; the alternative is a
 * package nobody notices is uncovered.
 */
export const elsewhere = {
  'libs/protocol': 'the schema, generated from the .proto and hand-written nowhere',
  'apps/docs': 'an Astro site: pages and prose, and no application of ours',
  'apps/landing': 'the same',
}
