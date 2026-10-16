<script setup lang="ts">
/**
 * Standalone preset editor tab component for Flashcards Studio plugin.
 */
import { ref } from 'vue'

/* ----------------------------- Props & Emits ------------------------------ */
interface PresetTabStateShape {
  errorMessage?: string
  hasExternalChange?: boolean
  reload?(): void
  setRetention?(retention: number): void
  setGoal?(goal: string): void
}

const props = defineProps<{
  state?: PresetTabStateShape
  path?: string
  at?: string
}>()

/* --------------------------------- State ---------------------------------- */
const targetRetention = ref(90)

/* -------------------------------- Handlers -------------------------------- */
function onRefresh() {
  props.state?.reload?.()
}

function onRetentionChange(event: Event) {
  const target = event.target as HTMLInputElement
  const val = Number(target.value)
  targetRetention.value = val
  props.state?.setRetention?.(val / 100)
}
</script>

<template>
  <div class="preset-editor-tab">
    <p v-if="props.state?.errorMessage" role="alert" class="preset__warning">
      {{ props.state.errorMessage }}
      <button type="button" @click="onRefresh">Refresh</button>
    </p>
    <div class="preset-controls">
      <label for="retention-slider">Target Retention: {{ targetRetention }}%</label>
      <input
        id="retention-slider"
        type="range"
        min="70"
        max="99"
        :value="targetRetention"
        @input="onRetentionChange"
      />
    </div>
  </div>
</template>
