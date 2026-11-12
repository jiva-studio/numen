/**
 * The interface modules, once, for every tool that reads them.
 *
 * There were two of these lists — the linter's and the graph's — and they had
 * drifted a module apart, which is how a rule stops covering something without
 * any owner being notified. A module left off either list is a rule that stops at its
 * border, so there is one list and both tools take it whole.
 */
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

export const root = resolve(dirname(fileURLToPath(import.meta.url)), '../..')

/** A module cruised from `src` of its folder `at`, with the fields `modules` describes. */
const createModule = (name, at, reads, book, says) => ({
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
  createModule(
    '@numen/ui',
    'libs/ui/components',
    'src/index.ts',
    'ui',
    'The shared components. Reaches nothing of ours.',
  ),
  createModule(
    '@numen/workbench',
    'libs/ui/workbench',
    'src/index.ts',
    'shell',
    'The chrome of the desktop windows: reaches the components.',
  ),
  createModule(
    '@numen/plugin-sdk',
    'libs/plugin/sdk-ts',
    'src/index.ts',
    'ui',
    "The plugin contract and what a window's ports are answered with.",
  ),
  createModule(
    '@numen/editor',
    'apps/desktop/editor',
    'src/app/App.vue',
    'shell',
    'The notes window: reaches the components, the shell, the wire and the schema.',
  ),
  createModule(
    '@numen/flashcards',
    'apps/desktop/flashcards',
    'src/app/App.vue',
    'shell',
    'The review window: the same four, and never the other window.',
  ),
  createModule(
    '@numen/mobile',
    'apps/mobile',
    'src/App.vue',
    'ui',
    'The phone: one vault, the components and the schema.',
  ),
]

/**
 * The packages under `libs` and `apps` that are none of this,
 * each with the reason. A package is here or it is above; the alternative is a
 * package no one notices is uncovered.
 */
export const elsewhere = {
  'libs/core/protocol': 'the schema, generated from the .proto and hand-written nowhere',
  'apps/docs': 'an Astro site: pages and prose, and no application of ours',
  'apps/landing': 'the same',
}
