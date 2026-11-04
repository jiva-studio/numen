<script setup lang="ts">
/**
 * The buttons that minimise, maximise and close the window, as squares at the right (`windows`) or
 * as three circles at the left (`mac`).
 */
import { Copy, Minus, Square, X } from '@lucide/vue'
import type { TitleBarWords } from '../model/words'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  look: 'windows' | 'mac'
  isMaximised: boolean
  words: TitleBarWords
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'minimise'): void
  (event: 'toggle-maximise'): void
  (event: 'close'): void
}>()
</script>

<template>
  <div class="window-controls" :class="`window-controls--${look}`">
    <button
      type="button"
      class="window-controls__button window-controls__button--minimise"
      data-control="minimise"
      tabindex="-1"
      :title="words.minimise"
      style="--wails-draggable: no-drag"
      @click="emit('minimise')"
    >
      <Minus v-if="look === 'windows'" :size="14" />
    </button>
    <button
      type="button"
      class="window-controls__button window-controls__button--maximise"
      data-control="toggle-maximise"
      tabindex="-1"
      :data-maximised="String(isMaximised)"
      :title="isMaximised ? words.restore : words.maximise"
      style="--wails-draggable: no-drag"
      @click="emit('toggle-maximise')"
    >
      <template v-if="look === 'windows'">
        <Copy v-if="isMaximised" :size="12" />
        <Square v-else :size="12" />
      </template>
    </button>
    <button
      type="button"
      class="window-controls__button window-controls__button--close"
      data-control="close"
      tabindex="-1"
      :title="words.close"
      style="--wails-draggable: no-drag"
      @click="emit('close')"
    >
      <X v-if="look === 'windows'" :size="14" />
    </button>
  </div>
</template>
