<script setup lang="ts">
/**
 * The plugin declarations that may read a secret: a select of the declarations that shows the
 * plugin allowed, or the first one and how many more. Choosing a plugin allows the declaration to
 * read the secret, or stops it.
 */
import { computed } from 'vue'
import type { SelectChoice } from '../../types'
import { MultiSelect } from '../multi-select'
import { isAllowed, type Secret } from '@/entities/settings'
import type { Declaration } from '../../lib/secretSections'
import { WORDS as words } from '../../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  secret?: Secret | undefined
  declarations: readonly Declaration[]
}>()

const emit = defineEmits<{
  (e: 'toggle', declaration: Declaration, isAllowed: boolean): void
}>()

/* --------------------------------- State ---------------------------------- */
const keyOf = (one: Declaration): string => `${one.pluginId}/${one.name}/${one.group}`
const choices = computed<readonly SelectChoice[]>(() =>
  props.declarations.map((one) => ({ id: keyOf(one), text: one.pluginName })),
)
const allowedIds = computed(() => {
  const secret = props.secret
  const allowed = props.declarations.filter((one) => secret !== undefined && isAllowed(secret, one))
  return allowed.map(keyOf)
})

/* -------------------------------- Handlers -------------------------------- */
function onToggle(id: string, isOn: boolean): void {
  const declaration = props.declarations.find((one) => keyOf(one) === id)
  if (declaration) emit('toggle', declaration, isOn)
}
</script>

<template>
  <MultiSelect
    :selected="allowedIds"
    :choices="choices"
    :placeholder="words.secretNoReaders"
    :name="words.columnUsedBy"
    :search-name="words.secretSearchPlugins"
    :get-more-text="words.secretMore"
    :is-disabled="!secret || declarations.length === 0"
    :title="secret ? undefined : words.secretSaveFirst"
    @toggle="onToggle"
  />
</template>
