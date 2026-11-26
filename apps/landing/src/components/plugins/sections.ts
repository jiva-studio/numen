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
type PluginRow = {
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

const createPlainText = (text: string): RowText => ({ text, key: null, isCode: false })

const createCodeText = (text: string): RowText => ({ text, key: null, isCode: true })

const createTranslatedText = (key: string, fallback: string): RowText => ({
  text: EN[key] ?? fallback,
  key,
  isCode: false,
})

/** The place in the window a slot is, in words; a slot the site has no words for is its id. */
export function getSlotPlace(slot: string): RowText {
  const key = `plugin_slot_${slot.replace(/\./g, '_')}`
  return EN[key] === undefined ? createCodeText(slot) : createTranslatedText(key, slot)
}

const createRow = (name: RowText[], value: RowText[][] = []): PluginRow => ({
  name,
  value: value
    .map((line) => line.filter((one) => one.text !== ''))
    .filter((line) => line.length > 0),
  status: null,
  isNew: false,
  href: null,
})

const createSection = (id: string, rows: PluginRow[], note: RowText | null = null): PluginSection => {
  const key = `plugin_${id}`
  return { id, key, title: EN[key] ?? id, rows, note }
}

/** The sections that have rows. */
function listShownSections(sections: readonly PluginSection[]): PluginSection[] {
  return sections.filter((one) => one.rows.length > 0)
}

/** What the plugin adds to Numen and what it may reach, with its downloads. */
export function listContributionSections(plugin: CatalogPlugin): PluginSection[] {
  const added = new Set(plugin.permissionsAdded)
  return listShownSections([
    createSection(
      'permissions',
      plugin.permissions.map((one) => ({
        ...createRow([createPlainText(one.description)], [[createCodeText(one.id)]]),
        isNew: added.has(one.id),
      })),
    ),
    createSection(
      'settings',
      plugin.settings.map((one) => createRow([createPlainText(one.title)], [[createPlainText(one.description)]])),
    ),
    createSection(
      'commands',
      plugin.commands.map((one) => createRow([createPlainText(one.title)], [[createCodeText(one.id)]])),
    ),
    createSection(
      'tabs',
      plugin.tabKinds.map((one) => createRow([createPlainText(one.title)], [[createCodeText(one.kind)]])),
    ),
    createSection(
      'file_types',
      plugin.fileTypes.map((one) => createRow([createPlainText(one.contentType)], [[createPlainText(one.title)]])),
    ),
    createSection(
      'text_kinds',
      plugin.textKinds.map((one) => createRow([createPlainText(one.title)], [[createCodeText(one.kind)]])),
    ),
    createSection(
      'themes',
      plugin.themes.map((one) => createRow([createPlainText(one.name)], [[createCodeText(one.id)]])),
    ),
    createSection(
      'slots',
      plugin.slots.map((one) =>
        createRow([one.title === '' ? createCodeText(one.target) : createPlainText(one.title)], [[getSlotPlace(one.slot)]]),
      ),
    ),
    createSection(
      'connections',
      plugin.connectionTypes.map((one) => createRow([createPlainText(one.title)], [[createPlainText(one.description)]])),
    ),
    createSection(
      'search_functions',
      plugin.searchFunctions.map((one) => createRow([createCodeText(one.name)], [[createPlainText(one.description)]])),
    ),
    createSection(
      'downloads',
      plugin.downloads.map((one) =>
        createRow(
          [createPlainText(`${one.id} ${one.version}`)],
          one.platforms.map((file) => [
            createPlainText(getPlatformName(file.platform)),
            file.size === null
              ? createTranslatedText('plugin_size_unknown', 'size unknown')
              : createPlainText(formatSize(file.size)),
          ]),
        ),
      ),
    ),
    createSection(
      'artifacts',
      plugin.artifacts.map((one) => ({
        ...createRow([createPlainText(one.name)], [[createPlainText(formatSize(one.size)), createCodeText(one.sha256 ?? '')]]),
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
    createRow([createTranslatedText(one.key, one.name)], [[createPlainText(String(quality.tests[one.kind]))]]),
  )
  const measured = quality.entries.flatMap((entry) => {
    const kind = getQualityKind(entry)
    const suffix = kind === '' ? [] : [createPlainText(kind)]
    return [
      { key: 'plugin_mutation', name: 'Mutation score', value: entry.mutationScore },
      { key: 'plugin_coverage', name: 'Coverage', value: entry.coverage },
    ]
      .filter((one) => one.value !== null)
      .map((one) =>
        createRow([createTranslatedText(one.key, one.name), ...suffix], [[createPlainText(formatPercent(one.value))]]),
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
    createRow(
      [createCodeText(one.name)],
      [
        [
          createPlainText(ECOSYSTEM_NAMES[one.ecosystem] ?? one.ecosystem),
          createCodeText(one.version),
          one.license === null
            ? createTranslatedText('plugin_license_unknown', 'license unknown')
            : createPlainText(one.license),
        ],
      ],
    ),
  )
  const indirect = INDIRECT_COUNTS.filter((one) => dependencies.indirect[one.ecosystem] > 0).map(
    (one) =>
      createRow([createTranslatedText(one.key, one.name)], [[createPlainText(String(dependencies.indirect[one.ecosystem]))]]),
  )
  return [...packages, ...indirect]
}

/** The checks the registry ran, the known vulnerabilities, the dependencies and the tests. */
export function listQualitySections(plugin: CatalogPlugin): PluginSection[] {
  const isReportedByAuthor = plugin.quality?.reportedBy === 'author'
  return listShownSections([
    createSection(
      'checks',
      plugin.checks.map((one) => ({
        ...createRow(
          [createTranslatedText(`plugin_check_${one.id}`, one.id)],
          one.detail.split('\n').map((line) => [createPlainText(line)]),
        ),
        status: one.status,
      })),
    ),
    createSection(
      'vulnerabilities',
      (plugin.vulnerabilities ?? []).map((one) => ({
        ...createRow([createPlainText(one.id)], [[createCodeText(`${one.name} ${one.version}`)], [createPlainText(one.summary)]]),
        href: one.url,
      })),
    ),
    createSection('dependencies', listDependencyRows(plugin)),
    createSection(
      'tests',
      listTestRows(plugin),
      isReportedByAuthor ? createTranslatedText('plugin_quality_author', 'Reported by the author.') : null,
    ),
  ])
}
