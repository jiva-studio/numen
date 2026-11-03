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

/** A module cruised from `src` of its folder `at`. */
const createModule = ({ name, at, reads, says, book }) => ({
  name,
  at,
  sources: ['src'],
  written: `${at}/src`,
  reads,
  says,
  book,
})

/**
 * Each module by name, with:
 *
 * - `at`, the folder it is cruised from, so its own tsconfig answers for `@/`;
 * - `sources`, what the cruise starts at, relative to `at`;
 * - `written`, where its hand-written source stands, relative to the repository;
 * - `reads`, one file of it, relative to `at`, that a cruise has to have read —
 *   a count alone cannot say which tree was walked;
 * - `says`, the one line the drawing carries;
 * - `book`, the Storybook its stories are drawn in, as `ci-stories` names them.
 */
export const modules = [
  createModule({ name: '@numen/ui', at: 'libs/ui/core', reads: 'src/index.ts', book: 'ui',
    says: 'The shared components. Reaches nothing of ours.' }),
  createModule({ name: '@numen/shell', at: 'libs/ui/shell', reads: 'src/index.ts', book: 'shell',
    says: 'The chrome of the desktop windows: reaches the components.' }),
  createModule({ name: '@numen/plugin-sdk', at: 'libs/plugin-sdk-ts', reads: 'src/index.ts', book: 'ui',
    says: "The plugin contract and what a window's ports are answered with." }),
  createModule({ name: '@numen/editor', at: 'apps/desktop/editor', reads: 'src/app/App.vue', book: 'shell',
    says: 'The notes window: reaches the components, the shell, the wire and the schema.' }),
  createModule({ name: '@numen/flashcards', at: 'apps/desktop/flashcards', reads: 'src/app/App.vue', book: 'shell',
    says: 'The review window: the same four, and never the other window.' }),
  createModule({ name: '@numen/mobile', at: 'apps/mobile', reads: 'src/App.vue', book: 'ui',
    says: 'The phone: one vault, the components and the schema.' }),
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
