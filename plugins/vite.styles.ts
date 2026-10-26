/**
 * The Tailwind utilities of one plugin frontend, and the check that every class its templates
 * name is defined by a stylesheet the window loads for it.
 *
 * A plugin view is drawn under the window's stylesheets, which carry the build of `@numen/ui`, and
 * the plugin's own `dist/styles.css` after them: in the window's document, or in a shadow root
 * that holds a copy of the window's stylesheets. The utilities a plugin uses are built with the
 * theme of `@numen/ui` and appended to that file, less every declaration the `@numen/ui` build
 * already carries.
 */
import { readFileSync, readdirSync } from 'node:fs'
import { dirname, extname, resolve, sep } from 'node:path'
import { compile } from '@tailwindcss/node'
import { Scanner } from '@tailwindcss/oxide'
import { Features, transform } from 'lightningcss'
import postcss, { type AtRule, type ChildNode, type Container, type Root } from 'postcss'
import { parse as parseSfc } from '@vue/compiler-sfc'
import type { Node as BabelNode } from '@babel/types'
import type { Plugin } from 'vite'

const modules = resolve(import.meta.dirname, '..')

/** The stylesheet of `@numen/ui` the utilities are built from. */
const THEME = resolve(modules, 'libs/ui/core/src/shared/tokens/theme.css')

/** The build of `@numen/ui` the window loads ahead of every plugin's stylesheet. */
export const UI_STYLES = resolve(modules, 'libs/ui/core/dist/numen-ui.css')

/**
 * The Lightning CSS options of a plugin's stylesheet, the ones the windows build theirs with.
 * `light-dark()` stays as written: the window chooses a half with `color-scheme`, which the
 * lowered form does not read.
 */
export const LIGHTNING_CSS = { exclude: Features.LightDark } as const

const TARGETS = { safari: (16 << 16) | (4 << 8), chrome: 111 << 16, firefox: 128 << 16 }

/** The stylesheet minified the way both sides of a comparison are. */
function getNormalCss(css: string): string {
  return transform({
    filename: 'styles.css',
    code: Buffer.from(css),
    minify: true,
    targets: TARGETS,
    exclude: LIGHTNING_CSS.exclude,
    errorRecovery: true,
  }).code.toString()
}

/** The candidates Tailwind finds in the templates and scripts of these files. */
export function getCandidates(files: readonly string[]): string[] {
  const scanner = new Scanner({})
  return scanner.scanFiles(
    files.map((file) => {
      const content = readFileSync(file, 'utf8')
      const extension = extname(file).slice(1)
      return {
        content: extension === 'vue' ? content.replace(/<style[\s\S]*?<\/style>/g, '') : content,
        extension,
      }
    }),
  )
}

/** The utilities of these candidates, built with the theme of `@numen/ui`. */
export async function getUtilityCss(candidates: readonly string[]): Promise<string> {
  const compiler = await compile(`@import ${JSON.stringify(THEME)};`, {
    base: dirname(THEME),
    onDependency: () => {},
  })
  return getNormalCss(compiler.build([...candidates]))
}

