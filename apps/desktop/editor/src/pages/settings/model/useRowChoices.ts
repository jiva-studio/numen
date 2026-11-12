import { computed, ref, watch } from 'vue'
import { runInBackground } from '@numen/ui'
import type { ConnectionFieldInfo, RowDraft, RowOption } from '@/entities/connection'
import { formatError } from '@/shared/api/errors'
import type { Result } from '@/shared/lib/result'
import { WORDS as words } from '../words'
import type { ListRowOptions, SelectChoice } from '../types'

/**
 * What a row reads from the provider of its connection and how it answers. getRowFields returns none while
 * the row names no connection.
 */
export interface RowChoicesSource {
  readonly getRow: () => RowDraft
  readonly getRowFields: () => readonly ConnectionFieldInfo[]
  readonly getOperations: () => readonly string[]
  readonly getListRowOptions: () => ListRowOptions | undefined
}

/** One row field as the row draws it: its choices, why they could not be listed, and whether it is open. */
export interface RowFieldChoices {
  readonly field: ConnectionFieldInfo
  readonly value: string
  readonly choices: readonly SelectChoice[]
  readonly problem: string
  readonly isDisabled: boolean
}

interface Listed {
  readonly options: readonly RowOption[]
  readonly problem: string
}

async function readAnswer(
  pending: Promise<Result<readonly RowOption[], string>>,
): Promise<Result<readonly RowOption[], string>> {
  try {
    return await pending
  } catch (caughtError) {
    return { ok: false, error: formatError(caughtError) }
  }
}

const readChoice = (one: RowOption): SelectChoice => ({
  id: one.value,
  text: one.title || one.value,
  ...(one.title === '' || one.title === one.value ? {} : { detail: one.value }),
  ...(one.group === '' ? {} : { group: one.group }),
})

/**
 * The choices of the row fields of the connection a row names. A field that names optionsFrom is asked
 * from the provider whenever the connection or a field before it changes, with the values the row holds
 * so far; a field after an empty required field waits. A field that is not required offers the default
 * of the provider first. No value is chosen for the person: a required field that is open and empty
 * asks for its value. An answer that comes after the row changed again is dropped.
 */
export function useRowChoices(source: RowChoicesSource) {
  const listed = ref<Readonly<Record<string, Listed>>>({})
  const requests = new Map<string, number>()

  const getValue = (field: ConnectionFieldInfo): string => source.getRow().options[field.key] ?? ''

  const isWaiting = (index: number): boolean =>
    source.getRow().connection === '' ||
    source
      .getRowFields()
      .slice(0, index)
      .some((one) => one.isRequired && getValue(one) === '')

  const readProblem = (
    field: ConnectionFieldInfo,
    index: number,
    listing: Listed | undefined,
  ): string => {
    if (listing && listing.problem !== '') return listing.problem
    if (!field.isRequired || getValue(field) !== '' || isWaiting(index)) return ''
    return words.connectionRowsChooseValue(field.title)
  }

  const fields = computed<readonly RowFieldChoices[]>(() =>
    source.getRowFields().map((field, index) => {
      const asked = listed.value[field.key]
      const options = field.optionsFrom
        ? (asked?.options ?? [])
        : field.options.map((one) => ({ ...one, group: '' }))
      const choices = [
        ...(field.isRequired ? [] : [{ id: '', text: words.connectionRowsDefault }]),
        ...options.map(readChoice),
      ]
      return {
        field,
        value: getValue(field),
        choices,
        problem: readProblem(field, index, asked),
        isDisabled:
          isWaiting(index) || (!field.isRequired && field.type !== 'string' && choices.length <= 1),
      }
    }),
  )

  const ask = async (field: ConnectionFieldInfo, index: number): Promise<void> => {
    const request = (requests.get(field.key) ?? 0) + 1
    requests.set(field.key, request)
    listed.value = Object.fromEntries(
      Object.entries(listed.value).filter(([key]) => key !== field.key),
    )
    const list = source.getListRowOptions()
    if (!field.optionsFrom || isWaiting(index) || !list) return
    const row = source.getRow()
    const answer = await readAnswer(
      list(row.connection, field.key, row.options, source.getOperations()),
    )
    if (requests.get(field.key) !== request) return
    listed.value = {
      ...listed.value,
      [field.key]: answer.ok
        ? { options: answer.value, problem: '' }
        : { options: [], problem: words.connectionRowsOptionsFailed(field.title, answer.error) },
    }
  }

  watch(
    () =>
      source.getRowFields().map((field, index) =>
        JSON.stringify([
          source.getRow().connection,
          field.key,
          ...source
            .getRowFields()
            .slice(0, index)
            .map((one) => getValue(one)),
        ]),
      ),
    (next, previous) => {
      source.getRowFields().forEach((field, index) => {
        if (previous?.[index] !== next[index]) runInBackground(ask(field, index), () => undefined)
      })
    },
    { immediate: true },
  )

  return { fields }
}
