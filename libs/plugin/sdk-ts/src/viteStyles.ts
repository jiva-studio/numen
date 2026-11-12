/**
 * The Tailwind utilities of one plugin frontend, and the check that every class its templates
 * name is defined by a stylesheet the window loads for it.
 *
 * A plugin view is rendered under the window's stylesheets, which carry the build of `@numen/ui`, and
 * the plugin's own `dist/styles.css` after them: in the window's document, or in a shadow root
 * that holds a copy of the window's stylesheets. The utilities a plugin uses are built with the
 * theme of `@numen/ui` and appended to that file, less every declaration the `@numen/ui` build
 * already carries.
 */
import { readFileSync } from 'node:fs'
import { resolve, sep } from 'node:path'
import postcss from 'postcss'
import { parse as parseSfc } from '@vue/compiler-sfc'
import type { Node as BabelNode } from '@babel/types'
import type { Plugin } from 'vite'
import { UI_STYLES, removePresentRules } from '@numen/ui/vite'
import { getCandidates, getUtilityCss } from './utilityCss.ts'

function getDefinedClasses(css: string): Set<string> {
  const classes = new Set<string>()
  postcss.parse(css).walkRules((rule) => {
    for (const [, escaped = ''] of rule.selector.matchAll(
      /\.((?:[\w-]|\\[0-9a-fA-F]{1,6}\s?|\\.)+)/g,
    )) {
      classes.add(
        escaped
          .replace(/\\([0-9a-fA-F]{1,6})\s?/g, (_, hex: string) =>
            String.fromCodePoint(Number.parseInt(hex, 16)),
          )
          .replace(/\\(.)/g, '$1'),
      )
    }
  })
  return classes
}

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

type TemplateProp = NonNullable<TemplateNode['props']>[number]

const isClassAttribute = (prop: TemplateProp): boolean =>
  prop.type === ATTRIBUTE && prop.name === 'class'

const isClassBinding = (prop: TemplateProp): boolean =>
  prop.type === DIRECTIVE && prop.name === 'bind' && prop.arg?.content === 'class'

function getPropClasses(prop: TemplateProp): string[] {
  if (isClassAttribute(prop)) return (prop.value?.content ?? '').split(/\s+/)
  if (isClassBinding(prop)) return getBoundClasses(prop.exp?.ast || null)
  return []
}

function getTemplateClasses(source: string): string[] {
  const ast = parseSfc(source).descriptor.template?.ast as TemplateNode | undefined
  const classes: string[] = []
  const visit = (node: TemplateNode) => {
    for (const prop of node.props ?? []) classes.push(...getPropClasses(prop))
    for (const child of [...(node.children ?? []), ...(node.branches ?? [])]) visit(child)
  }
  if (ast) visit(ast)
  return classes.filter(Boolean)
}

async function getUtilityClasses(classes: readonly string[]): Promise<Set<string>> {
  const baseClasses = getDefinedClasses(await getUtilityCss([]))
  const utilityClasses = getDefinedClasses(await getUtilityCss(classes))
  return new Set(classes.filter((name) => utilityClasses.has(name) && !baseClasses.has(name)))
}

async function getUndefinedUtilities(
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

function getPluginFiles(root: string, moduleIds: Iterable<string>): string[] {
  const prefix = resolve(root) + sep
  const files = new Set<string>()
  for (const id of moduleIds) {
    const file = id.replace(/\?.*/s, '')
    if (file.startsWith(prefix) && !file.includes(`${sep}node_modules${sep}`)) files.add(file)
  }
  return [...files].filter((file) => /\.(vue|ts|tsx|js)$/.test(file))
}

/**
 * Appends the utilities the plugin's files use to `styles.css`, and fails the build when a
 * template names a utility that neither that file nor the build of `@numen/ui` defines. `root` is
 * the plugin's directory.
 */
export function createUtilitiesPlugin(root: string): Plugin {
  return {
    name: 'numen-utilities',
    apply: 'build',
    enforce: 'post',
    async generateBundle(_, bundle) {
      const files = getPluginFiles(root, this.getModuleIds())
      const ui = readFileSync(UI_STYLES, 'utf8')
      const utilities = removePresentRules(await getUtilityCss(getCandidates(files)), ui)
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
