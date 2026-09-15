<script setup lang="ts">
/**
 * The disc at the end of the field. One disc either way; the glyph on it is
 * what changes, and one gives way to the other.
 */
import { computed } from 'vue'
import { Button } from '@/shared/ui/button'
import { Glyph } from './glyph'
import type { ComposerAction } from '../state'

const props = defineProps<{
  /** What pressing it does. */
  action: ComposerAction
  disabled: boolean
  /** What it is called while it sends. */
  sendLabel: string
  /** What it is called while it stops. */
  stopLabel: string
}>()

const emit = defineEmits<{
  (event: 'press'): void
}>()

/** What the disc is called. */
const named = computed(() => (props.action === 'stop' ? props.stopLabel : props.sendLabel))
</script>

<template>
  <div class="disc">
    <Transition name="disc__swap" mode="out-in">
      <Button
        :key="action"
        size="icon"
        :disabled="disabled"
        :aria-label="named"
        @click="emit('press')"
      >
        <slot v-if="action === 'send' && $slots.glyph" name="glyph" />
        <Glyph v-else :action="action" />
      </Button>
    </Transition>
  </div>
</template>

<style scoped>
.disc {
  display: flex;
  block-size: var(--numen-action-size);
  align-items: center;
  justify-content: center;
}

/* One disc gives way to the other, both halves together taking as long as a
   hover. */
.disc__swap-enter-active,
.disc__swap-leave-active {
  transition:
    opacity var(--numen-motion-hover) var(--numen-easing),
    transform var(--numen-motion-hover) var(--numen-easing);
}

.disc__swap-enter-from,
.disc__swap-leave-to {
  opacity: 0;
  transform: scale(0.75);
}
</style>
