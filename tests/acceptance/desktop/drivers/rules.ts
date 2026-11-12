/** The rules of the driver that need no browser: they say what a scenario may name and fail on the rest. */
import path from 'node:path'

export type TabKind = 'PDF Reader' | 'EPUB Reader' | 'Audio Player'

export interface Reference {
  reference: 'at' | 'titled' | 'for the web address'
  name: string
}

const POSITIONS: Record<TabKind, string> = {
  'PDF Reader': 'page',
  'EPUB Reader': 'page',
  'Audio Player': 'time',
}

export const getStem = (file: string): string => path.basename(file).replace(/\.[^.]+$/, '')

export const escapeRegExp = (text: string): string => text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

export const getTitlePattern = (title: string): RegExp => new RegExp(`^(Linked tab )?${escapeRegExp(title)}( Close |$)`)

/** The title the product gives the tab of a web address: the characters a file name cannot have become "-". */
export const getAddressTitle = (address: string): string => address.replace(/[/\\:*?"<>|#]/g, '-')

export const getLinkNotePath = (address: string): string => `${getAddressTitle(address)}.md`

export const parseReference = (reference: string, name: string): Reference => {
  if (reference !== 'at' && reference !== 'titled' && reference !== 'for the web address') {
    throw new Error(`"${reference}" does not name a tab or a note: use "at", "titled" or "for the web address"`)
  }
  return { reference, name }
}

export const assertPosition = (kind: TabKind, position: string, file: string): void => {
  if (POSITIONS[kind] !== position) {
    throw new Error(`the tab "${file}" is a ${kind} tab: its position is a ${POSITIONS[kind]}, not a ${position}`)
  }
}

export const parseTime = (time: string): number => {
  const match = /^(\d+):([0-5]\d)$/.exec(time)
  if (!match) throw new Error(`"${time}" is not a time: write minutes and seconds, like "2:00"`)
  return Number(match[1]) * 60 + Number(match[2])
}

export const assertEntry = (entry: string, path: string, isFolder: boolean): void => {
  if (entry !== 'file' && entry !== 'folder') throw new Error(`"${entry}" is not a file or a folder`)
  if ((entry === 'folder') !== isFolder) {
    throw new Error(`"${path}" is a ${isFolder ? 'folder' : 'file'}, not a ${entry}`)
  }
}

export const getAncestors = (entryPath: string): string[] => {
  const parts = entryPath.split('/').slice(0, -1)
  return parts.map((_, index) => parts.slice(0, index + 1).join('/'))
}

export type PaletteField = 'name' | 'item' | 'note' | 'vault' | 'text'

const PALETTE_FIELDS: Record<string, PaletteField> = { Name: 'name', Search: 'item', 'Search notes': 'note', 'Search vaults': 'vault' }

/** What the field of the open palette asks for, read from its placeholder: a name, one item of a list, a note found by words, or a text. */
export const getPaletteField = (placeholder: string): PaletteField => PALETTE_FIELDS[placeholder] ?? 'text'

export type PaletteAction = 'name' | 'pick' | 'type'

const ACTION_FIELDS: Record<PaletteAction, readonly PaletteField[]> = {
  name: ['name'],
  pick: ['item', 'note', 'vault'],
  type: ['text'],
}

const ACTION_WORDS: Record<PaletteAction, string> = {
  name: 'names it',
  pick: 'picks',
  type: 'types',
}

const FIELD_ASKS: Record<PaletteField, string> = {
  name: 'a name',
  item: 'an item to pick',
  note: 'an item to pick',
  vault: 'an item to pick',
  text: 'a text',
}

/** Fails when the palette asks for another kind of input than the step gives, so a step cannot do the work of another. */
export const assertPaletteField = (action: PaletteAction, kind: PaletteField): void => {
  if (!ACTION_FIELDS[action].includes(kind)) {
    throw new Error(`the person ${ACTION_WORDS[action]} here, but the palette asks for ${FIELD_ASKS[kind]}: use the step that matches`)
  }
}

/** The kinds of text a file or a link has, as the product titles its versions. */
export type TextKind = 'recognized text' | 'transcript' | 'proofread text' | 'article'

const TEXT_KIND_TITLES: Record<TextKind, string> = {
  'recognized text': 'Recognized text',
  transcript: 'Transcript',
  'proofread text': 'Proofread text',
  article: 'Article',
}

/** The title the product gives a kind of text in the list of the versions of a file. */
export const getVersionTitle = (kind: TextKind): string => TEXT_KIND_TITLES[kind]

/** The title the product gives the live version of a kind of text in the list it offers. */
export const getCurrentVersionLabel = (kind: TextKind): string => `${TEXT_KIND_TITLES[kind]} (current)`

/** The application a world runs: the server of a world runs one, and the flashcards window opens one vault. */
export const assertWorldApplication = (application: string, vaults: readonly string[], running: string | undefined): void => {
  if (running !== undefined && running !== application) {
    throw new Error(`the world runs the application "${running}" already: a world has one application, not "${application}" too`)
  }
  if (application === 'flashcards' && vaults.length > 1) {
    throw new Error(`the application "flashcards" opens one vault, and the world has ${vaults.join(' and ')}: a world for the flashcards window has one vault`)
  }
}

export type Application = 'editor' | 'flashcards'

export const parseApplication = (name: string): Application => {
  if (name !== 'editor' && name !== 'flashcards') {
    throw new Error(`"${name}" is not an application: use "editor" or "flashcards"`)
  }
  return name
}

const RATINGS = new Map([
  ['Again', 1],
  ['Hard', 2],
  ['Good', 3],
  ['Easy', 4],
])

/** The number the review log writes for an answer. */
export const getRatingNumber = (answer: string): number => {
  const rating = RATINGS.get(answer)
  if (rating === undefined) throw new Error(`"${answer}" is not an answer: use Again, Hard, Good or Easy`)
  return rating
}

export const parseReviewScope = (scope: string): 'vault' | 'deck' => {
  if (scope !== 'vault' && scope !== 'deck') throw new Error(`"${scope}" is not reviewed: use "vault" or "deck"`)
  return scope
}

export const readNoteType = (content: string): string => {
  const frontmatter = /^---\r?\n([\s\S]*?)\r?\n---/.exec(content)
  const match = /^type:\s*(\S+)\s*$/m.exec(frontmatter?.[1] ?? '')
  return (match?.[1] ?? '').replace(/^["']|["']$/g, '')
}

export const pickByTitle = (title: string, paths: readonly string[]): string => {
  const found = paths.filter((one) => getStem(one) === title)
  if (found.length === 0) throw new Error(`no tab is titled "${title}": use "at" with its path`)
  if (found.length > 1) throw new Error(`more than one tab is titled "${title}": use "at" with its path`)
  return found[0]
}

export const assertSingleTab = (title: string, count: number): void => {
  if (count > 1) throw new Error(`${count} tabs are titled "${title}": use "at" with the path`)
}

/** The name a step gives for the only vault of a world, when it does not name one. */
export const DEFAULT_VAULT = 'default'

/** The vault a step names: the words `the vault`, or `the vault "Archive"`. */
export const parseVaultName = (text: string): string => {
  const match = /^the vault(?: "([^"]+)")?$/.exec(text)
  if (!match) throw new Error(`"${text}" does not name a vault: write the vault or the vault "Name"`)
  const name = match[1] as string | undefined
  return name ?? DEFAULT_VAULT
}

/** The vault of a world a step means: the one it names, or the only one when it names none. */
export const pickVault = (vaults: readonly string[], name: string): string => {
  if (name !== DEFAULT_VAULT) {
    if (vaults.includes(name)) return name
    throw new Error(`the world has no vault "${name}": it has ${vaults.join(', ') || 'none'}`)
  }
  if (vaults.length === 1) return vaults[0]
  if (vaults.length === 0) throw new Error('the world has no vault: begin with the empty vault "Name"')
  throw new Error(`name the vault: the world has ${vaults.join(' and ')}`)
}

/** The name of a vault folder in the world: a plain name, not the word that stands for the only vault. */
export const assertVaultName = (name: string): void => {
  if (name === '' || name === DEFAULT_VAULT || name.startsWith('.') || /[/\\:*?"<>|]/.test(name)) {
    throw new Error(`"${name}" cannot name a vault: use a plain name other than "${DEFAULT_VAULT}", with no path in it`)
  }
}

/** The file the product's create-note command makes for a title: in the root of the vault, named by the title. */
export const getNoteFile = (title: string): string => {
  if (title.trim() === '' || /[/\\:*?"<>|#]/.test(title)) {
    throw new Error(`"${title}" cannot be the title of a note file: leave out / \\ : * ? " < > | #`)
  }
  return `${title.trim()}.md`
}

/** A path inside a vault: relative, and never leaving the vault. */
export const assertVaultPath = (entryPath: string): void => {
  const parts = entryPath.split('/')
  if (entryPath.startsWith('/') || parts.some((one) => one === '' || one === '.' || one === '..')) {
    throw new Error(`"${entryPath}" is not a path inside the vault: write it relative to the vault, like "assets/a.pdf"`)
  }
}

/** The kind a step means by "any" tab: it names none, so a tab of any kind is meant. */
export const ANY_TAB = 'any'

/** The words a scenario puts before "tab", with the id and the title the product gives the kind (the `tabKinds` of the plugin manifests). */
export const TAB_KINDS = {
  note: { id: 'note', title: 'Note' },
  Plex: { id: 'plex', title: 'Plex' },
  PDF: { id: 'reader:reader-pdf', title: 'PDF Reader' },
  book: { id: 'reader:reader-epub', title: 'EPUB Reader' },
  recording: { id: 'reader:media-viewer', title: 'Audio Player' },
  Media: { id: 'media', title: 'Media' },
  Files: { id: 'files', title: 'Files' },
  Agent: { id: 'agent', title: 'Agent' },
  Settings: { id: 'settings', title: 'Settings' },
  Deck: { id: 'deck', title: 'Deck' },
  'Card Model': { id: 'card-model', title: 'Card Model' },
  Preset: { id: 'preset', title: 'Preset' },
} as const

export type TabKindWord = keyof typeof TAB_KINDS

export type TabKindChoice = TabKindWord | typeof ANY_TAB

export interface KindedTab {
  readonly id: string
  readonly kind: string
}

const isTabKindWord = (word: string): word is TabKindWord => Object.hasOwn(TAB_KINDS, word)

/** The kind a step names in "the note tab" or "the tab": a word of TAB_KINDS, or "any" for none. */
export const parseTabKind = (text: string): TabKindChoice => {
  const match = /^(?:([A-Za-z]+(?: [A-Za-z]+)?) )?tab$/.exec(text)
  if (!match) throw new Error(`"${text}" does not name a tab: write the tab or the note tab`)
  const word = match[1] as string | undefined
  if (word === undefined) return ANY_TAB
  if (!isTabKindWord(word)) {
    throw new Error(`the tab kind "${word}" is not a tab kind: use ${Object.keys(TAB_KINDS).join(', ')}`)
  }
  return word
}

const getKindTitle = (kind: string): string =>
  Object.values(TAB_KINDS).find((one) => one.id === kind)?.title ?? kind

/** The tab of the kind among the tabs the reference names; none of the kind fails with the kind the tabs have. */
export const pickKindTab = <T extends KindedTab>(
  kind: TabKindWord,
  candidates: readonly T[],
  reference: string,
  name: string,
): T | undefined => {
  const found = candidates.filter((one) => one.kind === TAB_KINDS[kind].id)
  if (found.length > 1) throw new Error(`${found.length} ${TAB_KINDS[kind].title} tabs match "${name}": use "at" with the path`)
  if (found.length === 1 || candidates.length === 0) return found[0]
  const kinds = [...new Set(candidates.map((one) => `${getKindTitle(one.kind)} tab`))].join(' and a ')
  throw new Error(`the tab ${reference} "${name}" is a ${kinds}, not a ${TAB_KINDS[kind].title} tab`)
}

export interface Place {
  position: string
  where: string
}

/** The place a step names in `page "2"` or `time "2:00"`: the kind of position and the value of it. */
export const parsePlace = (text: string): Place => {
  const match = /^(page|time) "([^"]+)"$/.exec(text)
  if (!match) throw new Error(`"${text}" is not a place: write page "2" or time "2:00"`)
  return { position: match[1], where: match[2] }
}

/** What a command runs over: nothing, a file or a folder of the vault, or a tab. */
export type CommandTarget =
  | { readonly kind: 'none' }
  | { readonly kind: 'entry'; readonly entry: string; readonly path: string }
  | { readonly kind: 'tab'; readonly tab: TabKindChoice; readonly reference: string; readonly name: string }

/** The words a step puts after the command: ` on the file at "a.md"`, ` on the Plex tab titled "A"`, or nothing. */
export const parseTarget = (text: string): CommandTarget => {
  if (text === '') return { kind: 'none' }
  const entry = /^ on the (file|folder) at "([^"]+)"$/.exec(text)
  if (entry) return { kind: 'entry', entry: entry[1], path: entry[2] }
  const tab = /^ on the ((?:[A-Za-z]+ ){0,2}tab) (at|titled) "([^"]+)"$/.exec(text)
  if (!tab) throw new Error(`"${text}" does not name what the command runs on: use the file at, the folder at or a tab`)
  return { kind: 'tab', tab: parseTabKind(tab[1]), reference: tab[2], name: tab[3] }
}

