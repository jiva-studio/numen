/**
 * The lucide icons of the plugins, drawn as inline SVG at build time from the icon data of
 * `@lucide/vue`, which the editor and the plugins draw the same icons from.
 */
import { createRequire } from 'node:module'
import { pathToFileURL } from 'node:url'

/** The elements of an icon: a tag name and its attributes. */
type IconNode = [string, Record<string, string | number>][]

/** The icon drawn for a name lucide does not have. */
const FALLBACK_ICON = 'puzzle'

const require = createRequire(import.meta.url)
const loaded = new Map<string, Promise<IconNode | null>>()

const ESCAPES: Readonly<Record<string, string>> = {
  '&': '&amp;',
  '<': '&lt;',
  '>': '&gt;',
  '"': '&quot;',
}

const escapeAttribute = (value: string): string => value.replace(/[&<>"]/g, (one) => ESCAPES[one] ?? one)

const importIconNode = async (name: string): Promise<IconNode | null> => {
  if (!/^[a-z0-9-]+$/.test(name)) return null
  let file: string
  try {
    file = require.resolve(`@lucide/vue/dist/esm/icons/${name}.mjs`)
  } catch {
    return null
  }
  const module = (await import(/* @vite-ignore */ pathToFileURL(file).href)) as {
    __iconData?: { node?: IconNode }
  }
  return module.__iconData?.node ?? null
}

const readIconNode = (name: string): Promise<IconNode | null> => {
  let node = loaded.get(name)
  if (!node) {
    node = importIconNode(name)
    loaded.set(name, node)
  }
  return node
}

/** The inner SVG markup of a lucide icon, the fallback icon's for a name lucide does not have. */
export async function getIconMarkup(name: string): Promise<string> {
  const node = (await readIconNode(name)) ?? (await readIconNode(FALLBACK_ICON)) ?? []
  return node
    .map(([tag, attributes]) => {
      const pairs = Object.entries(attributes)
        .filter(([key]) => key !== 'key')
        .map(([key, value]) => `${key}="${escapeAttribute(String(value))}"`)
      return `<${tag} ${pairs.join(' ')}/>`
    })
    .join('')
}
