<script setup lang="ts">
/**
 * The one modal of the interface, for every window and plugin: a dimmed backdrop and a panel in its middle with
 * a title, a hint, the body and a row of buttons at the right. The panel is `sm` (22rem), `md` (28rem) or `lg`
 * (36rem) wide and as high as its content. It covers the area the window gave with `provideDialogHost` (the content
 * between the title bar and the status bar) and the whole window where none was given. Escape and a press on the backdrop emit `cancel`, Tab stays inside
 * the panel, the first control takes the focus when the panel opens and each time `step` changes, and the focus
 * returns to the control that had it before.
 */
import type { DialogEvents, DialogProps } from './Dialog.types'
import DialogPanel from './DialogPanel.vue'
import { useDialogHost } from './host'

/* --------------------------------- Props ---------------------------------- */
defineProps<DialogProps>()

const emit = defineEmits<DialogEvents>()

/* --------------------------------- State ---------------------------------- */
const host = useDialogHost()
</script>

<template>
  <Teleport :to="host ?? 'body'">
    <DialogPanel
      v-bind="$props"
      :scope="host === null ? 'window' : 'host'"
      @submit="emit('submit')"
      @cancel="emit('cancel')"
    >
      <slot />
      <template v-if="$slots['actions']" #actions><slot name="actions" /></template>
    </DialogPanel>
  </Teleport>
</template>
