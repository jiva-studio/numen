<script setup lang="ts">
/** The six digits the host shows, typed on the new device, and the tries that are left. */
import { WORDS as words } from '../../words'
import LinkField from './LinkField.vue'

/* ----------------------------- Props & Emits ------------------------------ */
defineProps<{ modelValue: string; triesLeft: number }>()

const emit = defineEmits<{ (e: 'update:modelValue', code: string): void }>()
</script>

<template>
  <div class="link-code">
    <LinkField
      :model-value="modelValue"
      :label="words.linkCodeField"
      is-mono
      inputmode="numeric"
      :maxlength="6"
      @update:model-value="emit('update:modelValue', $event)"
    />
    <p class="link-code__tries">
      {{ triesLeft > 0 && triesLeft < 3 ? words.linkTriesLeft(triesLeft) : '' }}
    </p>
  </div>
</template>

<style scoped>
.link-code {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.link-code__tries {
  min-block-size: calc(1em * var(--numen-line-height));
  margin: 0;
  color: var(--numen-hushed);
}
</style>