/** The at-rules and the selector a node is nested in, from the outside in. */
function getContext(node: ChildNode): string {
  const path: string[] = []
  for (
    let parent: Container | undefined = node.parent;
    parent && parent.type !== 'root';
    parent = parent.parent
  ) {
    if (parent.type === 'rule') path.unshift((parent as postcss.Rule).selector)
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
  const root = postcss.parse(css)
  root.walk((node) => {
    const key = getKey(node)
    if (key && keys.has(key)) node.remove()
  })
  // The order of the layers is the one the window's stylesheets declare ahead of this one.
  root.walkAtRules('layer', (rule) => {
    if (!rule.nodes) rule.remove()
  })
  removeEmpty(root)
  return root.nodes.some((node) => node.type !== 'comment') ? getNormalCss(root.toString()) : ''
}

/** Every class a selector of the stylesheet names. */
export function getDefinedClasses(css: string): Set<string> {
  const classes = new Set<string>()
  postcss.parse(css).walkRules((rule) => {
    for (const [, escaped] of rule.selector.matchAll(/\.((?:[\w-]|\\[0-9a-fA-F]{1,6}\s?|\\.)+)/g)) {
      classes.add(
        escaped!
          .replace(/\\([0-9a-fA-F]{1,6})\s?/g, (_, hex: string) =>
            String.fromCodePoint(parseInt(hex, 16)),
          )
          .replace(/\\(.)/g, '$1'),
      )
    }
  })
  return classes
}

/** The literal class names an expression bound to `class` can evaluate to. */
function getBoundClasses(node: BabelNode | null | undefined): string[] {
  if (!node) return []
  switch (node.type) {
    case 'StringLiteral':
      return node.value.split(/\s+/)
    case 'TemplateLiteral':
      return node.quasis.flatMap((quasi, index) => {
        const words = (quasi.value.cooked ?? '').split(/\s+/)
        // A word touching an interpolation is only part of a class name.
        if (index > 0) words.shift()
        if (index < node.quasis.length - 1) words.pop()
        return words
      })
    case 'ArrayExpression':
      return node.elements.flatMap((element) => getBoundClasses(element))
    case 'ConditionalExpression':
      return [...getBoundClasses(node.consequent), ...getBoundClasses(node.alternate)]
    case 'LogicalExpression':
      return node.operator === '&&'
        ? getBoundClasses(node.right)
        : [...getBoundClasses(node.left), ...getBoundClasses(node.right)]
    case 'ObjectExpression':
      return node.properties.flatMap((property) => {
        if (property.type !== 'ObjectProperty' || property.computed) return []
        if (property.key.type === 'Identifier') return [property.key.name]
        if (property.key.type === 'StringLiteral') return property.key.value.split(/\s+/)
        return []
      })
    case 'CallExpression':
      return node.arguments.flatMap((argument) => getBoundClasses(argument))
    default:
      return []
  }
}

interface TemplateNode {
  readonly type: number
  readonly props?: readonly {
    readonly type: number
    readonly name: string
    readonly value?: { readonly content: string }
    readonly arg?: { readonly content?: string }
    readonly exp?: { readonly ast?: BabelNode | null | false }
  }[]
  readonly children?: readonly TemplateNode[]
  readonly branches?: readonly TemplateNode[]
}

const ATTRIBUTE = 6
const DIRECTIVE = 7

/** The class names one attribute or `:class` binding of an element names. */
function getPropClasses(prop: NonNullable<TemplateNode['props']>[number]): string[] {
  if (prop.type === ATTRIBUTE && prop.name === 'class') {
    return (prop.value?.content ?? '').split(/\s+/)
  }
  if (prop.type === DIRECTIVE && prop.name === 'bind' && prop.arg?.content === 'class') {
    return getBoundClasses(prop.exp?.ast || null)
  }
  return []
}

/** Every literal class name the template of a single-file component names. */
export function getTemplateClasses(source: string): string[] {
  const ast = parseSfc(source).descriptor.template?.ast as TemplateNode | undefined
  const classes: string[] = []
  const visit = (node: TemplateNode) => {
    for (const prop of node.props ?? []) classes.push(...getPropClasses(prop))
    for (const child of [...(node.children ?? []), ...(node.branches ?? [])]) visit(child)
  }
  if (ast) visit(ast)
  return classes.filter(Boolean)
}

/** The names among these classes that the theme of `@numen/ui` builds a utility for. */
export async function getUtilityClasses(classes: readonly string[]): Promise<Set<string>> {
  const baseClasses = getDefinedClasses(await getUtilityCss([]))
  const utilityClasses = getDefinedClasses(await getUtilityCss(classes))
  return new Set(classes.filter((name) => utilityClasses.has(name) && !baseClasses.has(name)))
}

/**
 * The utilities the templates of these files name that no stylesheet defines, by file. A class
 * Tailwind builds nothing for is a name the plugin's own rules or its tests select by, and is
 * left to them.
 */
export async function getUndefinedUtilities(
  files: readonly string[],
  stylesheets: readonly string[],
): Promise<Map<string, string[]>> {
  const defined = new Set(stylesheets.flatMap((css) => [...getDefinedClasses(css)]))
  const classesByFile = new Map(
    files
      .filter((file) => file.endsWith('.vue'))
      .map((file) => [file, [...new Set(getTemplateClasses(readFileSync(file, 'utf8')))]]),
  )
  const utilities = await getUtilityClasses([...new Set([...classesByFile.values()].flat())])
  const missing = new Map<string, string[]>()
  for (const [file, classes] of classesByFile) {
    const absent = classes.filter((name) => utilities.has(name) && !defined.has(name))
    if (absent.length) missing.set(file, absent)
  }
  return missing
}

/** The source files of the bundle that belong to the plugins. */
function getPluginFiles(moduleIds: Iterable<string>): string[] {
  const plugins = import.meta.dirname + sep
  const files = new Set<string>()
  for (const id of moduleIds) {
    const file = id.split('?')[0]!
    if (file.startsWith(plugins) && !file.includes(`${sep}node_modules${sep}`)) files.add(file)
  }
  return [...files].filter((file) => /\.(vue|ts|tsx|js)$/.test(file))
}

/**
 * Appends the utilities the plugin's files use to `styles.css`, and fails the build when a
 * template names a utility that neither that file nor the build of `@numen/ui` defines.
 */
export function createUtilitiesPlugin(): Plugin {
  return {
    name: 'numen-utilities',
    apply: 'build',
    enforce: 'post',
    async generateBundle(_, bundle) {
      const files = getPluginFiles(this.getModuleIds())
      const ui = readFileSync(UI_STYLES, 'utf8')
      const utilities = removePresent(await getUtilityCss(getCandidates(files)), ui)
      const asset = bundle['styles.css']
      const styles = asset?.type === 'asset' ? asset : undefined
      const componentCss = styles ? String(styles.source) : ''
      const css = [componentCss, utilities].filter(Boolean).join('\n')
      if (styles) styles.source = css
      else if (css) this.emitFile({ type: 'asset', fileName: 'styles.css', source: css })

      const missing = await getUndefinedUtilities(files, [ui, css])
      if (missing.size) {
        const lines = [...missing].map(([file, classes]) => `  ${file}: ${classes.join(' ')}`)
        this.error(`utilities no stylesheet defines:\n${lines.join('\n')}`)
      }
    },
  }
}

/** The id the stories import the utilities of every plugin frontend by. */
const STORY_UTILITIES = 'virtual:plugin-utilities.css'

/** Every source file of every plugin frontend, stories and tests left out. */
function getFrontendFiles(): string[] {
  return readdirSync(import.meta.dirname, { recursive: true, encoding: 'utf8' })
    .filter((path) => /^[^/]+\/frontend\/.+\.(vue|ts)$/.test(path.split(sep).join('/')))
    .filter((path) => !/\.(stories|test)\.ts$/.test(path))
    .map((path) => resolve(import.meta.dirname, path))
}

/**
 * Serves `virtual:plugin-utilities.css` to the stories: the utilities every plugin frontend uses
 * that the build of `@numen/ui` does not carry, as each plugin's `dist/styles.css` carries them.
 */
export function createStoryUtilitiesPlugin(): Plugin {
  return {
    name: 'numen-story-utilities',
    resolveId: (id) => (id === STORY_UTILITIES ? `\0${STORY_UTILITIES}` : undefined),
    async load(id) {
      if (id !== `\0${STORY_UTILITIES}`) return undefined
      const css = await getUtilityCss(getCandidates(getFrontendFiles()))
      return removePresent(css, readFileSync(UI_STYLES, 'utf8'))
    },
  }
}
