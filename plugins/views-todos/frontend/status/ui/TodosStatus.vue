<script setup lang="ts">
/**
 * The chip of the title bar and the popover with the lists Overdue, Today and No date.
 */
import { computed, ref, useTemplateRef, watch } from 'vue'
import { ListChecks } from '@lucide/vue'
import { isPageMove, useMenuDismissal } from '@numen/ui'
import { getChip } from '../model/chip'
import { getNoteTitle, useTaskGroups } from '../model/useTaskGroups'
import { WORDS as words } from '../../words'
import type { Summary, TaskReader } from '../../types'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  summary: Summary | null
  api: TaskReader
  getToday: () => string
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'open', target: { path: string; line: number }): void
}>()

/* --------------------------------- State ---------------------------------- */
const isOpen = ref(false)
const root = useTemplateRef<HTMLElement>('root')
const dismissal = useMenuDismissal(root, onClose, isPageMove)
const chip = computed(() => getChip(props.summary))
const label = computed(() => `${words.chipLabel}: ${chip.value.count}`)
const { groups, load } = useTaskGroups(props.api, props.getToday)

/* --------------------------------- Hooks ---------------------------------- */
watch(isOpen, (open) => {
  if (!open) return dismissal.release()
  dismissal.listen()
  return load()
})
watch(
  () => props.summary,
  () => {
    if (isOpen.value) void load()
  },
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
  <div v-if="chip.isVisible" ref="root" class="todos-status">
    <button
      type="button"
      tabindex="-1"
      class="todos-status__trigger"
      :class="{ 'todos-status__trigger--overdue': chip.isOverdue }"
      :aria-label="label"
      :title="label"
      :aria-expanded="isOpen"
      aria-haspopup="dialog"
      @click="onToggle"
    >
      <ListChecks :size="12" aria-hidden="true" />
      <span>{{ chip.count }}</span>
    </button>

    <div v-if="isOpen" class="todos-status__popover" role="dialog" :aria-label="words.chipLabel">
      <section v-for="group in groups" :key="group.key" class="todos-status__group">
        <h3 class="todos-status__heading">
          {{ group.label }} <span class="todos-status__total">{{ group.total }}</span>
        </h3>
        <ul class="todos-status__list">
          <li v-for="task in group.tasks" :key="`${task.path}:${task.line}`" class="todos-status__row">
            <input type="checkbox" disabled :aria-label="words.taskDone" />
            <button
              type="button"
              class="todos-status__text"
              @click="emit('open', { path: task.path, line: task.line })"
            >
              {{ task.text }}
            </button>
            <span v-if="task.due" class="todos-status__due">{{ task.due }}</span>
            <span class="todos-status__note">{{ getNoteTitle(task.path) }}</span>
          </li>
        </ul>
      </section>
    </div>
  </div>
</template>

<style scoped>
.todos-status {
  --popover-offset: 0.25rem;
  position: relative;
  display: inline-flex;
  align-items: center;
  height: 100%;
  padding-inline: calc(var(--numen-inset) / 4);
  color: var(--numen-ink);
  user-select: none;
}

.todos-status__trigger {
  --chip-height: 1.25rem;
  --chip-text: 0.75rem;
  --chip-pad: 0.5rem;
  display: inline-flex;
  align-items: center;
  gap: calc(var(--chip-pad) / 2);
  box-sizing: border-box;
  height: var(--chip-height);
  padding: 0 var(--chip-pad);
  background: var(--numen-surface);
  border: none;
  border-radius: var(--numen-radius);
  outline: none;
  color: var(--numen-ink);
  font: inherit;
  font-size: var(--chip-text);
  font-variant-numeric: tabular-nums;
  line-height: 1;
  cursor: pointer;
}

.todos-status__trigger:hover,
.todos-status__trigger[aria-expanded='true'] {
  background: color-mix(in oklab, var(--numen-surface), var(--numen-ink) 10%);
}

.todos-status__trigger--overdue,
.todos-status__trigger--overdue:hover,
.todos-status__trigger--overdue[aria-expanded='true'] {
  background: var(--numen-alarm-bg);
  color: var(--numen-alarm);
}

.todos-status__popover {
  --popover-width: 24rem;
  --popover-max-height: 22rem;
  --popover-pad: 0.5rem;
  position: absolute;
  top: calc(100% + var(--popover-offset));
  right: 0;
  z-index: 50;
  width: var(--popover-width);
  max-width: calc(100vw - 1rem);
  max-height: var(--popover-max-height);
  padding: var(--popover-pad);
  overflow-y: auto;
  background: var(--numen-raised);
  color: var(--numen-ink);
  border: 1px solid var(--numen-panel-border);
  border-radius: var(--numen-radius-popup);
  box-shadow: var(--numen-panel-shadow);
}

.todos-status__heading {
  margin: 0;
  padding: 0.25rem 0.5rem;
  font-size: var(--numen-text-1);
  font-weight: 600;
  color: var(--numen-hushed);
}

.todos-status__list {
  margin: 0;
  padding: 0;
  list-style: none;
}

.todos-status__row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.25rem 0.5rem;
  font-size: var(--numen-text-2);
}

.todos-status__text {
  flex: 1;
  min-width: 0;
  padding: 0;
  overflow: hidden;
  background: none;
  border: none;
  color: inherit;
  font: inherit;
  text-align: left;
  text-overflow: ellipsis;
  white-space: nowrap;
  cursor: pointer;
}

.todos-status__due,
.todos-status__note {
  flex: none;
  font-size: var(--numen-text-1);
  color: var(--numen-hushed);
}
</style>
