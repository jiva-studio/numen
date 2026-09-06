/**
 * Every package that runs stories, the preview that runs them, and the stories
 * it runs. The keyboard walk is a rule over exactly this list: a package that
 * draws stories and is not on it is a corner of the tree nothing walks.
 */
import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative } from 'node:path'
import { modules, root } from './source.mjs'

function walk(at, found) {
  if (!existsSync(at)) return found
  for (const name of readdirSync(at)) {
    if (name === 'node_modules') continue
    const path = join(at, name)
    if (statSync(path).isDirectory()) walk(path, found)
    else if (path.endsWith('.stories.ts')) found.push(path)
  }
  return found
}

/** A title or an export name as Storybook spells it in an address. */
const kebab = (said) =>
  said
    .replace(/([a-z0-9])([A-Z])/g, '$1 $2')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-|-$/g, '')

/** How Storybook addresses one story: the title it stands under, then its export. */
export const idOf = (title, name) => `${kebab(title)}--${kebab(name)}`

/**
 * Every story a file stages, by the address Storybook gives it. The title is
 * the meta's own, and not the first thing in the file to be called a title: a
 * story staging a window writes down the title of every tab in it.
 */
function addressed(text) {
  const title = /^const meta[^=]*=\s*\{\s*\n?\s*title:\s*'([^']+)'/m.exec(text)
  if (!title) return []
  return [...text.matchAll(/^export const (\w+)/gm)].map((one) => idOf(title[1], one[1]))
}

/**
 * The packages that run stories: what the preview is, what the story files are,
 * and every address among them. A package holding no story is not one of these
 * and is owed no walk.
 */
export function corpora() {
  const found = []
  for (const { name, at } of modules) {
    const stories = walk(join(root, at), [])
    if (stories.length === 0) continue
    const ids = new Set(stories.flatMap((path) => addressed(readFileSync(path, 'utf8'))))
    found.push({
      name,
      preview: join(at, '.storybook/preview.ts'),
      stories: stories.map((path) => relative(root, path)),
      ids,
    })
  }
  return found
}
