import { computed, watch, type Ref } from 'vue'
import { readRows, type ConnectionInfo, type RowDraft } from '@/entities/connection'
import type { JsonValue } from '@/shared/api/json'
import { WORDS as words } from '../words'

/** A setting of rows as the discovery of its connections reads it. */
export interface AutomaticRowsSource {
  readonly isDiscovering: () => boolean
  readonly getValue: () => JsonValue | undefined
  readonly stored: Readonly<Ref<readonly ConnectionInfo[]>>
  readonly discovered: Readonly<Ref<readonly ConnectionInfo[]>>
  readonly isSingle: Readonly<Ref<boolean>>
  readonly drafts: Readonly<Ref<readonly RowDraft[]>>
  readonly add: () => void
}

/**
 * Whether a setting of rows runs on the connections it discovers: while discovery is on, a connection
 * exists and no row is written. The line names those connections, or says that none serves. A setting
 * of one row that does not run on them shows its row at once, so that its connection is chosen in place.
 */
export function useAutomaticRows(source: AutomaticRowsSource) {
  const isAutomatic = computed(
    () =>
      source.isDiscovering() &&
      source.stored.value.length > 0 &&
      readRows(source.getValue()).length === 0,
  )
  const automaticText = computed(() =>
    source.discovered.value.length > 0
      ? words.connectionRowsAutomatic(source.discovered.value.map((one) => one.name))
      : words.connectionRowsNoneServes,
  )
  watch(
    [() => source.stored.value.length, () => source.drafts.value.length, () => source.isSingle.value, isAutomatic],
    () => {
      const isEmpty = source.stored.value.length > 0 && source.drafts.value.length === 0
      if (source.isSingle.value && !isAutomatic.value && isEmpty) source.add()
    },
    { immediate: true },
  )
  return { isAutomatic, automaticText }
}