/** The pairs of a table of arguments that has a name and a value in each row and no header. */
export const parseArguments = (rows: readonly (readonly string[])[]): [string, string][] =>
  rows.map((row) => {
    if (row.length !== 2) throw new Error(`the row "${row.join(' | ')}" must have a name and a value: "| Name | Brine log |"`)
    return [row[0], row[1]]
  })

/** Whether the palette asks for what the table row names: the placeholder of its field, or the title of a group it lists. */
export const isAskedFor = (key: string, placeholder: string, groups: readonly string[]): boolean =>
  key === placeholder || groups.includes(key)

/** The Playwright name of a key combination written as "Ctrl+Shift+E". */
export const parseKeys = (text: string): string => {
  const names = new Map([
    ['Ctrl', 'Control'],
    ['Shift', 'Shift'],
    ['Alt', 'Alt'],
  ])
  const parts = text.split('+')
  const key = parts.pop() ?? ''
  const modifiers = parts.map((one) => names.get(one))
  if (key === '' || modifiers.some((one) => one === undefined)) {
    throw new Error(`"${text}" is not a key combination: write Ctrl, Shift or Alt, a plus and a key, like "Ctrl+Shift+E"`)
  }
  return [...modifiers, key].join('+')
}

/** The rows of a table with one column, below its header. */
export const parseColumn = (rows: readonly (readonly string[])[], header: string): string[] => {
  if (rows.length === 0 || rows[0]?.length !== 1 || rows[0][0] !== header) {
    throw new Error(`the table must start with the header "| ${header} |"`)
  }
  return rows.slice(1).map((row) => row[0])
}

