<script setup lang="ts">
/**
 * A Lucide icon drawn by its name, such as a manifest writes it (`trash-2`). It draws nothing
 * until the icon is found, and nothing for a name with no icon.
 *
 * It is drawn for the eye alone.
 */
import { shallowRef, watch, type Component } from 'vue'
import { findNamedIcon } from './icons'

defineOptions({ inheritAttrs: false })

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  /** The Lucide name of the icon, in lowercase words joined by hyphens. */
  name: string
}>()

/* --------------------------------- State ---------------------------------- */
const icon = shallowRef<Component | null>(null)

/* --------------------------------- Hooks ---------------------------------- */
watch(
  () => props.name,
  async (name) => {
    icon.value = null
    const found = await findNamedIcon(name)
    if (props.name === name) icon.value = found
  },
  { immediate: true },
)
</script>

<template>
  <component :is="icon" v-if="icon" v-bind="$attrs" aria-hidden="true" />
</template>
