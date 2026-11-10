<script setup lang="ts">
/**
 * One row of the list of background tasks.
 *
 * The progress ring and the action button share one slot, stacked in one grid cell. The row draws
 * the same elements whether or not the pointer is over it; the pointer on the row, or focus inside
 * it, changes only their opacity.
 *
 * A running task is reported again many times a second. The row is one component for the life of
 * the task and updates its text and ring in place; whether the pointer is on it is a flag of the
 * row, set by the pointer entering and leaving, so no update decides it. The detail line keeps the
 * last source a running task named, so a report without one neither removes a line from the row
 * nor moves the row from under the pointer.
 *
 * A notice has an information icon on the left, all of its text on as many lines as it needs, and
 * a dismiss button with a cross that is always visible. It has no ring.
 */
import { computed, ref, watch } from 'vue'
import { Info, RotateCw, X } from '@lucide/vue'
import { Button } from '@numen/ui'
import type { BackgroundTask } from '@numen/plugin-sdk'
import { getProgress } from '../lib/summary'
import { WORDS as words } from '../lib/words'
import CircularProgress from './CircularProgress.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  task: BackgroundTask
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'cancel', id: string): void
  (event: 'retry', id: string): void
  (event: 'dismiss', id: string): void
  (event: 'open-settings', pluginId: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const isHovered = ref(false)
const lastAbout = ref(props.task.about)
const isNotice = computed(() => props.task.state === 'notice')
const isFailed = computed(() => props.task.state === 'failed')
const isCancelling = computed(() => props.task.state === 'cancelling')
const progress = computed(() => getProgress(props.task))
const subtext = computed(() => {
  if (isFailed.value) return props.task.error || props.task.about
  return props.task.about || lastAbout.value
})
const isCancelShown = computed(
  () => (props.task.state === 'running' || isCancelling.value) && props.task.allowed.cancel,
)
const isRetryShown = computed(() => isFailed.value && props.task.allowed.retry)
const isDismissShown = computed(
  () => isNotice.value || (isFailed.value && !props.task.allowed.retry),
)
const hasAction = computed(() => isCancelShown.value || isRetryShown.value || isDismissShown.value)

/* --------------------------------- Hooks ---------------------------------- */
watch(
  () => props.task.about,
  (about) => {
    if (about) lastAbout.value = about
  },
)

/* -------------------------------- Handlers -------------------------------- */
function onPointerEnter() {
  isHovered.value = true
}

function onPointerLeave() {
  isHovered.value = false
}

function onCancel() {
  emit('cancel', props.task.id)
}

function onRetry() {
  emit('retry', props.task.id)
}

function onDismiss() {
  emit('dismiss', props.task.id)
}

function onOpenSettings() {
  emit('open-settings', props.task.action.pluginId)
}
</script>

<template>
  <div
    class="task-item"
    :class="[`task-item--${task.state}`, { 'task-item--hovered': isHovered }]"
    :data-task-id="task.id"
    @pointerenter="onPointerEnter"
    @pointerleave="onPointerLeave"
  >
    <Info v-if="isNotice" class="task-item__notice-icon" :size="14" aria-hidden="true" />

    <div class="task-item__content">
      <span class="task-item__label" :title="task.label">{{ task.label }}</span>
      <span
        v-if="subtext"
        class="task-item__detail"
        :class="{
          'text-alarm': isFailed,
          'task-item__detail--wrapped': isFailed || isNotice,
        }"
        :title="subtext"
      >
        {{ subtext }}
      </span>
      <div v-if="isFailed && task.action.label" class="task-item__failure-row">
        <button type="button" class="task-item__action-link" @click.stop="onOpenSettings">
          {{ task.action.label }} &rarr;
        </button>
      </div>
    </div>

    <div class="task-item__slot" :class="{ 'task-item__slot--swaps': hasAction && !isNotice }">
      <CircularProgress
        v-if="!isNotice"
        class="task-item__indicator"
        :progress="progress"
        :state="task.state"
        :size="20"
        :stroke-width="2.2"
      />

      <Button
        v-if="isCancelShown"
        variant="ghost"
        size="icon-small"
        class="task-item__btn"
        :disabled="isCancelling"
        :title="isCancelling ? words.cancelling : words.cancel"
        :aria-label="isCancelling ? words.cancelling : words.cancel"
        @click.stop="onCancel"
      >
        <X :size="12" aria-hidden="true" />
      </Button>

      <Button
        v-else-if="isRetryShown"
        variant="ghost"
        size="icon-small"
        class="task-item__btn"
        :title="words.retry"
        :aria-label="words.retry"
        @click.stop="onRetry"
      >
        <RotateCw :size="12" aria-hidden="true" />
      </Button>

      <Button
        v-else-if="isDismissShown"
        variant="ghost"
        size="icon-small"
        class="task-item__btn"
        :class="{ 'task-item__btn--visible': isNotice }"
        :title="words.dismiss"
        :aria-label="words.dismiss"
        @click.stop="onDismiss"
      >
        <X :size="12" aria-hidden="true" />
      </Button>
    </div>
  </div>
</template>
