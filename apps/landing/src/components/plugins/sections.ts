/**
 * The sections of the Contributions and Quality tabs of a plugin page, each a list of rows of a
 * name and a value. A section without rows is left out.
 */
import { EN } from '../../scripts/i18n-en.ts'
import {
  type CatalogPlugin,
  formatPercent,
  formatSize,
  getPlatformName,
  getQualityKind,
} from './catalog.ts'
import type { PluginCheckStatus } from './registry.ts'

/** A piece of text in a row; `key` names its translation. */
export type RowText = { text: string; key: string | null; isCode: boolean }

/** One row: its name, its value as lines of text, and the status of a check. */
export type PluginRow = {
  name: RowText[]
  value: RowText[][]
  status: PluginCheckStatus | null
  isNew: boolean
  href: string | null
}

/** A titled list of rows, with an optional note under it. */
export type PluginSection = {
  id: string
  key: string
  title: string
  rows: PluginRow[]
  note: RowText | null
}

const plain = (text: string): RowText => ({ text, key: null, isCode: false })

const code = (text: string): RowText => ({ text, key: null, isCode: true })

const translated = (key: string, fallback: string): RowText => ({
  text: EN[key] ?? fallback,
  key,
  isCode: false,
})

/** The place in the window a slot is, in words; a slot the site has no words for is its id. */
export function getSlotPlace(slot: string): RowText {
  const key = `plugin_slot_${slot.replace(/\./g, '_')}`
  return EN[key] === undefined ? code(slot) : translated(key, slot)
}

const row = (name: RowText[], value: RowText[][] = []): PluginRow => ({
  name,
  value: value
    .map((line) => line.filter((one) => one.text !== ''))
    .filter((line) => line.length > 0),
  status: null,
  isNew: false,
  href: null,
})

const section = (id: string, rows: PluginRow[], note: RowText | null = null): PluginSection => {
  const key = `plugin_${id}`
  return { id, key, title: EN[key] ?? id, rows, note }
}

/** The sections that have rows. */
export function listShownSections(sections: readonly PluginSection[]): PluginSection[] {
  return sections.filter((one) => one.rows.length > 0)
}

/** What the plugin adds to Numen and what it may reach, with its downloads. */
export function listContributionSections(plugin: CatalogPlugin): PluginSection[] {
  const added = new Set(plugin.permissionsAdded)
  return listShownSections([
    section(
      'permissions',
      plugin.permissions.map((one) => ({
        ...row([plain(one.description)], [[code(one.id)]]),
        isNew: added.has(one.id),
      })),
    ),
    section(
      'settings',
      plugin.settings.map((one) => row([plain(one.title)], [[plain(one.description)]])),
    ),
    section(
      'commands',
      plugin.commands.map((one) => row([plain(one.title)], [[code(one.id)]])),
    ),
    section(
      'tabs',
      plugin.tabKinds.map((one) => row([plain(one.title)], [[code(one.kind)]])),
    ),
    section(
      'file_types',
      plugin.fileTypes.map((one) => row([plain(one.contentType)], [[plain(one.title)]])),
    ),
    section(
      'text_kinds',
      plugin.textKinds.map((one) => row([plain(one.title)], [[code(one.kind)]])),
    ),
    section(
      'themes',
      plugin.themes.map((one) => row([plain(one.name)], [[code(one.id)]])),
    ),
    section(
      'slots',
      plugin.slots.map((one) =>
        row([one.title === '' ? code(one.target) : plain(one.title)], [[getSlotPlace(one.slot)]]),
      ),
    ),
    section(
      'connections',
      plugin.connectionTypes.map((one) => row([plain(one.title)], [[plain(one.description)]])),
    ),
    section(
      'search_functions',
      plugin.searchFunctions.map((one) => row([code(one.name)], [[plain(one.description)]])),
    ),
    section(
      'downloads',
      plugin.downloads.map((one) =>
        row(
          [plain(`${one.id} ${one.version}`)],
          one.platforms.map((file) => [
            plain(getPlatformName(file.platform)),
            file.size === null
              ? translated('plugin_size_unknown', 'size unknown')
              : plain(formatSize(file.size)),
          ]),
        ),
      ),
    ),
    section(
      'artifacts',
      plugin.artifacts.map((one) => ({
        ...row([plain(one.name)], [[plain(formatSize(one.size)), code(one.sha256 ?? '')]]),
        href: one.url,
      })),
    ),
  ])
}

