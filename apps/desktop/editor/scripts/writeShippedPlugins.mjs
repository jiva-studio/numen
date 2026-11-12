/**
 * Writes src/pages/settings/testing/shippedPlugins.ts from plugins/<name>/manifest.json and
 * README.md: the plugins, their catalog entries and their READMEs as the host serves them, laid out
 * the way Prettier lays them out. The contributions follow the rules of ListContributions in
 * libs/core/kernel/domain/plugin/contribution.go.
 *
 * Usage: node apps/desktop/editor/scripts/writeShippedPlugins.mjs
 */
import { existsSync, readdirSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const EDITOR = join(dirname(fileURLToPath(import.meta.url)), '..')
const PLUGINS = join(EDITOR, '../../../plugins')
const OUTPUT = join(EDITOR, 'src/pages/settings/testing/shippedPlugins.ts')
const WIDTH = 100

const item = (text, detail = '', keys = '') => ({ text, detail, keys })
const titled = (title, id) => item(title || id, id)

const listCommands = (contributes) => {
  const keys = new Map()
  for (const one of contributes.keybindings ?? []) {
    if (!keys.has(one.command)) keys.set(one.command, one.key)
  }
  return (contributes.commands ?? []).map((one) =>
    item(one.title || one.id, one.id, keys.get(one.id) ?? ''),
  )
}

const listForeignKeybindings = (contributes) => {
  const declared = new Set((contributes.commands ?? []).map((one) => one.id))
  return (contributes.keybindings ?? [])
    .filter((one) => !declared.has(one.command))
    .map((one) => item(one.command, '', one.key))
}

const getSlotTarget = (one) => one.command || one.view || one.submenu || ''

const getListenerPort = (one) => one.portSetting || (one.port ? String(one.port) : '')

/** The list a manifest holds under a key, each entry made an item; a list it does not hold is empty. */
const listItems = (source, key, toItem) => (source[key] ?? []).map(toItem)

/** The kinds of contributes, in the order a page lists them, each with how an entry becomes an item. */
const CONTRIBUTION_KINDS = [
  ['tabKinds', (one) => titled(one.title, one.kind)],
  ['readers', (one) => item(one.id, (one.contentTypes ?? []).join(', '))],
  ['handlers', (one) => titled(one.title, one.command)],
  ['triggers', (one) => item(one.event, one.command)],
  ['textKinds', (one) => titled(one.title, one.kind)],
  ['noteTypes', (one) => item(one.type)],
  ['themes', (one) => titled(one.name, one.id)],
  ['slots', (one) => item(one.slot, getSlotTarget(one))],
  ['listeners', (one) => item(one.name, getListenerPort(one))],
  ['viewCapabilities', (one) => item(one)],
  ['dropHandlers', (one) => item((one.types ?? []).join(', '))],
  ['frameOrigins', (one) => item(one)],
  ['connectionTypes', (one) => titled(one.title, one.id)],
  ['operations', (one) => titled(one.title, one.name)],
]

/** The lists at the top of a manifest that a page shows with the contributions. */
const MANIFEST_KINDS = [
  ['permissions', (one) => item(one)],
  ['dependencies', (one) => item(one.id, one.kind ?? '')],
  ['services', (one) => item(one)],
  ['capabilities', (one) => item(one)],
  ['activationEvents', (one) => item(one)],
]

const listContributions = (manifest) => {
  const c = manifest.contributes ?? {}
  const sections = [
    ['commands', listCommands(c)],
    ['keybindings', listForeignKeybindings(c)],
    ...CONTRIBUTION_KINDS.map(([kind, toItem]) => [kind, listItems(c, kind, toItem)]),
    ...MANIFEST_KINDS.map(([kind, toItem]) => [kind, listItems(manifest, kind, toItem)]),
  ]
  return sections.filter(([, items]) => items.length > 0).map(([kind, items]) => ({ kind, items }))
}

const readOption = (option) =>
  typeof option === 'object'
    ? {
        value: String(option.value),
        title: option.title ?? String(option.value),
        detail: option.detail ?? '',
        ...(option.when ? { when: option.when } : {}),
      }
    : { value: String(option), title: String(option), detail: '' }

const OPTIONAL_FIELDS = ['palette', 'icon', 'when', 'accept', 'multiple', 'min', 'max', 'step']
const MORE_FIELDS = ['preview', 'optionsFrom', 'operations']

const readSetting = (setting) => ({
  key: setting.key,
  type: setting.type,
  title: setting.title,
  description: setting.description ?? '',
  default: setting.default,
  ...(setting.options ? { options: setting.options.map(readOption) } : {}),
  ...(setting.optionsBy
    ? {
        optionsBy: {
          setting: setting.optionsBy.setting,
          options: Object.fromEntries(
            Object.entries(setting.optionsBy.options).map(([value, list]) => [
              value,
              list.map(readOption),
            ]),
          ),
        },
      }
    : {}),
  ...Object.fromEntries(
    [...OPTIONAL_FIELDS, ...MORE_FIELDS]
      .filter((field) => setting[field] !== undefined)
      .map((field) => [field, setting[field]]),
  ),
})

const readPlugin = (manifest) => ({
  id: manifest.id,
  name: manifest.name,
  version: manifest.version,
  description: manifest.description ?? '',
  isActive: true,
  hasFrontend: manifest.frontend !== undefined,
  hasBackend: manifest.backend !== undefined,
  capabilities: manifest.capabilities ?? [],
  settings: (manifest.settings ?? []).map(readSetting),
  settingsViews: manifest.settingsViews ?? [],
  contributions: listContributions(manifest),
})

const readCatalogEntry = (manifest) => ({
  id: manifest.id,
  name: manifest.name,
  version: manifest.version,
  description: manifest.description ?? '',
  category: manifest.contributes?.category ?? '',
  feature: manifest.contributes?.feature ?? '',
  tabKinds: (manifest.contributes?.tabKinds ?? []).map((one) => one.kind),
  isAvailable: false,
})

/* The layout of Prettier with singleQuote, no semicolons, trailing commas and a width of 100. */

const formatString = (text) => {
  const singles = (text.match(/'/g) ?? []).length
  const doubles = (text.match(/"/g) ?? []).length
  const quote = singles > doubles ? '"' : "'"
  const escaped = text
    .replace(/\\/g, '\\\\')
    .replace(/\n/g, '\\n')
    .replace(/\r/g, '\\r')
    .replace(/\t/g, '\\t')
    .replaceAll(quote, `\\${quote}`)
  return `${quote}${escaped}${quote}`
}

const formatKey = (key) => (/^[A-Za-z_$][\w$]*$/.test(key) ? key : formatString(key))

const isPrimitive = (value) => value === null || typeof value !== 'object'

/** The value as source text, given the indent of its line and the column it starts at. */
const formatValue = (value, indent, column) => {
  if (value === undefined) return 'undefined'
  if (value === null) return 'null'
  if (typeof value === 'string') return formatString(value)
  if (typeof value !== 'object') return String(value)
  if (Array.isArray(value)) return formatArray(value, indent, column)
  return formatObject(value, indent)
}

const formatArray = (list, indent, column) => {
  if (list.length === 0) return '[]'
  const inner = `${indent}  `
  const flat = `[${list.map((one) => formatValue(one, inner, 0)).join(', ')}]`
  if (list.every(isPrimitive) && column + flat.length + 1 <= WIDTH) return flat
  const lines = list.map((one) => `${inner}${formatValue(one, inner, inner.length)},\n`)
  return `[\n${lines.join('')}${indent}]`
}

const formatObject = (object, indent) => {
  const entries = Object.entries(object)
  if (entries.length === 0) return '{}'
  const inner = `${indent}  `
  const lines = entries.map(([key, value]) => {
    const name = formatKey(key)
    return `${inner}${name}: ${formatValue(value, inner, inner.length + name.length + 2)},\n`
  })
  return `{\n${lines.join('')}${indent}}`
}

const plugins = []
const catalog = []
const readmes = {}
for (const name of readdirSync(PLUGINS).sort()) {
  const file = join(PLUGINS, name, 'manifest.json')
  if (!existsSync(file)) continue
  const manifest = JSON.parse(readFileSync(file, 'utf8'))
  plugins.push(readPlugin(manifest))
  catalog.push(readCatalogEntry(manifest))
  const readme = join(PLUGINS, name, 'README.md')
  if (existsSync(readme)) readmes[manifest.id] = readFileSync(readme, 'utf8')
}

writeFileSync(
  OUTPUT,
  `/**
 * The plugins of the repository as an installation lists them, written from the manifest.json and
 * README.md of every folder under plugins by apps/desktop/editor/scripts/writeShippedPlugins.mjs.
 */
import type { CatalogEntry } from '@/entities/plugin'
import type { PluginInfo } from '@/entities/settings'

export const SHIPPED_PLUGINS: readonly PluginInfo[] = ${formatValue(plugins, '', 0)}

export const SHIPPED_CATALOG: readonly CatalogEntry[] = ${formatValue(catalog, '', 0)}

export const SHIPPED_READMES: Readonly<Record<string, string>> = ${formatValue(readmes, '', 0)}
`,
)
