/**
 * The rows of the popover: a label, a value and, where the meaning is not obvious, a one-line hint.
 */
import type { Kind, KindCount, Stats } from '../types'
import { WORDS as words } from '../words'
import { formatBytes, formatCount, formatRate, formatTime } from './format'

export interface Row {
  readonly id: string
  readonly label: string
  readonly value: string
  readonly hint: string
  readonly isNested: boolean
}

const KIND_LABELS: Readonly<Record<Kind, string>> = {
  notes: words.notes,
  pdf: words.pdf,
  epub: words.epub,
  media: words.media,
  other: words.other,
}

const PROFILE_LABELS: Readonly<Record<string, string>> = {
  eco: words.eco,
  balanced: words.balanced,
  maximum: words.maximum,
}

const row = (
  id: string,
  label: string,
  value: string,
  extra: Partial<Pick<Row, 'hint' | 'isNested'>> = {},
): Row => ({ id, label, value, hint: extra.hint ?? '', isNested: extra.isNested ?? false })

function getKindRow(one: KindCount, locale?: string): Row[] {
  if (one.sources <= 0 && one.chunks <= 0) return []
  const value = `${formatCount(one.chunks, locale)} ${words.chunkUnit} · ${formatCount(one.sources, locale)} ${words.fileUnit}`
  return [row(`kind-${one.kind}`, KIND_LABELS[one.kind], value, { isNested: true })]
}

function getProfileValue(stats: Stats, locale?: string): string {
  const name = PROFILE_LABELS[stats.profile] ?? stats.profile
  const unit = stats.threads === 1 ? words.thread : words.threadUnit
  return `${name} · ${formatCount(stats.threads, locale)} ${unit}`
}

function getActivityRows(stats: Stats, locale?: string): Row[] {
  const rows: Row[] = []
  if (stats.waiting > 0) {
    const value =
      stats.speed === undefined
        ? words.none
        : `${formatRate(stats.speed, locale)} ${words.perSecond}`
    rows.push(row('speed', words.speed, value, { hint: words.speedHint }))
  }
  const changed = stats.changedAt === undefined ? '' : formatTime(stats.changedAt, locale)
  rows.push(
    row('changed-at', words.changedAt, changed || words.none, { hint: words.changedAtHint }),
  )
  return rows
}

/** Every row of the popover, in the order they are shown. */
export function getRows(stats: Stats, locale?: string): Row[] {
  return [
    row('vectors', words.vectors, formatCount(stats.vectors, locale), { hint: words.vectorsHint }),
    row('chunks', words.chunks, formatCount(stats.chunks, locale), { hint: words.chunksHint }),
    row('waiting', words.waiting, formatCount(stats.waiting, locale), { hint: words.waitingHint }),
    row('sources', words.sources, formatCount(stats.sources, locale), { hint: words.sourcesHint }),
    ...stats.kinds.flatMap((one) => getKindRow(one, locale)),
    row('model', words.model, stats.model),
    row('recipe', words.recipe, stats.recipe, { hint: words.recipeHint }),
    row(
      'dimensions',
      words.dimensions,
      `${formatCount(stats.dimensions, locale)} ${words.dimensionUnit}`,
    ),
    row('profile', words.profile, getProfileValue(stats, locale), { hint: words.profileHint }),
    ...getActivityRows(stats, locale),
    row('database', words.databaseSize, formatBytes(stats.databaseBytes, locale)),
    row('vector-size', words.vectorSize, formatBytes(stats.vectorBytes, locale), {
      hint: words.vectorSizeHint,
    }),
  ]
}
