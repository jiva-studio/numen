<script setup lang="ts">
/**
 * The search relevance section of the plugin's settings: the threshold below which a match is
 * not shown, how it was chosen, and the noise measured on the vault.
 */
import { computed, onMounted } from 'vue'
import { Button, SegmentedControl, Slider } from '@numen/ui'
import type { JsonValue } from '@numen/plugin-sdk'
import { useFloor } from '../model/useFloor'
import { formatFloor, formatNoise, getCalibrationNote, getSourceLabel } from '../model/present'
import type { FloorClient } from '../model/types'
import FloorHistogram from './FloorHistogram.vue'
import FloorProbe from './FloorProbe.vue'
import { SLIDER_MAX, SLIDER_MIN, SLIDER_STEP, WORDS } from '../lib/words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  pluginId: string
  values: Readonly<Record<string, JsonValue>>
  client?: FloorClient
}>()

/* --------------------------------- State ---------------------------------- */
const floor = useFloor(props.client === undefined ? {} : { client: props.client })
const { state, isManual, manualValue, isTooFew } = floor.threshold
const { status, loadError, actionError, isCalibrating } = floor.progress
const { probeResult, probeError, isProbing } = floor.probing
const { load, calibrate, setMode, setManualFloor, probe, reportFailure } = floor.actions

const choices = [
  { id: 'auto', text: WORDS.modeAuto },
  { id: 'manual', text: WORDS.modeManual },
]
const mode = computed({
  get: () => (isManual.value ? 'manual' : 'auto'),
  set: (id: string) => {
    setMode(id === 'manual').catch(reportFailure)
  },
})
const slider = computed({
  get: () => manualValue.value,
  set: (value: number) => {
    setManualFloor(value).catch(reportFailure)
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
  load().catch(reportFailure)
})

/* -------------------------------- Handlers -------------------------------- */
function onRecalculate(): void {
  calibrate().catch(reportFailure)
}

function onRetry(): void {
  load().catch(reportFailure)
}

function onCheck(query: string): void {
  probe(query).catch(reportFailure)
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

      <div v-if="isManual" class="flex flex-col gap-1" data-testid="floor-manual">
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
