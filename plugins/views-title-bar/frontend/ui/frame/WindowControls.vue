<script setup lang="ts">
/**
 * The buttons that minimize, maximize and close the window, as squares at the right (`windows`) or
 * as three circles at the left (`mac`). On Windows each button is marked as the system button it
 * replaces, so the maximize button opens Snap Layouts.
 */
import { Copy, Minus, Square, X } from '@lucide/vue'
import type { WindowControlsWords } from '../../lib/frame'

/* --------------------------------- Props ---------------------------------- */
withDefaults(
  defineProps<{
    look: 'windows' | 'mac'
    isMaximized?: boolean
    words: WindowControlsWords
  }>(),
  { isMaximized: false },
)

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'minimize-window'): void
  (event: 'toggle-maximize-window'): void
  (event: 'close-window'): void
}>()
</script>

<template>
  <div class="window-controls" :class="`window-controls--${look}`" data-window-controls>
    <button
      type="button"
      class="window-controls__button window-controls__button--minimize"
      data-control="minimize"
      tabindex="-1"
      :title="words.minimize"
      style="--wails-draggable: no-drag; --wails-non-client-region: minimize"
      @click="emit('minimize-window')"
    >
      <Minus v-if="look === 'windows'" :size="14" />
    </button>
    <button
      type="button"
      class="window-controls__button window-controls__button--maximize"
      data-control="toggle-maximize"
      tabindex="-1"
      :data-maximized="String(isMaximized)"
      :title="isMaximized ? words.restore : words.maximize"
      style="--wails-draggable: no-drag; --wails-non-client-region: maximize"
      @click="emit('toggle-maximize-window')"
    >
      <template v-if="look === 'windows'">
        <Copy v-if="isMaximized" :size="12" />
        <Square v-else :size="12" />
      </template>
    </button>
    <button
      type="button"
      class="window-controls__button window-controls__button--close"
      data-control="close"
      tabindex="-1"
      :title="words.close"
      style="--wails-draggable: no-drag; --wails-non-client-region: close"
      @click="emit('close-window')"
    >
      <X v-if="look === 'windows'" :size="14" />
    </button>
  </div>
</template>
