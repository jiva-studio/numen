/**
 * The build settings of a stylesheet loaded after the one of `@numen/ui`: the shell's, and each
 * plugin's.
 *
 * A stylesheet built with the theme of `@numen/ui` repeats every utility its sources name. A
 * repeated rule loaded later wins over the rules of `@numen/ui` that override it, so `.prose`
 * repeated by a later stylesheet takes back the colours `.prose-numen` gives it. Such a stylesheet
 * is built less every declaration the build of `@numen/ui` carries.
 */
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { Features, transform } from 'lightningcss'
import postcss, { type AtRule, type ChildNode, type Container, type Root, type Rule } from 'postcss'
import type { Plugin } from 'vite'

/** The build of `@numen/ui` every window loads ahead of the other stylesheets. */
export const UI_STYLES = fileURLToPath(new URL('../dist/numen-ui.css', import.meta.url))

/**
 * The Lightning CSS options of every stylesheet. `light-dark()` stays as written: the window
 * chooses a half with `color-scheme`, which the lowered form does not read.
 */
export const LIGHTNING_CSS = { exclude: Features.LightDark } as const

const TARGETS = { safari: (16 << 16) | (4 << 8), chrome: 111 << 16, firefox: 128 << 16 }

/** The stylesheet minified the way both sides of a comparison are. */
export function getNormalCss(css: string): string {
  return transform({
    filename: 'styles.css',
    code: Buffer.from(css),
    minify: true,
    targets: TARGETS,
    exclude: LIGHTNING_CSS.exclude,
    errorRecovery: true,
  }).code.toString()
}

/** The at-rules and the selector a node is nested in, from the outside in. */
function getContext(node: ChildNode): string {
  const path: string[] = []
  for (
    let parent: Container | undefined = node.parent;
    parent && parent.type !== 'root';
    parent = parent.parent as Container | undefined
  ) {
    if (parent.type === 'rule') path.unshift((parent as Rule).selector)
    if (parent.type === 'atrule')
      path.unshift(`atrule ${(parent as AtRule).name} ${(parent as AtRule).params}`)
  }
  return path.join('\n')
}

/** The key a declaration, or an at-rule without a block, is compared by. */
function getKey(node: ChildNode): string | null {
  if (node.type === 'decl') return `${getContext(node)}\n${node.prop}:${node.value}`
  if (node.type === 'atrule' && !node.nodes) {
    return `${getContext(node)}\natrule ${node.name} ${node.params}`
  }
  return null
}

function getKeys(root: Root): Set<string> {
  const keys = new Set<string>()
  root.walk((node) => {
    const key = getKey(node)
    if (key) keys.add(key)
  })
  return keys
}

/** Removes the rules and at-rules left without a declaration. */
function removeEmpty(container: Container): void {
  for (const node of [...(container.nodes ?? [])]) {
    if (node.type !== 'rule' && node.type !== 'atrule') continue
    if (!node.nodes) continue
    removeEmpty(node)
    if (node.nodes.length === 0) node.remove()
  }
}

/** The stylesheet less every declaration `present` carries in the same place. */
export function removePresent(css: string, present: string): string {
  const keys = getKeys(postcss.parse(getNormalCss(present)))
  const root = postcss.parse(getNormalCss(css))
  root.walk((node) => {
    const key = getKey(node)
    if (key && keys.has(key)) node.remove()
  })
  // The order of the layers is the one the stylesheet of `@numen/ui` declares ahead of this one.
  root.walkAtRules('layer', (rule) => {
    if (!rule.nodes) rule.remove()
  })
  removeEmpty(root)
  return root.nodes.some((node) => node.type !== 'comment') ? getNormalCss(root.toString()) : ''
}

/**
 * Builds the stylesheet at `file` less every declaration the build of `@numen/ui` carries, in the
 * dev server and in the build.
 */
export function createLayeredStylesPlugin(file: string): Plugin {
  return {
    name: 'numen-layered-styles',
    transform(code, id) {
      if (id.split('?')[0] !== file) return undefined
      return { code: removePresent(code, readFileSync(UI_STYLES, 'utf8')), map: null }
    },
  }
}
