/**
 * The build settings of a stylesheet loaded after the one of `@numen/ui`: the host's, and each
 * add-on's.
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

const getBrowserVersion = (major: number, minor = 0): number => (major << 16) | (minor << 8)

const TARGETS = {
  safari: getBrowserVersion(16, 4),
  chrome: getBrowserVersion(111),
  firefox: getBrowserVersion(128),
}

/** The stylesheet minified the way both sides of a comparison are. */
export function getNormalCss(css: string): string {
  return transform({
    // Stryker disable next-line StringLiteral: filename is only used by Lightning CSS for diagnostics
    filename: 'styles.css',
    code: Buffer.from(css),
    minify: true,
    targets: TARGETS,
    exclude: LIGHTNING_CSS.exclude,
    // Stryker disable next-line BooleanLiteral: valid CSS in tests does not trigger recovery
    errorRecovery: true,
  }).code.toString()
}

function getContext(node: ChildNode): string {
  // Stryker disable next-line ArrayDeclaration: constant key suffix on both sides of Set lookup is invariant
  const path: string[] = []
  for (
    let parent: Container | undefined = node.parent;
    // Stryker disable next-line ConditionalExpression,StringLiteral: root has no parent and matches neither rule nor atrule
    parent && parent.type !== 'root';
    parent = parent.parent as Container | undefined
  ) {
    if (parent.type === 'rule') {
      path.unshift((parent as Rule).selector)
    } else {
      path.unshift(`atrule ${(parent as AtRule).name} ${(parent as AtRule).params}`)
    }
  }
  // Stryker disable next-line StringLiteral: separator joins distinct context frames on both sides of Set lookup
  return path.join('\n')
}

function getKey(node: ChildNode): string | null {
  if (node.type === 'decl') return `${getContext(node)}\n${node.prop}:${node.value}`
  // Stryker disable next-line ConditionalExpression: getNormalCss strips comments and rules carry nodes
  if (node.type === 'atrule' && !node.nodes) {
    return `${getContext(node)}\natrule ${node.name} ${node.params}`
  }
  return null
}

function getKeys(root: Root): Set<string> {
  const keys = new Set<string>()
  root.walk((node) => {
    const key = getKey(node)
    // Stryker disable next-line ConditionalExpression: removePresentRules only queries truthy keys
    if (key) keys.add(key)
  })
  return keys
}

function removeEmptyRules(container: Container): void {
  // Stryker disable next-line ArrayDeclaration: Rule and AtRule containers with nodes always define container.nodes
  for (const node of [...(container.nodes ?? [])]) {
    // Stryker disable next-line ConditionalExpression: declarations and comments have no nodes property
    if (node.type !== 'rule' && node.type !== 'atrule') continue
    if (!node.nodes) continue
    removeEmptyRules(node)
    if (node.nodes.length === 0) node.remove()
  }
}

/** The stylesheet less every declaration `present` carries in the same place. */
export function removePresentRules(css: string, present: string): string {
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
  removeEmptyRules(root)
  // Stryker disable next-line MethodExpression,ConditionalExpression,StringLiteral: getNormalCss strips comments and returns empty string for an empty root
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
      // Stryker disable next-line StringLiteral: Buffer.from in getNormalCss accepts both UTF-8 strings and Buffers
      return { code: removePresentRules(code, readFileSync(UI_STYLES, 'utf8')), map: null }
    },
  }
}
