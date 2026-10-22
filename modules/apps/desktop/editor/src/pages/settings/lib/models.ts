/**
 * The models a setting can be set to, as a list a person picks from.
 *
 * The file is what the installation is doing, and the presets are what this
 * build offers. What the file holds is always on the list and always the one in
 * force; a preset the file does not name is an offer beside it.
 */
import type { Model, Presence, SettingFields } from '@/entities/settings'
import type { SelectChoice } from '@numen/ui'

/** What this says in the window's voice. */
export interface Words {
  /** Said on the row an installation nobody has configured runs on. */
  readonly byDefault: string
  /** The group a person's own value relies on. */
  readonly owned: string
  /** What a model's files are on this machine, where that means anything. */
  readonly present: string
  readonly notFetched: string
}

/**
 * What a model is called, read off the value where the build names none: the
 * last segment of a path, a repository or an address.
 */
export const getLastPathSegment = (value: string): string => {
  const basePath = value.split(/[?#]/, 1)[0] ?? ''
  const last = basePath
    .split('/')
    .filter((one) => one.length > 0)
    .at(-1)
  return last ?? value
}

/** Whether a value is an address to somewhere, and not a word on its own. */
const isAddress = (value: string): boolean => value.includes('/')

/**
 * What a model's files are on this machine, in a word. A model reached over
 */
const getPresenceWord = (presence: Presence, words: Words): string => {
  if (presence === 'present') return words.present
  return presence === 'not fetched' ? words.notFetched : ''
}

/** The second line under a model's name: what it is, then where it is. */
const getDetail = (parts: readonly string[]): string => parts.filter(Boolean).join(' · ')

/**
 * What a model is called on the row. A model the build names in words is called
 * that; one carrying an address where its name would be is called by the last
 * segment of it, which is the part that says which model it is.
 */
const getModelName = (model: Model): string =>
  model.title && !isAddress(model.title) ? model.title : getLastPathSegment(model.name)

/** One preset, as the list offers it. */
const getModelChoice = (model: Model, words: Words): SelectChoice => {
  const name = getModelName(model)
  // A model addressed by a path, a repository or an address is named by its
  // own words and addressed under them.
  const detail = getDetail([
    getPresenceWord(model.presence, words),
    isAddress(model.name) ? model.name : '',
  ])
  return {
    id: model.name,
    text: model.isDefault ? `${name} — ${words.byDefault}` : name,
    ...(detail ? { detail } : {}),
    ...(model.group ? { group: model.group } : {}),
  }
}

/**
 * The choices for one setting: what the file holds, then the presets this build
 * offers. A value the presets do not name stands first, under the group that
 * says it is the person's own.
 */
export const createChoices = (
  models: readonly Model[],
  now: string,
  words: Words,
): readonly SelectChoice[] => {
  const presets = models.map((one) => getModelChoice(one, words))
  if (!now || presets.some((one) => one.id === now)) return presets
  const own: SelectChoice = {
    id: now,
    text: getLastPathSegment(now),
    ...(isAddress(now) ? { detail: now } : {}),
    group: words.owned,
  }
  return [own, ...presets]
}

/** The choices for one setting of the fields, read from what the file holds there. */
export const createModelChoices = (
  fields: Pick<SettingFields, 'getModels' | 'getString'>,
  path: readonly string[],
  words: Words,
): readonly SelectChoice[] => createChoices(fields.getModels(path), fields.getString(path), words)
