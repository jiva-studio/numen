<script setup lang="ts">
/**
 * The summary of the background tasks of the window as a chip: a ring filled by the progress of the
 * batch, and Idle, how many tasks are done of how many. A failed task tints the chip and changes no text. The list of the tasks
 * opens from it.
 */
import { computed, ref, useTemplateRef, watch } from 'vue'
import { isPageMove, useMenuDismissal } from '@numen/ui'
import type { BackgroundTask } from '@numen/plugin-sdk'
import { summarizeBatch, summarizeTasks } from '../lib/summary'
import { type ChipState } from '../model/chipDisplay'
import { useChipDisplay } from './useChipDisplay'
import { WORDS as words } from '../lib/words'
import TaskPopover from './TaskPopover.vue'
import './tasksStatus.css'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  tasks: readonly BackgroundTask[]
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'cancel', id: string): void
  (event: 'retry', id: string): void
  (event: 'dismiss', id: string): void
  (event: 'run-action', id: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const isOpen = ref(false)
const root = useTemplateRef<HTMLElement>('root')
const dismissal = useMenuDismissal(root, onClose, isPageMove)

const summary = computed(() => summarizeTasks(props.tasks))
// The tasks of one busy period: the ids are forgotten when the window is idle again.
const knownIds = ref<ReadonlySet<string>>(new Set())
const batch = computed(() => summarizeBatch(props.tasks, knownIds.value))
const hasFailed = computed(() => summary.value.failedCount > 0)
const liveChip = computed<ChipState>(() => ({
  ...batch.value,
  isActive: summary.value.activeCount > 0,
}))
const chip = useChipDisplay(() => liveChip.value)
const summaryText = computed(() => {
  if (!chip.value.isActive) return words.idle
  return `${chip.value.done} ${words.of} ${chip.value.total}`
})
const label = computed(() => {
  const failed = hasFailed.value ? `${summary.value.failedCount} ${words.failed}` : ''
  if (!chip.value.isActive) return failed || words.noneRunning
  const progressText = `${chip.value.done} ${words.of} ${chip.value.total} ${words.tasksDone}`
  return failed ? `${progressText}, ${failed}` : progressText
})

// The ring is a circle of RING_SIZE pixels; its arc is the share of the batch that is done.
const RING_SIZE = 12
const RING_STROKE = 2
const RING_RADIUS = (RING_SIZE - RING_STROKE) / 2
const RING_LENGTH = 2 * Math.PI * RING_RADIUS
const ringOffset = computed(() => RING_LENGTH * (1 - chip.value.fraction))

/* --------------------------------- Hooks ---------------------------------- */
watch(isOpen, (open) => (open ? dismissal.listen() : dismissal.release()))
watch(
  () => props.tasks,
  (tasks) => {
    const active = tasks.filter((task) => task.state === 'running' || task.state === 'cancelling')
    knownIds.value =
      active.length === 0
        ? new Set()
        : new Set([...knownIds.value, ...active.map((task) => task.id)])
  },
  { immediate: true },
)

/* -------------------------------- Handlers -------------------------------- */
function onToggle() {
  isOpen.value = !isOpen.value
}

function onClose() {
  isOpen.value = false
}
</script>

<template>
  <div
    ref="root"
    class="tasks-status"
    :class="{ 'tasks-status--active': chip.isActive }"
    data-tasks-status
  >
    <button
      type="button"
      tabindex="-1"
      class="tasks-status__trigger"
      :class="{ 'tasks-status__trigger--failed': hasFailed }"
      :aria-label="label"
      :title="label"
      :aria-expanded="isOpen"
      aria-haspopup="dialog"
      @click="onToggle"
    >
      <svg
        class="tasks-status__ring"
        :width="RING_SIZE"
        :height="RING_SIZE"
        :viewBox="`0 0 ${RING_SIZE} ${RING_SIZE}`"
        aria-hidden="true"
      >
        <circle
          class="tasks-status__ring-track"
          :cx="RING_SIZE / 2"
          :cy="RING_SIZE / 2"
          :r="RING_RADIUS"
          :stroke-width="RING_STROKE"
        />
        <circle
          v-if="chip.isActive && chip.fraction > 0"
          class="tasks-status__ring-fill"
          :cx="RING_SIZE / 2"
          :cy="RING_SIZE / 2"
          :r="RING_RADIUS"
          :stroke-width="RING_STROKE"
          :stroke-dasharray="RING_LENGTH"
          :stroke-dashoffset="ringOffset"
        />
      </svg>

      <span class="tasks-status__summary">{{ summaryText }}</span>
    </button>

    <div v-if="isOpen" class="tasks-status__popover-container">
      <TaskPopover
        :tasks="tasks"
        @close="onClose"
        @cancel="(id) => emit('cancel', id)"
        @retry="(id) => emit('retry', id)"
        @dismiss="(id) => emit('dismiss', id)"
        @run-action="(id) => emit('run-action', id)"
      />
    </div>
  </div>
</template>
