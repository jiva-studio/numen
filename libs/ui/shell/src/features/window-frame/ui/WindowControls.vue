<script setup lang="ts">
/**
 * The buttons that minimise, maximise and close the window, as squares at the right (`windows`) or
 * as three circles at the left (`mac`). On Windows each button is marked as the system button it
 * stands for, so the maximise button opens Snap Layouts.
 */
import { computed } from 'vue'
import { Copy, Minus, Square, X } from '@lucide/vue'
import { WINDOW_CONTROLS_WORDS, type WindowControlsWords } from '../lib/types'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    look: 'windows' | 'mac'
    isMaximised?: boolean
    words?: Partial<WindowControlsWords>
  }>(),
  { isMaximised: false, words: () => ({}) },
)

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'minimise'): void
  (event: 'toggle-maximise'): void
  (event: 'close'): void
}>()

/* --------------------------------- State ---------------------------------- */
const allWords = computed(() => ({ ...WINDOW_CONTROLS_WORDS, ...props.words }))
</script>

<template>
  <div class="window-controls" :class="`window-controls--${look}`" data-window-controls>
    <button
      type="button"
      class="window-controls__button window-controls__button--minimise"
      data-control="minimise"
      tabindex="-1"
      :title="allWords.minimise"
      style="--wails-draggable: no-drag; --wails-non-client-region: minimize"
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
      :title="isMaximised ? allWords.restore : allWords.maximise"
      style="--wails-draggable: no-drag; --wails-non-client-region: maximize"
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
      :title="allWords.close"
      style="--wails-draggable: no-drag; --wails-non-client-region: close"
      @click="emit('close')"
    >
      <X v-if="look === 'windows'" :size="14" />
    </button>
  </div>
</template>