const TEST_COUNTS = [
  { kind: 'go', key: 'plugin_go_tests', name: 'Go tests' },
  { kind: 'typescript', key: 'plugin_typescript_tests', name: 'TypeScript tests' },
  { kind: 'stories', key: 'plugin_stories', name: 'Stories' },
] as const

const listTestRows = (plugin: CatalogPlugin): PluginRow[] => {
  const quality = plugin.quality
  if (!quality) return []
  const counts = TEST_COUNTS.filter((one) => quality.tests[one.kind] > 0).map((one) =>
    row([translated(one.key, one.name)], [[plain(String(quality.tests[one.kind]))]]),
  )
  const measured = quality.entries.flatMap((entry) => {
    const kind = getQualityKind(entry)
    const suffix = kind === '' ? [] : [plain(kind)]
    return [
      { key: 'plugin_mutation', name: 'Mutation score', value: entry.mutationScore },
      { key: 'plugin_coverage', name: 'Coverage', value: entry.coverage },
    ]
      .filter((one) => one.value !== null)
      .map((one) =>
        row([translated(one.key, one.name), ...suffix], [[plain(formatPercent(one.value))]]),
      )
  })
  return [...counts, ...measured]
}

const ECOSYSTEM_NAMES: Readonly<Record<string, string>> = { go: 'Go', npm: 'npm' }

const INDIRECT_COUNTS = [
  { ecosystem: 'go', key: 'plugin_indirect_go', name: 'Indirect Go modules' },
  { ecosystem: 'npm', key: 'plugin_indirect_npm', name: 'Indirect npm packages' },
] as const

const listDependencyRows = (plugin: CatalogPlugin): PluginRow[] => {
  const dependencies = plugin.dependencies
  if (!dependencies) return []
  const packages = dependencies.packages.map((one) =>
    row(
      [code(one.name)],
      [
        [
          plain(ECOSYSTEM_NAMES[one.ecosystem] ?? one.ecosystem),
          code(one.version),
          one.license === null
            ? translated('plugin_license_unknown', 'license unknown')
            : plain(one.license),
        ],
      ],
    ),
  )
  const indirect = INDIRECT_COUNTS.filter((one) => dependencies.indirect[one.ecosystem] > 0).map(
    (one) =>
      row([translated(one.key, one.name)], [[plain(String(dependencies.indirect[one.ecosystem]))]]),
  )
  return [...packages, ...indirect]
}

/** The checks the registry ran, the known vulnerabilities, the dependencies and the tests. */
export function listQualitySections(plugin: CatalogPlugin): PluginSection[] {
  const isReportedByAuthor = plugin.quality?.reportedBy === 'author'
  return listShownSections([
    section(
      'checks',
      plugin.checks.map((one) => ({
        ...row(
          [translated(`plugin_check_${one.id}`, one.id)],
          one.detail.split('\n').map((line) => [plain(line)]),
        ),
        status: one.status,
      })),
    ),
    section(
      'vulnerabilities',
      (plugin.vulnerabilities ?? []).map((one) => ({
        ...row([plain(one.id)], [[code(`${one.name} ${one.version}`)], [plain(one.summary)]]),
        href: one.url,
      })),
    ),
    section('dependencies', listDependencyRows(plugin)),
    section(
      'tests',
      listTestRows(plugin),
      isReportedByAuthor ? translated('plugin_quality_author', 'Reported by the author.') : null,
    ),
  ])
}
