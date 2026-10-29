<script setup lang="ts">
/**
 * A query typed by the person, checked against the threshold in force.
 */
import { computed, ref } from 'vue'
import { Button } from '@numen/ui'
import type { FloorProbeResult } from '../model/types'
import { describeProbe } from '../model/present'
import { WORDS } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  result: FloorProbeResult | null
  error: string
  isBusy: boolean
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  check: [query: string]
}>()

/* --------------------------------- State ---------------------------------- */
const query = ref('')
const verdict = computed(() =>
  props.result === null ? '' : describeProbe(props.result.bestSimilarity, props.result.doesPass),
)

/* -------------------------------- Handlers -------------------------------- */
function onSubmit(): void {
  emit('check', query.value)
}
</script>

<template>
  <form class="flex flex-col gap-1" @submit.prevent="onSubmit">
    <div class="flex items-center gap-2">
      <input
        v-model="query"
        type="text"
        class="border-field-rule bg-field text-ink rounded-node h-8 min-w-0 flex-1 border px-2 text-base"
        :aria-label="WORDS.probeLabel"
        :placeholder="WORDS.probePlaceholder"
        data-testid="floor-probe-input"
      />
      <Button
        type="submit"
        variant="outline"
        :disabled="isBusy || query.trim() === ''"
        data-testid="floor-probe-button"
      >
        {{ isBusy ? WORDS.probing : WORDS.probeButton }}
      </Button>
    </div>
    <p
      v-if="verdict !== ''"
      class="text-small text-ink"
      role="status"
      data-testid="floor-probe-result"
    >
      {{ verdict }}
    </p>
    <p
      v-if="error !== ''"
      class="text-small text-alarm"
      role="alert"
      data-testid="floor-probe-error"
    >
      {{ error }}
    </p>
  </form>
</template>
