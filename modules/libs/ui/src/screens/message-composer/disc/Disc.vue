<script setup lang="ts">
/* --------------------------------- Props ---------------------------------- */
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

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'press'): void
}>()

/* --------------------------------- State ---------------------------------- */
const named = computed(() => (props.action === 'stop' ? props.stopLabel : props.sendLabel))
</script>

<template>
  <div class="disc">
    <Transition name="disc__swap" mode="out-in">
      <Button
        :key="action"
        size="icon-small"
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
  block-size: 1.75rem;
  inline-size: 1.75rem;
  align-items: center;
  justify-content: center;
}

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
