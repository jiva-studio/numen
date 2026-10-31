<script setup lang="ts">
/**
 * The search relevance section of the plugin's settings: the threshold below which a match is
 * not shown, how it was chosen, and the noise measured on the vault.
 */
import { computed, onMounted, watch } from 'vue'
import { Button, SegmentedControl, Slider } from '@numen/ui'
import { createSearchClient } from '@numen/plugin-sdk'
import { useFloor } from '../model/useFloor'
import { formatFloor, formatNoise, getCalibrationNote, getSourceLabel } from '../model/present'
import type { FloorClient } from '../model/types'
import FloorHistogram from './FloorHistogram.vue'
import FloorProbe from './FloorProbe.vue'
import { SLIDER_MAX, SLIDER_MIN, SLIDER_STEP, WORDS } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  pluginId: string
  values: Record<string, unknown>
  /** The client of the core search service. */
  client?: FloorClient
}>()

/* --------------------------------- State ---------------------------------- */
const floor = useFloor({ client: props.client ?? createSearchClient() })
const {
  state,
  status,
  loadError,
  actionError,
  isCalibrating,
  isTooFew,
  isFileSet,
  isManual,
  manualValue,
  probeResult,
  probeError,
  isProbing,
} = floor

const choices = [
  { id: 'auto', text: WORDS.modeAuto },
  { id: 'manual', text: WORDS.modeManual },
]
const mode = computed({
  get: () => (isManual.value ? 'manual' : 'auto'),
  set: (id: string) => {
    floor.setMode(id === 'manual').catch(floor.reportFailure)
  },
})
const slider = computed({
  get: () => manualValue.value,
  set: (value: number) => {
    floor.setManualFloor(value).catch(floor.reportFailure)
  },
})
const shownFloor = computed(() => (state.value === null ? '' : formatFloor(state.value.floor)))
const sourceLabel = computed(() => (state.value === null ? '' : getSourceLabel(state.value.source)))
const note = computed(() =>
  state.value === null ? '' : getCalibrationNote(state.value, isTooFew.value),
)
const automaticPercent = computed(() =>
  state.value === null
    ? 0
    : ((state.value.automaticFloor - SLIDER_MIN) / (SLIDER_MAX - SLIDER_MIN)) * 100,
)
const isAutomaticInRange = computed(
  () =>
    state.value !== null &&
    state.value.automaticFloor >= SLIDER_MIN &&
    state.value.automaticFloor <= SLIDER_MAX,
)

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => {
  floor.load().catch(floor.reportFailure)
})

watch(
  () => [props.values.provider, props.values.model],
  () => {
    floor.load().catch(floor.reportFailure)
  },
)

/* -------------------------------- Handlers -------------------------------- */
function onRecalculate(): void {
  floor.calibrate().catch(floor.reportFailure)
}

function onRetry(): void {
  floor.load().catch(floor.reportFailure)
}

function onCheck(query: string): void {
  floor.probe(query).catch(floor.reportFailure)
}
</script>

<template>
  <section class="flex flex-col gap-3" :aria-label="WORDS.title" data-testid="floor-panel">
    <p
      v-if="status === 'loading'"
      class="text-small text-hushed"
      role="status"
      data-testid="floor-loading"
    >
      {{ WORDS.loading }}
    </p>

    <div v-else-if="status === 'error'" class="flex flex-col gap-2" data-testid="floor-load-error">
      <p class="text-small text-alarm" role="alert">{{ WORDS.loadFailed }} {{ loadError }}</p>
      <div>
        <Button variant="outline" @click="onRetry">{{ WORDS.retry }}</Button>
      </div>
    </div>

    <template v-else-if="state !== null">
      <div class="flex items-center justify-between gap-3">
        <span id="floor-mode-label" class="text-ink text-base">{{ WORDS.modeLabel }}</span>
        <SegmentedControl
          v-model="mode"
          :choices="choices"
          :is-disabled="isFileSet"
          aria-labelledby="floor-mode-label"
          data-testid="floor-mode"
        />
      </div>

      <div class="flex items-center justify-between gap-3">
        <span class="text-ink text-base">{{ WORDS.thresholdLabel }}</span>
        <span class="flex items-center gap-3">
          <span class="text-ink text-base tabular-nums" data-testid="floor-value">
            {{ shownFloor }}
          </span>
          <Button
            variant="outline"
            :disabled="isCalibrating"
            data-testid="floor-recalculate"
            @click="onRecalculate"
          >
            {{ isCalibrating ? WORDS.recalculating : WORDS.recalculate }}
          </Button>
        </span>
      </div>

      <div class="flex items-center justify-between gap-3">
        <span class="text-ink text-base">{{ WORDS.sourceLabel }}</span>
        <span class="text-hushed text-base" data-testid="floor-source">{{ sourceLabel }}</span>
      </div>
      <p v-if="isFileSet" class="text-small text-hushed" data-testid="floor-file-note">
        {{ WORDS.fileNote }}
      </p>

      <div class="flex flex-col gap-1">
        <div class="flex items-center justify-between gap-3">
          <span class="text-ink text-base">{{ WORDS.noiseLabel }}</span>
          <span
            v-if="state.calibration !== undefined"
            class="text-hushed text-small"
            data-testid="floor-noise"
          >
            {{ formatNoise(state.calibration) }}
          </span>
          <span v-else class="text-hushed text-small" data-testid="floor-noise">
            {{ WORDS.noiseNone }}
          </span>
        </div>
        <FloorHistogram
          v-if="state.calibration !== undefined"
          :calibration="state.calibration"
          :floor="state.floor"
        />
        <p class="text-small text-hushed" data-testid="floor-note">{{ note }}</p>
      </div>

      <div v-if="isManual && !isFileSet" class="flex flex-col gap-1" data-testid="floor-manual">
        <div class="relative">
          <Slider
            v-model="slider"
            :min="SLIDER_MIN"
            :max="SLIDER_MAX"
            :step="SLIDER_STEP"
            :aria-label="WORDS.sliderLabel"
          />
          <span
            v-if="isAutomaticInRange"
            class="bg-ink pointer-events-none absolute top-1/2 h-3 w-px -translate-y-1/2"
            :style="{ left: `${automaticPercent}%` }"
            :title="WORDS.automaticMarker"
            data-testid="floor-automatic-marker"
          />
        </div>
        <p class="text-small text-hushed">{{ WORDS.sliderHint }}</p>
      </div>

      <p
        v-if="actionError !== ''"
        class="text-small text-alarm"
        role="alert"
        data-testid="floor-action-error"
      >
        {{ WORDS.actionFailed }} {{ actionError }}
      </p>

      <div class="flex flex-col gap-1">
        <span class="text-ink text-base">{{ WORDS.probeLabel }}</span>
        <FloorProbe
          :result="probeResult"
          :error="probeError"
          :is-busy="isProbing"
          @check="onCheck"
        />
      </div>
    </template>
  </section>
</template>
