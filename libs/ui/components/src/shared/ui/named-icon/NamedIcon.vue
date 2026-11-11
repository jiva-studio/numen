<script setup lang="ts">
/**
 * A Lucide icon drawn by its name, such as a manifest writes it (`trash-2`). It draws nothing
 * until the icon is found, a placeholder for a name the set does not have, and nothing for an
 * empty name.
 *
 * It is drawn for the eye alone.
 */
import { useNamedIcon } from './useNamedIcon'

defineOptions({ inheritAttrs: false })

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  name: string
}>()

/* --------------------------------- State ---------------------------------- */
const icon = useNamedIcon(() => props.name)
</script>

<template>
  <component :is="icon" v-if="icon" v-bind="$attrs" aria-hidden="true" />
</template>
