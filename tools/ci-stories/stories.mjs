/**
 * Every story the tree declares, under the id Storybook addresses it by, and the Storybook that
 * draws it.
 */
import { readdir, readFile } from 'node:fs/promises'
import { join } from 'node:path'
import { modules, root } from '../shared/modules.mjs'

/**
 * The Storybooks a story is drawn in, each by the folder it is started from:
 * the components' own, the shell's, the editor window's, and the plugins',
 * which draws every plugin's frontend.
 */
export const BOOKS = {
  ui: 'libs/ui/components',
  shell: 'libs/ui/workbench',
  editor: 'apps/desktop/editor',
  plugins: 'libs/plugin/sdk-ts/stories',
}

/**
 * Everywhere a story can be written, and the Storybook that draws it: the
 * hand-written source of every module the repository has, and the frontend of
 * every plugin.
 */
const ROOTS = [
  ...modules.map((one) => ({ at: join(root, one.written), book: one.book, only: /./ })),
  { at: join(root, 'plugins'), book: 'plugins', only: /^[^/]+\/frontend\// },
]

/** The stories declared under `at`, counted by the same declaration `stories` reads. */
export const countStories = async (at) => {
  const files = (await readdir(at, { recursive: true })).filter(
    (name) => name.endsWith('.stories.ts') && !name.includes('node_modules'),
  )
  let count = 0
  for (const file of files) {
    const source = await readFile(join(at, file), 'utf8')
    count += [...source.matchAll(/^export const [A-Z][A-Za-z0-9]*\s*:/gm)].length
  }
  return count
}

/** Every story file of the tree, by absolute path, with the Storybook that draws it. */
export const listStoryFiles = async () => {
  const found = []
  for (const { at, book, only } of ROOTS) {
    const files = (await readdir(at, { recursive: true })).filter(
      (name) => name.endsWith('.stories.ts') && only.test(name) && !name.includes('node_modules'),
    )
    for (const file of files) found.push({ path: join(at, file), book })
  }
  return found
}

/**
 * The id Storybook addresses a story by: the meta's title and the export's name, each cut to
 * lower case and dashes.
 */
export const getStoryId = (title, exportName) => {
  const under = title.toLowerCase().replace(/[^a-z0-9]+/g, '-')
  const words = exportName
    .replace(/([a-z0-9])([A-Z])/g, '$1-$2')
    .replace(/([A-Z])([A-Z][a-z])/g, '$1-$2')
  return `${under}--${words.toLowerCase()}`
}

/**
 * Every story id the tree declares against the Storybook it is drawn in. An id two Storybooks
 * declare is drawn in `both`, which names neither.
 */
export const stories = async () => {
  const found = new Map()
  for (const { path, book } of await listStoryFiles()) {
    const source = await readFile(path, 'utf8')
    const meta = source.match(/const meta[^=]*=\s*\{[\s\S]{0,200}?title:\s*'([^']+)'/)
    if (!meta) continue
    for (const [, name] of source.matchAll(/^export const ([A-Z][A-Za-z0-9]*)\s*:/gm)) {
      const id = getStoryId(meta[1], name)
      const before = found.get(id)
      found.set(id, before && before !== book ? 'both' : book)
    }
  }
  return found
}
