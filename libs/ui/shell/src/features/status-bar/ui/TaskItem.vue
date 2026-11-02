<script setup lang="ts">
/**
 * Single task row in the background tasks popover.
 *
 * The progress ring and the action button share one slot, stacked in one grid cell. The row draws
 * the same elements whether or not the pointer is over it; the pointer on the row, or focus inside
 * it, changes only their opacity.
 *
 * A running task is reported again many times a second. The row is one component for the life of
 * the task and updates its text and ring in place; whether the pointer is on it is a flag of the
 * row, set by the pointer entering and leaving, so no update decides it. The detail line keeps the
 * last source a running task named, so a report without one does not take a line out of the row
 * and move it from under the pointer.
 */
import { computed, ref, watch } from 'vue'
import { RotateCw, X } from '@lucide/vue'
import { Button } from '@numen/ui'
import type { StatusBarTask } from '../model/types'
import CircularProgress from './CircularProgress.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  task: StatusBarTask
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'cancel', id: string): void
  (event: 'retry', id: string): void
  (event: 'dismiss', id: string): void
  (event: 'action', id: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const isHovered = ref(false)
const lastAbout = ref(props.task.about ?? '')
watch(
  () => props.task.about,
  (about) => {
    if (about) lastAbout.value = about
  },
)
const subtext = computed(() => {
  if (props.task.state === 'failed') {
    return props.task.error || props.task.about || ''
  }
  return props.task.about || lastAbout.value
})
const isCancelShown = computed(
  () =>
    (props.task.state === 'running' || props.task.state === 'cancelling') &&
    props.task.canCancel !== false,
)
const isCancelling = computed(() => props.task.state === 'cancelling')
const hasAction = computed(
  () => isCancelShown.value || props.task.state === 'failed' || props.task.state === 'completed',
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

function onAction() {
  emit('action', props.task.id)
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
    <div class="task-item__content">
      <span class="task-item__label" :title="task.label">{{ task.label }}</span>
      <span
        v-if="subtext"
        class="task-item__detail"
        :class="{ 'text-alarm task-item__detail--wrapped': task.state === 'failed' }"
        :title="subtext"
      >
        {{ subtext }}
      </span>
      <div v-if="task.state === 'failed' && task.actionLabel" class="task-item__failure-row">
        <button type="button" class="task-item__action-link" @click.stop="onAction">
          {{ task.actionLabel }} &rarr;
        </button>
      </div>
    </div>

    <div class="task-item__slot" :class="{ 'task-item__slot--swaps': hasAction }">
      <CircularProgress
        class="task-item__indicator"
        :progress="task.progress"
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
        :title="isCancelling ? 'Cancelling' : 'Cancel task'"
        :aria-label="isCancelling ? 'Cancelling' : 'Cancel task'"
        @click.stop="onCancel"
      >
        <X :size="12" aria-hidden="true" />
      </Button>

      <Button
        v-else-if="task.state === 'failed'"
        variant="ghost"
        size="icon-small"
        class="task-item__btn"
        title="Retry task"
        aria-label="Retry task"
        @click.stop="onRetry"
      >
        <RotateCw :size="12" aria-hidden="true" />
      </Button>

      <Button
        v-else-if="task.state === 'completed'"
        variant="ghost"
        size="icon-small"
        class="task-item__btn"
        title="Dismiss"
        aria-label="Dismiss"
        @click.stop="onDismiss"
      >
        <X :size="12" aria-hidden="true" />
      </Button>
    </div>
  </div>
</template>