export interface PaletteRow {
  readonly list: string
  readonly row: string
  readonly isDisabled: boolean
}

/** The rows a table with the header `list | row | disabled` names; disabled is True or False. */
export const parsePaletteRows = (rows: readonly (readonly string[])[]): PaletteRow[] => {
  if (rows.length === 0 || rows[0]?.join('|') !== 'list|row|disabled') {
    throw new Error('the table must start with the header "| list | row | disabled |"')
  }
  return rows.slice(1).map(([list, row, disabled]) => {
    if (disabled !== 'True' && disabled !== 'False') throw new Error(`disabled is True or False, not "${disabled}"`)
    return { list, row, isDisabled: disabled === 'True' }
  })
}

/** The tabs a table with the header `kind | path` lists. */
export const parseTabRows = (rows: readonly (readonly string[])[]): { kind: TabKindWord; path: string }[] => {
  if (rows.length === 0 || rows[0]?.join('|') !== 'kind|path') throw new Error('the table must start with the header "| kind | path |"')
  return rows.slice(1).map((row) => {
    const kind = row[0] ?? ''
    if (!isTabKindWord(kind)) throw new Error(`the tab kind "${kind}" is not a tab kind: use ${Object.keys(TAB_KINDS).join(', ')}`)
    return { kind, path: row[1] ?? '' }
  })
}

/** The address a link note holds, read from its frontmatter. */
export const readLinkAddress = (content: string): string | undefined => {
  if (readNoteType(content) !== 'link') return undefined
  const frontmatter = /^---\r?\n([\s\S]*?)\r?\n---/.exec(content)
  const match = /^url:\s*(.+?)\s*$/m.exec(frontmatter?.[1] ?? '')
  return match?.[1]?.replace(/^["']|["']$/g, '')
}

/** A tab a step names: the kind of it, how it is named, and the name. */
export interface TabRef {
  readonly kind: TabKindChoice
  readonly reference: string
  readonly name: string
}

/** The tab in the words `the PDF tab at "a.pdf"`. */
export const parseTabRef = (text: string): TabRef => {
  const match = /^((?:[A-Za-z]+ ){0,2}tab) (at|titled|for the web address) "([^"]+)"$/.exec(text)
  if (!match) throw new Error(`"${text}" does not name a tab: write the tab at "path" or the tab titled "Title"`)
  return { kind: parseTabKind(match[1]), reference: match[2], name: match[3] }
}
