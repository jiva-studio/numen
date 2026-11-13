/**
 * The plugins of the repository as an installation lists them, read from the manifest.json and
 * README.md of every folder under plugins when a story loads. The contributions follow the rules of
 * ListContributions in libs/core/kernel/domain/plugin/contribution.go.
 */
import type { CatalogEntry } from '@/entities/plugin'
import type { PluginInfo } from '@/entities/settings'
import type { JsonObject, JsonValue } from '@/shared/api/json'
import type {
  PluginContribution,
  PluginContributionItem,
  PluginContributionKind,
  PluginSettingInfo,
  PluginSettingOption,
} from '@numen/plugin-manager'

type Entry = JsonObject

type ManifestOptionEntry = {
  readonly value: string | number
  readonly title?: string
  readonly detail?: string
  readonly when?: string
}

type ManifestOption = string | number | ManifestOptionEntry

type ManifestSetting = Entry & {
  readonly key: string
  readonly type: PluginSettingInfo['type']
  readonly title: string
  readonly description?: string
  readonly default?: JsonValue
  readonly options?: readonly ManifestOption[]
  readonly optionsBy?: {
    readonly setting: string
    readonly options: Readonly<Record<string, readonly ManifestOption[]>>
  }
}

type Manifest = Entry & {
  readonly id: string
  readonly name: string
  readonly version: string
  readonly description?: string
  readonly frontend?: JsonValue
  readonly backend?: JsonValue
  readonly capabilities?: readonly string[]
  readonly settings?: readonly ManifestSetting[]
  readonly settingsViews?: PluginInfo['settingsViews']
  readonly contributes?: Entry & {
    readonly category?: string
    readonly feature?: string
    readonly commands?: readonly Entry[]
    readonly keybindings?: readonly Entry[]
    readonly tabKinds?: readonly Entry[]
  }
}

type Field = JsonValue | undefined

const item = (text: string, detail = '', keys = ''): PluginContributionItem => ({
  text,
  detail,
  keys,
})

const getText = (value: Field): string => (typeof value === 'string' ? value : '')

const titled = (title: Field, id: Field): PluginContributionItem =>
  item(getText(title) || getText(id), getText(id))

const listCommands = (contributes: NonNullable<Manifest['contributes']>) => {
  const keys = new Map<string, string>()
  for (const one of contributes.keybindings ?? []) {
    const command = getText(one.command)
    if (!keys.has(command)) keys.set(command, getText(one.key))
  }
  return (contributes.commands ?? []).map((one) =>
    item(getText(one.title) || getText(one.id), getText(one.id), keys.get(getText(one.id)) ?? ''),
  )
}

const listForeignKeybindings = (contributes: NonNullable<Manifest['contributes']>) => {
  const declared = new Set((contributes.commands ?? []).map((one) => getText(one.id)))
  return (contributes.keybindings ?? [])
    .filter((one) => !declared.has(getText(one.command)))
    .map((one) => item(getText(one.command), '', getText(one.key)))
}

const getSlotTarget = (one: Entry): string =>
  getText(one.command) || getText(one.view) || getText(one.submenu)

const getListenerPort = (one: Entry): string =>
  getText(one.portSetting) || (one.port ? String(one.port) : '')

const getList = (value: Field): readonly string[] =>
  Array.isArray(value) ? value.map(String) : []

const asEntry = (value: Field): Entry =>
  typeof value === 'object' && value !== null && !Array.isArray(value) ? (value as Entry) : {}

type ToItem = (one: JsonValue) => PluginContributionItem

/** An item of an entry that is an object, read through `read`. */
const fromEntry =
  (read: (one: Entry) => PluginContributionItem): ToItem =>
  (one) =>
    read(asEntry(one))

/** An item of an entry that is a text. */
const fromText: ToItem = (one) => item(getText(one))

