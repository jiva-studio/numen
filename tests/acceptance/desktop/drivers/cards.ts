/**
 * The files of the flashcards plugin as the product reads them: card models, decks and presets,
 * built from the words of a table in a scenario. A card model lists its fields in its frontmatter
 * and its templates as sections; a deck is cards, each a section naming its card model; a preset
 * is settings in its frontmatter.
 */
import { createHash } from 'node:crypto'

export interface Template {
  readonly name: string
  readonly front: string
  readonly back: string
}

/** The folder of the vault where the steps put card models, decks and presets. */
const CARDS_FOLDER = 'cards'

/** The letters a card id is written in (the idPattern of the flashcards manifest). */
const CARD_ID_LETTERS = '0123456789abcdefghjkmnpqrstvwxyz'
const CARD_ID_LENGTH = 10
const CARD_HEADING = /^## .* \^[0-9a-hjkmnp-tv-z]{10}$/gm

export const getCardFile = (title: string): string => {
  if (title.trim() === '' || /[/\\:*?"<>|#]/.test(title)) {
    throw new Error(`"${title}" cannot name a file: leave out / \\ : * ? " < > | #`)
  }
  return `${CARDS_FOLDER}/${title.trim()}.md`
}

/** The type a card file declares in its frontmatter: card-model, deck or preset. */
const readCardType = (text: string): string | undefined => {
  const frontmatter = /^---\r?\n([\s\S]*?)\r?\n---/.exec(text)?.[1] ?? ''
  return /^type:\s*(\S+)\s*$/m.exec(frontmatter)?.[1]
}

export const assertCardFileIsFree = (vault: string, title: string, existing: string | undefined): void => {
  if (existing !== undefined) {
    throw new Error(`the vault ${vault} already has ${getCardFile(title)}: a card model, a deck and a preset need different titles`)
  }
}

const CARD_KIND_NAMES: Readonly<Record<string, string>> = { 'card-model': 'card model', deck: 'deck', preset: 'preset' }

/** The text of the card file to add to; fails when the file is missing or is not of the kind named. */
export const assertCardFileKind = (existing: string | undefined, title: string, kind: 'card model' | 'deck'): string => {
  const hint = kind === 'card model' ? 'make it with its fields first' : 'make it first'
  if (existing === undefined) throw new Error(`the vault has no ${kind} "${title}": ${hint}`)
  const found = CARD_KIND_NAMES[readCardType(existing) ?? ''] ?? 'file of another type'
  if (found !== kind) throw new Error(`the vault has a ${found} "${title}", not a ${kind}`)
  return existing
}

const quote = (text: string): string => (/^[\w][\w ]*$/u.test(text) ? text : JSON.stringify(text))

const assertHeader = (rows: readonly string[][], header: readonly string[], what: string): string[][] => {
  const [first, ...body] = rows
  if (rows.length === 0 || first.join('|') !== header.join('|')) {
    throw new Error(`${what} is a table with the header ${header.map((one) => `"${one}"`).join(', ')}`)
  }
  if (body.length === 0) throw new Error(`${what} has no row`)
  return body
}

const assertColumnsAreNew = (columns: readonly string[], what: string): void => {
  const repeated = columns.find((one, index) => columns.indexOf(one) !== index)
  if (repeated !== undefined) throw new Error(`${what} has the column "${repeated}" twice`)
}

const parseNames = (cell: string, title: string): string[] => {
  const names = cell.split(',').map((one) => one.trim())
  if (names.some((one) => one === '')) throw new Error(`the fields of the card model "${title}" are names with commas between them`)
  if (new Set(names).size !== names.length) throw new Error(`the fields ${names.join(', ')} name one field twice`)
  return names
}

export interface CardModel {
  readonly title: string
  readonly fields: string[]
  readonly templates: Template[]
}

const CARD_MODEL_HEADER = ['card model', 'fields', 'template', 'front', 'back']

/** The card model of a row: a first row makes it with its fields, and a later row takes it as it is. */
const getModel = (models: Map<string, CardModel>, title: string, fields: string): CardModel => {
  const known = models.get(title)
  if (known !== undefined) {
    if (fields !== '') {
      throw new Error(`the card model "${title}" has its fields in more than one row: write them in its first row only`)
    }
    return known
  }
  if (fields === '') throw new Error(`the card model "${title}" has no fields: write them in its first row`)
  const model: CardModel = { title, fields: parseNames(fields, title), templates: [] }
  models.set(title, model)
  return model
}

/** The card models of a table: the fields stand in the first row of a model, and each row gives a template. */
export const parseCardModels = (rows: readonly string[][]): CardModel[] => {
  const models = new Map<string, CardModel>()
  for (const [title, fields, name, front, back] of assertHeader(rows, CARD_MODEL_HEADER, 'the card models')) {
    if (title === '') throw new Error('a row of the card models has no card model: write its title in the first column')
    const model = getModel(models, title, fields)
    if (name === '' && (front !== '' || back !== '')) throw new Error(`a template of the card model "${title}" has no name`)
    if (name === '') continue
    if (model.templates.some((one) => one.name === name)) {
      throw new Error(`the card model "${title}" has the template "${name}" twice`)
    }
    model.templates.push({ name, front, back })
  }
  return [...models.values()]
}

export interface DeckCard {
  readonly deck: string
  readonly model: string | undefined
  readonly values: [string, string][]
}

/** The cards of a table: after the deck and the card model each column is a field; a row with only a deck makes the deck. */
export const parseCards = (rows: readonly string[][]): DeckCard[] => {
  const [header = [], ...body] = rows
  if (header[0] !== 'deck' || header[1] !== 'card model' || header.slice(2).some((one) => one === '')) {
    throw new Error('the cards are a table with the columns "deck" and "card model", then one column for each field')
  }
  const fields = header.slice(2)
  assertColumnsAreNew(header, 'the cards')
  if (body.length === 0) throw new Error('the cards have no row')
  return body.map(([deck, model, ...values]) => {
    if (deck === '') throw new Error('a card has no deck: write its deck in the first column')
    if (model === '' && values.some((one) => one !== '')) {
      throw new Error(`a card in the deck "${deck}" has values and no card model: write the card model in the second column`)
    }
    const pairs = fields.map((field, index): [string, string] => [field, values[index]])
    return { deck, model: model === '' ? undefined : model, values: pairs }
  })
}

export const createCardModelText = (title: string, fields: readonly string[]): string =>
  `---\ntype: card-model\ntitle: ${quote(title)}\nfields:\n${fields.map((field) => `  - ${quote(field)}\n`).join('')}---\n`

export const createTemplatesText = (templates: readonly Template[]): string =>
  templates.map((one) => `\n## ${one.name}\n\n### Front\n\n${one.front}\n\n### Back\n\n${one.back}\n`).join('')

/** The fields a card model file lists in its frontmatter. */
export const readCardModelFields = (text: string): string[] => {
  const frontmatter = /^---\r?\n([\s\S]*?)\r?\n---/.exec(text)?.[1] ?? ''
  const list = /^fields:\s*\r?\n((?:\s+-\s+.*\r?\n?)*)/m.exec(frontmatter)?.[1] ?? ''
  return [...list.matchAll(/^\s+-\s+(.*?)\s*$/gm)].map((one) => one[1].replace(/^"(.*)"$/, '$1'))
}

export const createDeckText = (title: string): string => `---\ntype: deck\ntitle: ${quote(title)}\n---\n`

/** The id the nth card of a deck has: ten letters no other card of the deck repeats. */
export const createCardId = (deck: string, index: number): string => {
  const digest = createHash('sha1').update(`${deck}:${index}`).digest()
  return Array.from({ length: CARD_ID_LENGTH }, (_, position) => CARD_ID_LETTERS[digest[position] % CARD_ID_LETTERS.length]).join('')
}

export const countCards = (deckText: string): number => deckText.match(CARD_HEADING)?.length ?? 0

/** The section of a card: headed by its first value, naming its card model, one section for each value that has text. */
export const createCardText = (
  model: string,
  fields: readonly string[],
  values: readonly (readonly [string, string])[],
  id: string,
): string => {
  for (const [field] of values) {
    if (!fields.includes(field)) {
      throw new Error(`the card model "${model}" has no field "${field}": it has ${fields.join(', ')}`)
    }
  }
  const filled = values.filter(([, value]) => value !== '')
  if (filled.length === 0) throw new Error(`a card of the card model "${model}" has no value`)
  const heading = filled.find(([field]) => field === fields[0])?.[1] ?? filled[0][1]
  const sections = filled.map(([field, value]) => `### ${field}\n\n${value}\n`)
  return `\n## ${heading} ^${id}\n\n[[${model}]]\n\n${sections.join('\n')}`
}

interface PresetSetting {
  readonly key: string
  readonly parse: (value: string) => string
}

const parseNumber = (value: string): string => {
  const number = /^(\d+(?:\.\d+)?)%?$/.exec(value.trim())
  if (number === null) throw new Error(`"${value}" is not a number`)
  return number[1]
}

const parseShare = (value: string): string => {
  if (!value.trim().endsWith('%')) throw new Error(`"${value}" is not a percentage: write it like 87%`)
  return String(Number(parseNumber(value)) / 100)
}

const parseChoice =
  (choices: Record<string, string>) =>
  (value: string): string => {
    if (!Object.hasOwn(choices, value)) throw new Error(`"${value}" is not one of ${Object.keys(choices).join(', ')}`)
    return choices[value]
  }

/** The settings of a preset by the name its tab gives them (the labels of the preset frontend), with the key of the frontmatter. */
const PRESET_SETTINGS: Readonly<Record<string, PresetSetting>> = {
  Goal: { key: 'goal', parse: parseChoice({ 'Minutes per day': 'minutes_a_day', Retention: 'retention', 'Target date': 'by_date' }) },
  'New per day': { key: 'new_a_day', parse: parseNumber },
  'Reviews per day': { key: 'reviews_a_day', parse: parseNumber },
  'Minutes per day': { key: 'minutes_a_day', parse: parseNumber },
  Retention: { key: 'retention', parse: parseShare },
  'Target date': { key: 'by_date', parse: (value) => value.trim() },
  'Budget unit': { key: 'counts', parse: parseChoice({ Cards: 'cards', Reviews: 'shows' }) },
  'Learned when': { key: 'learned', parse: parseChoice({ 'By days': 'interval', 'By remembering': 'retention' }) },
  'Days apart': { key: 'interval', parse: parseNumber },
  'Overdue share': { key: 'backlog', parse: parseNumber },
  'Even load': { key: 'even_load', parse: parseChoice({ Yes: 'true', No: 'false' }) },
}

const parseSetting = (name: string, value: string): string => {
  try {
    return PRESET_SETTINGS[name].parse(value)
  } catch (error) {
    throw new Error(`the setting "${name}": ${(error as Error).message}`, { cause: error })
  }
}

export interface Preset {
  readonly title: string
  readonly settings: [string, string][]
}

/** The presets of a table: after the title each column is a setting, named as the preset tab names it; an empty cell leaves it out. */
export const parsePresets = (rows: readonly string[][]): Preset[] => {
  const [header = [], ...body] = rows
  if (header[0] !== 'preset') {
    throw new Error('the presets are a table that starts with the column "preset", then one column for each setting')
  }
  const names = header.slice(1)
  for (const name of names) {
    if (!Object.hasOwn(PRESET_SETTINGS, name)) {
      throw new Error(`"${name}" is not a setting of a preset: use ${Object.keys(PRESET_SETTINGS).join(', ')}`)
    }
  }
  assertColumnsAreNew(header, 'the presets')
  if (body.length === 0) throw new Error('the presets have no row')
  return body.map(([title, ...values]) => {
    if (title === '') throw new Error('a preset has no title: write it in the first column')
    const settings: [string, string][] = []
    for (const [index, name] of names.entries()) {
      if (values[index] !== '') settings.push([PRESET_SETTINGS[name].key, parseSetting(name, values[index])])
    }
    return { title, settings }
  })
}

export const createPresetText = (title: string, settings: readonly (readonly [string, string])[]): string =>
  `---\ntype: preset\ntitle: ${quote(title)}\n${settings.map(([key, value]) => `${key}: ${value}\n`).join('')}---\n\n# ${title}\n`
