/**
 * The parts of a registry entry read from a plugin's own files, the same for a plugin of this
 * repository and one of another: the manifest's fields, the README and the screenshots.
 */
import { toJson } from '@bufbuild/protobuf'
import { ManifestSchema } from '../../libs/shared/protocol/src/numen/v1/manifest_pb.ts'
import { describePermission } from '../../libs/plugin/manager/src/words.ts'

/** The line that ends the introduction of a README. */
export const README_MORE = '<!-- more -->'

/** The id prefix of the plugins Jiva Studio publishes. */
export const OFFICIAL_PREFIX = 'md.numen.'

const SCREENSHOT = /^(.+)-(light|dark)\.webp$/

/** The lucide icon of a plugin whose manifest declares none, by category. */
export const CATEGORY_ICONS = {
  agents: 'bot',
  connections: 'plug',
  engines: 'cpu',
  readers: 'book-open',
  themes: 'palette',
  tools: 'wrench',
  views: 'layout-panel-left',
}

/** The lucide icon of a plugin of no known category that declares no icon. */
export const DEFAULT_ICON = 'puzzle'

/**
 * The lucide icon a plugin is shown with: the first icon its manifest declares on a tab kind, a
 * command, a slot or a setting, in that order, else the icon of its category.
 *
 * @param {Record<string, any>} json - the output of `toManifestJson`
 * @returns {string}
 */
export const getIcon = (json) => {
  const contributes = json.contributes ?? {}
  const declared = [contributes.tabKinds, contributes.commands, contributes.slots, json.settings]
    .flatMap((list) => list ?? [])
    .find((one) => one.icon)?.icon
  return declared ?? CATEGORY_ICONS[contributes.category] ?? DEFAULT_ICON
}

/**
 * A README split for the catalog: `intro` is the text before `<!-- more -->` and `text` is the
 * whole README, both without the first-level heading the README opens with and without the
 * `<picture>` blocks of its screenshots, which the page shows from the screenshots folder.
 *
 * @param {string | null} source - the README, or null when the plugin has none
 * @returns {{ intro: string, text: string, hasIntro: boolean } | null}
 */
export const splitReadme = (source) => {
  if (source === null) return null
  const text = source
    .replace(/^\s*# [^\n]*\n/, '')
    .replace(/<picture>[\s\S]*?<\/picture>\s*/g, '')
    .trim()
  const at = text.indexOf(README_MORE)
  return {
    intro: (at === -1 ? '' : text.slice(0, at)).trim(),
    text: text.replace(README_MORE, '').trim(),
    hasIntro: at !== -1,
  }
}

/**
 * The screenshots among `names`, paired by name: `overview-light.webp` and `overview-dark.webp`
 * are one screenshot. A name with one of the two is kept with the other left null.
 *
 * @param {string[]} names - file names in the screenshots folder
 * @param {(name: string) => string} getPath - the published path of a file name
 * @returns {{ name: string, light: string | null, dark: string | null }[]} ordered by name
 */
export const pairScreenshots = (names, getPath) => {
  const byName = new Map()
  for (const file of names) {
    const match = SCREENSHOT.exec(file)
    if (!match) continue
    const [, name, appearance] = match
    const pair = byName.get(name) ?? { name, light: null, dark: null }
    pair[appearance] = getPath(file)
    byName.set(name, pair)
  }
  return [...byName.values()].sort((a, b) => a.name.localeCompare(b.name))
}

const getSettings = (settings = []) =>
  settings.map((one) => ({
    key: one.key,
    title: one.title ?? one.key,
    description: one.description ?? '',
  }))

const getCommands = (commands = []) =>
  commands.map((one) => ({ id: one.id, title: one.title ?? one.id }))

const getTabKinds = (tabKinds = []) =>
  tabKinds.map((one) => ({ kind: one.kind, title: one.title ?? one.kind }))

const getFileTypes = (readers = [], tabKinds = []) =>
  readers.flatMap((reader) => {
    const tab = tabKinds.find((one) => one.kind === `reader:${reader.id}`)
    return (reader.contentTypes ?? []).map((contentType) => ({
      contentType,
      title: tab?.title ?? reader.id,
    }))
  })

const getTextKinds = (textKinds = []) =>
  textKinds.map((one) => ({ kind: one.kind, title: one.title ?? one.kind }))

const getThemes = (themes = []) => themes.map((one) => ({ id: one.id, name: one.name ?? one.id }))

const getSlots = (slots = []) =>
  slots.map((one) => ({
    slot: one.slot,
    title: one.title ?? '',
    target: one.command ?? one.view ?? one.submenu ?? '',
  }))

const getConnectionTypes = (connectionTypes = []) =>
  connectionTypes.map((one) => ({
    id: one.id,
    title: one.title ?? one.id,
    description: one.description ?? '',
  }))

const getSearchFunctions = (searchFunctions) =>
  (searchFunctions?.functions ?? []).map((one) => ({
    name: one.name,
    description: one.description ?? '',
  }))

const getLicenses = (licenses = []) =>
  licenses.map((one) => ({ id: one.id, title: one.title ?? one.id, description: one.description ?? '' }))

/** A manifest message in the schema's JSON form. */
export const toManifestJson = (manifest) => toJson(ManifestSchema, manifest)

/**
 * The catalog fields of a manifest in the schema's JSON form.
 *
 * @param {Record<string, any>} json - the output of `toManifestJson`
 * @returns {object} the fields of the entry the manifest gives
 */
export const describeManifest = (json) => {
  const contributes = json.contributes ?? {}
  const licenses = getLicenses(contributes.licenses)
  return {
    id: json.id,
    name: json.name,
    version: json.version,
    description: json.description ?? '',
    category: contributes.category ?? '',
    feature: contributes.feature ?? '',
    icon: getIcon(json),
    targets: json.targets ?? [],
    isOfficial: json.id.startsWith(OFFICIAL_PREFIX),
    isPaid: licenses.length > 0,
    licenses,
    permissions: (json.permissions ?? []).map((id) => ({ id, description: describePermission(id) })),
    settings: getSettings(json.settings),
    commands: getCommands(contributes.commands),
    tabKinds: getTabKinds(contributes.tabKinds),
    fileTypes: getFileTypes(contributes.readers, contributes.tabKinds),
    textKinds: getTextKinds(contributes.textKinds),
    themes: getThemes(contributes.themes),
    slots: getSlots(contributes.slots),
    connectionTypes: getConnectionTypes(contributes.connectionTypes),
    searchFunctions: getSearchFunctions(contributes.searchFunctions),
  }
}

/**
 * The permissions of `entry` that its previous version did not declare. A version seen before keeps
 * the list it had, so the list holds until the next version.
 *
 * @param {{ version: string, permissions: { id: string }[] }} entry - the entry read now
 * @param {{ version: string, permissions: { id: string }[], permissionsAdded?: string[] } | undefined} previous
 * @returns {string[]}
 */
export const listAddedPermissions = (entry, previous) => {
  if (!previous) return []
  if (previous.version === entry.version) return previous.permissionsAdded ?? []
  const before = new Set(previous.permissions.map((one) => one.id))
  return entry.permissions.map((one) => one.id).filter((id) => !before.has(id))
}
