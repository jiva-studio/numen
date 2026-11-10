<script setup lang="ts">
/* --------------------------------- Props ---------------------------------- */
import { computed } from 'vue'
import { Button } from '@numen/ui'
import { Glyph } from './glyph'
import type { ComposerAction } from '../../model/state'

const props = defineProps<{
  action: ComposerAction
  isDisabled: boolean
  sendLabel: string
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
    <Button
      size="icon-small"
      :data-action="action"
      :disabled="isDisabled"
      :aria-label="named"
      @click="emit('press')"
    >
      <slot v-if="action === 'send' && $slots.glyph" name="glyph" />
      <Glyph v-else :action="action" />
    </Button>
  </div>
</template>

<style scoped>
.disc {
  --disc-size: 1.75rem;
  display: flex;
  block-size: var(--disc-size);
  inline-size: var(--disc-size);
  align-items: center;
  justify-content: center;
}
</style>
