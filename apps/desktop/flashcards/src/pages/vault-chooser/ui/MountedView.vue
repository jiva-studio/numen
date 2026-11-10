<script setup lang="ts">
/**
 * A view declared by a plugin, rendered into an element of the window. The caller says how it
 * is mounted, and it is read when the view mounts: a list row is keyed by the item it represents.
 */
import { onBeforeUnmount, onMounted, useTemplateRef } from 'vue'
import type { MountedView } from '@numen/plugin-loader'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  mount: (element: HTMLElement) => MountedView
}>()

/* --------------------------------- State ---------------------------------- */
const element = useTemplateRef<HTMLElement>('element')
let view: MountedView | null = null

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => {
  if (element.value) view = props.mount(element.value)
})
onBeforeUnmount(() => view?.dispose())
</script>

<template>
  <span ref="element" class="mounted-view" />
</template>

<style scoped>
.mounted-view {
  display: inline-flex;
}
</style>
