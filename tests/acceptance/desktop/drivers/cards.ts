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
export const CARDS_FOLDER = 'cards'

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
export const readCardType = (text: string): string | undefined => {
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

export const assertTemplatesAreNew = (modelText: string, title: string): void => {
  if (/^## /m.test(modelText)) throw new Error(`the card model "${title}" already has its templates: they are given once`)
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

/** The names of the fields of a card model: one row of names. */
export const parseFieldNames = (rows: readonly string[][]): string[] => {
  if (rows.length !== 1 || rows[0].length === 0 || rows[0].some((one) => one.trim() === '')) {
    throw new Error('the fields are one row of names, like | Name | Height |')
  }
  const names = rows[0].map((one) => one.trim())
  if (new Set(names).size !== names.length) throw new Error(`the fields ${names.join(', ')} name one field twice`)
  return names
}

export const parseTemplates = (rows: readonly string[][]): Template[] =>
  assertHeader(rows, ['Template', 'Front', 'Back'], 'the templates').map(([name, front, back]) => ({ name, front, back }))

export const parseValues = (rows: readonly string[][]): [string, string][] =>
  assertHeader(rows, ['Field', 'Value'], 'the values of a card').map(([field, value]) => [field, value])

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

/** The section of a card: headed by its first value, naming its card model, one section for each value. */
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
  const heading = values.find(([field]) => field === fields[0])?.[1] ?? values[0][1]
  const sections = values.map(([field, value]) => `### ${field}\n\n${value}\n`)
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

export const parsePresetSettings = (rows: readonly string[][]): [string, string][] => {
  const settings = assertHeader(rows, ['Setting', 'Value'], 'the settings of a preset')
  return settings.map(([name, value]) => {
    if (!Object.hasOwn(PRESET_SETTINGS, name)) {
      throw new Error(`"${name}" is not a setting of a preset: use ${Object.keys(PRESET_SETTINGS).join(', ')}`)
    }
    const setting = PRESET_SETTINGS[name]
    try {
      return [setting.key, setting.parse(value)]
    } catch (error) {
      throw new Error(`the setting "${name}": ${(error as Error).message}`, { cause: error })
    }
  })
}

export const createPresetText = (title: string, settings: readonly (readonly [string, string])[]): string =>
  `---\ntype: preset\ntitle: ${quote(title)}\n${settings.map(([key, value]) => `${key}: ${value}\n`).join('')}---\n\n# ${title}\n`