/** The kinds of contributes, in the order a page lists them, each with how an entry becomes an item. */
const CONTRIBUTION_KINDS: readonly (readonly [PluginContributionKind, ToItem])[] = [
  ['tabKinds', fromEntry((one) => titled(one.title, one.kind))],
  ['readers', fromEntry((one) => item(getText(one.id), getList(one.contentTypes).join(', ')))],
  ['handlers', fromEntry((one) => titled(one.title, one.command))],
  ['triggers', fromEntry((one) => item(getText(one.event), getText(one.command)))],
  ['textKinds', fromEntry((one) => titled(one.title, one.kind))],
  ['noteTypes', fromEntry((one) => item(getText(one.type)))],
  ['themes', fromEntry((one) => titled(one.name, one.id))],
  ['slots', fromEntry((one) => item(getText(one.slot), getSlotTarget(one)))],
  ['listeners', fromEntry((one) => item(getText(one.name), getListenerPort(one)))],
  ['viewCapabilities', fromText],
  ['dropHandlers', fromEntry((one) => item(getList(one.types).join(', ')))],
  ['frameOrigins', fromText],
  ['connectionTypes', fromEntry((one) => titled(one.title, one.id))],
  ['operations', fromEntry((one) => titled(one.title, one.name))],
]

/** The lists at the top of a manifest that a page shows with the contributions. */
const MANIFEST_KINDS: readonly (readonly [PluginContributionKind, ToItem])[] = [
  ['permissions', fromText],
  ['dependencies', fromEntry((one) => item(getText(one.id), getText(one.kind)))],
  ['services', fromText],
  ['capabilities', fromText],
  ['activationEvents', fromText],
]

/** The list a source holds under a key, each entry made an item; a list it does not hold is empty. */
const listItems = (source: Entry, key: string, toItem: ToItem) => {
  const list = source[key]
  return Array.isArray(list) ? list.map((one: JsonValue) => toItem(one)) : []
}

const listContributions = (manifest: Manifest): readonly PluginContribution[] => {
  const contributes = manifest.contributes ?? {}
  const sections: (readonly [PluginContributionKind, readonly PluginContributionItem[]])[] = [
    ['commands', listCommands(contributes)],
    ['keybindings', listForeignKeybindings(contributes)],
    ...CONTRIBUTION_KINDS.map(
      ([kind, toItem]) => [kind, listItems(contributes, kind, toItem)] as const,
    ),
    ...MANIFEST_KINDS.map(([kind, toItem]) => [kind, listItems(manifest, kind, toItem)] as const),
  ]
  return sections.filter(([, items]) => items.length > 0).map(([kind, items]) => ({ kind, items }))
}

const readOption = (option: ManifestOption): PluginSettingOption =>
  typeof option === 'object'
    ? {
        value: String(option.value),
        title: option.title ?? String(option.value),
        detail: option.detail ?? '',
        ...(option.when ? { when: option.when } : {}),
      }
    : { value: String(option), title: String(option), detail: '' }

const COPIED_FIELDS = [
  'palette',
  'icon',
  'when',
  'accept',
  'multiple',
  'min',
  'max',
  'step',
  'preview',
  'optionsFrom',
  'operations',
]

const readSetting = (setting: ManifestSetting): PluginSettingInfo => ({
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
    COPIED_FIELDS.filter((field) => setting[field] !== undefined).map((field) => [
      field,
      setting[field],
    ]),
  ),
})

const readPlugin = (manifest: Manifest): PluginInfo => ({
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

const readCatalogEntry = (manifest: Manifest): CatalogEntry => ({
  id: manifest.id,
  name: manifest.name,
  version: manifest.version,
  description: manifest.description ?? '',
  category: manifest.contributes?.category ?? '',
  feature: manifest.contributes?.feature ?? '',
  tabKinds: (manifest.contributes?.tabKinds ?? []).map((one) => getText(one.kind)),
  isAvailable: false,
})

const MANIFESTS = import.meta.glob<Manifest>('../../../../../../../plugins/*/manifest.json', {
  eager: true,
  import: 'default',
})
const READMES = import.meta.glob<string>('../../../../../../../plugins/*/README.md', {
  eager: true,
  query: '?raw',
  import: 'default',
})

/** The manifests in the order of their folders, each with the README of its folder. */
const SHIPPED = Object.keys(MANIFESTS)
  .sort()
  .map((file) => ({
    manifest: MANIFESTS[file] as Manifest,
    readme: READMES[file.replace(/manifest\.json$/, 'README.md')],
  }))

export const SHIPPED_PLUGINS: readonly PluginInfo[] = SHIPPED.map((one) => readPlugin(one.manifest))

export const SHIPPED_CATALOG: readonly CatalogEntry[] = SHIPPED.map((one) =>
  readCatalogEntry(one.manifest),
)

export const SHIPPED_READMES: Readonly<Record<string, string>> = Object.fromEntries(
  SHIPPED.flatMap((one) => (one.readme === undefined ? [] : [[one.manifest.id, one.readme]])),
)
