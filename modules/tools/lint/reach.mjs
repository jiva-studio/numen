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
export function corpora(over = modules, under = root) {
  const found = []
  for (const { name, at } of over) {
    const stories = walk(join(under, at), [])
    if (stories.length === 0) continue
    const ids = new Set(stories.flatMap((path) => addressed(readFileSync(path, 'utf8'))))
    found.push({
      name,
      preview: join(at, '.storybook/preview.ts'),
      stories: stories.map((path) => relative(under, path)),
      ids,
    })
  }
  return found
}

/** Where the one walk is written, and what a preview reaches it by. */
export const CHECK = 'modules/libs/ui/.storybook/check.ts'

const read = (under, at) => (existsSync(join(under, at)) ? readFileSync(join(under, at), 'utf8') : '')

/** What is wrong with how one package runs the walk after each of its stories. */
export function unwalked({ name, preview, stories }, under = root) {
  if (!existsSync(join(under, preview))) {
    return [`${name} draws ${stories.length} stories and has no ${preview}`]
  }
  const text = read(under, preview)
  if (!/afterEach/.test(text)) return [`${preview} runs nothing after a story`]
  if (!/afterEach:\s*reachCheck\(/.test(text)) {
    return [`${preview} does not walk the keyboard through the story it has just drawn`]
  }
  return []
}

/** What is wrong with the one walk, and with how each preview reaches it. */
export function unshared(found, under = root, check = CHECK) {
  const wrong = []
  const text = read(under, check)
  if (!/await walk\(\)/.test(text)) wrong.push(`${check} does not walk the keyboard`)
  if (!/faults\(/.test(text)) {
    wrong.push(`${check} walks the keyboard and makes nothing of what it finds`)
  }
  for (const { preview } of found) {
    const to = relative(join(under, preview, '..'), join(under, check)).replace(/\.ts$/, '')
    const at = to.startsWith('.') ? to : `./${to}`
    if (!read(under, preview).includes(`from '${at}'`)) {
      wrong.push(`${preview} does not take the walk from ${check}`)
    }
  }
  return wrong
}

/** What is wrong with the story one package proves its walk against. */
export function unproved({ name, preview, ids }, under = root) {
  const proof = /story:\s*'([^']+)',\s*stops:\s*(\d+)/.exec(read(under, preview))
  if (!proof) return [`${preview} names no story to prove its walk against`]
  if (!ids.has(proof[1])) {
    return [`${preview} is proved against ${proof[1]}, which is no story of ${name}`]
  }
  if (!(Number(proof[2]) > 0)) {
    return [`${preview} asks its walk to find no stops, which anything does`]
  }
  return []
}

/**
 * What is wrong with the stories found, held against `wanted`: for each package
 * the fewest story files it holds, and the ones it holds by name.
 */
export function unstaged(found, wanted) {
  const held = new Map(found.map((one) => [one.name, one.stories]))
  const wrong = []
  for (const [name, [least, named]] of Object.entries(wanted)) {
    const stories = held.get(name) ?? []
    if (stories.length < least) {
      wrong.push(`${stories.length} story files found under ${name}: the walk is not reading them`)
    }
    for (const one of named) {
      if (!stories.includes(one)) wrong.push(`no ${one}, which the keyboard rule was written for`)
    }
  }
  return wrong
}
