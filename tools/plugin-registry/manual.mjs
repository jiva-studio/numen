/**
 * The pages of the user manual (`apps/docs/src/content/docs`, published at docs.numen.md) that name
 * a plugin: by its manifest name or its id, in the text people wrote. The blocks
 * `apps/docs/tools/manual.mjs` generates (between the BEGIN AUTOGEN and END AUTOGEN marks) list
 * every plugin and are left out.
 */
import { existsSync, readFileSync, readdirSync } from 'node:fs'
import { join, relative, sep } from 'node:path'

/** The folder of the manual pages, under the repository root. */
export const MANUAL_DIR = join('apps', 'docs', 'src', 'content', 'docs')

/** Where the manual is published. */
export const MANUAL_URL = 'https://docs.numen.md'

const AUTOGEN = [
  /\{\/\* BEGIN AUTOGEN \*\/\}[\s\S]*?\{\/\* END AUTOGEN \*\/\}/g,
  /<!-- BEGIN AUTOGEN -->[\s\S]*?<!-- END AUTOGEN -->/g,
]

const PAGE = /\.mdx?$/

const listPages = (dir) =>
  readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const path = join(dir, entry.name)
    if (entry.isDirectory()) return listPages(path)
    return entry.isFile() && PAGE.test(entry.name) ? [path] : []
  })

const escapeRegExp = (text) => text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

const createNamePattern = (name) => new RegExp(`(?<![\\w.-])${escapeRegExp(name)}(?![\\w-])`)

/** The published address of a page file, relative to the manual folder. */
export const getManualUrl = (file) => {
  const slug = file.split(sep).join('/').replace(PAGE, '').replace(/(^|\/)index$/, '')
  return `${MANUAL_URL}/${slug === '' ? '' : `${slug}/`}`
}

/**
 * The addresses of the manual pages that name the plugin, ordered by address.
 *
 * @param {string} root - the repository root
 * @param {{ id: string, name: string }} plugin - the manifest id and name
 * @returns {string[] | null} null when the repository has no manual
 */
export const listManualPages = (root, plugin) => {
  const dir = join(root, MANUAL_DIR)
  if (!existsSync(dir)) return null
  const patterns = [plugin.id, plugin.name].filter(Boolean).map(createNamePattern)
  return listPages(dir)
    .filter((file) => {
      const text = AUTOGEN.reduce((rest, block) => rest.replace(block, ''), readFileSync(file, 'utf8'))
      return patterns.some((pattern) => pattern.test(text))
    })
    .map((file) => getManualUrl(relative(dir, file)))
    .sort()
}
