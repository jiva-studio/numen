<script setup lang="ts">
/**
 * A view declared by a plugin, rendered into an element of the window. The context is
 * read when the view mounts: a list row is keyed by the item it represents.
 */
import { onBeforeUnmount, onMounted, useTemplateRef } from 'vue'
import type { MountedView } from '@numen/plugin-loader'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  capability: string
  context: unknown
  mountView: (capability: string, element: HTMLElement, context: unknown) => MountedView
}>()

/* --------------------------------- State ---------------------------------- */
const element = useTemplateRef<HTMLElement>('element')
let view: MountedView | null = null

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => {
  if (element.value) view = props.mountView(props.capability, element.value, props.context)
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
