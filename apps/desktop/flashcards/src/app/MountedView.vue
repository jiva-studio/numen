<script setup lang="ts">
/**
 * A view a plugin declares, drawn into an element of the window. The context is
 * read as the view is mounted: a row of a list is keyed by what it stands for.
 */
import { onBeforeUnmount, onMounted, useTemplateRef } from 'vue'
import type { MountedView } from '@numen/plugin-sdk'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  /** What the view does, as the plugin declaring it names it. */
  capability: string
  /** What the view is passed. */
  context: unknown
  /** How the window draws a declared view into an element. */
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
