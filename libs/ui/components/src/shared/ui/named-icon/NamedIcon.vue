<script setup lang="ts">
/**
 * A Lucide icon drawn by its name, such as a manifest writes it (`trash-2`). It draws nothing
 * until the icon is found, a placeholder for a name the set does not have, and nothing for an
 * empty name.
 *
 * It is drawn for the eye alone.
 */
import { shallowRef, watch, type Component } from 'vue'
import { findNamedIcon } from './icons'

defineOptions({ inheritAttrs: false })

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  name: string
}>()

/** The icon drawn for a name the set does not have. */
const PLACEHOLDER_ICON = 'circle-question-mark'

/* --------------------------------- State ---------------------------------- */
const icon = shallowRef<Component | null>(null)

/* --------------------------------- Hooks ---------------------------------- */
watch(
  () => props.name,
  async (name) => {
    icon.value = null
    const found =
      (await findNamedIcon(name)) ?? (name === '' ? null : await findNamedIcon(PLACEHOLDER_ICON))
    if (props.name === name) icon.value = found
  },
  { immediate: true },
)
</script>

<template>
  <component :is="icon" v-if="icon" v-bind="$attrs" aria-hidden="true" />
</template>
