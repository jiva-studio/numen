<script setup lang="ts">
/**
 * The chip of the title bar and the popover with the lists Overdue, Today, Next 7 days and No date.
 */
import { computed, ref, useTemplateRef, watch } from 'vue'
import { ListChecks } from '@lucide/vue'
import { isPageMove, Tooltip, useMenuDismissal, useTooltip } from '@numen/ui'
import { getChip, type ShownGroups } from '../model/chip'
import { getNoteTitle, useTaskGroups } from '../model/useTaskGroups'
import { useTaskTicks } from '../model/useTaskTicks'
import { taskDoneLabel, WORDS as words } from '../../words'
import type { Summary, Task, TaskReader, TaskWriter } from '../../types'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  summary: Summary | null
  api: TaskReader & TaskWriter
  getToday: () => string
  shown: ShownGroups
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'open', target: { path: string; line: number; raw: string }): void
  (event: 'showError', message: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const isOpen = ref(false)
const root = useTemplateRef<HTMLElement>('root')
const dismissal = useMenuDismissal(root, onClose, isPageMove)
const chip = computed(() => getChip(props.summary, props.shown))
const label = computed(() => `${words.chipLabel}: ${chip.value.count}`)
const { groups, load } = useTaskGroups(props.api, props.getToday, () => props.shown)
const ticks = useTaskTicks(props.api, props.getToday, load, (message) => emit('showError', message))
const { isRecurring, isTickable } = ticks
const tip = useTooltip()

/* --------------------------------- Hooks ---------------------------------- */
watch(isOpen, (open) => {
  if (!open) {
    dismissal.release()
    return
  }
  dismissal.listen()
  load().catch(() => undefined) // the popover keeps the lists it has; opening it again reads again
})
watch(
  () => [props.summary, props.shown],
  () => {
    if (isOpen.value) load().catch(() => undefined) // the popover keeps the lists it has until the next summary
  },
)

/* -------------------------------- Handlers -------------------------------- */
function onToggle() {
  isOpen.value = !isOpen.value
}

function onClose() {
  isOpen.value = false
}

function onTick(task: Task) {
  if (!isTickable(task)) return
  ticks.tick(task).catch(() => emit('showError', words.tickFailed))
}
</script>

<template>
  <div v-if="chip.isVisible" ref="root" class="todos-status">
    <button
      type="button"
      tabindex="-1"
      class="todos-status__trigger"
      :aria-label="label"
      :aria-expanded="isOpen"
      aria-haspopup="dialog"
      @mouseenter="tip.open($event, label)"
      @mouseleave="tip.close"
      @focus="tip.open($event, label)"
      @blur="tip.close"
      @click="onToggle"
    >
      <ListChecks :size="12" aria-hidden="true" />
      <span>{{ chip.count }}</span>
    </button>

    <div v-if="isOpen" class="todos-status__popover" role="dialog" :aria-label="words.chipLabel">
      <section v-for="group in groups" :key="group.key" class="todos-status__group">
        <h3 class="todos-status__heading">
          {{ group.label }}
          <span class="todos-status__total">{{ group.total }}</span>
        </h3>
        <ul class="todos-status__list">
          <li
            v-for="task in group.tasks"
            :key="`${task.path}:${task.line}`"
            class="todos-status__row"
          >
            <span
              class="todos-status__tick"
              @mouseenter="tip.open($event, isRecurring(task) ? words.recurringNotTickable : '')"
              @mouseleave="tip.close"
              @focusin="tip.open($event, isRecurring(task) ? words.recurringNotTickable : '')"
              @focusout="tip.close"
            >
              <input
                type="checkbox"
                :aria-disabled="!isTickable(task)"
                :aria-label="taskDoneLabel(task.text)"
                @click.prevent="onTick(task)"
              />
            </span>
            <button
              type="button"
              class="todos-status__text"
              @click="emit('open', { path: task.path, line: task.line, raw: task.raw })"
            >
              {{ task.text }}
            </button>
            <span v-if="task.due" class="todos-status__due">{{ task.due }}</span>
            <span class="todos-status__note">{{ getNoteTitle(task.path) }}</span>
          </li>
        </ul>
      </section>
    </div>
    <Teleport to="body">
      <Tooltip v-if="tip.at.value && tip.text.value" :id="tip.id" :at="tip.at.value">
        {{ tip.text.value }}
      </Tooltip>
    </Teleport>
  </div>
</template>

<style scoped>
.todos-status {
  --popover-offset: calc(0.25 * var(--numen-chrome-unit));
  position: relative;
  display: inline-flex;
  align-items: center;
  height: 100%;
  padding-inline: calc(var(--numen-chrome-inset) / 4);
  color: var(--numen-ink);
  user-select: none;
}

.todos-status__trigger {
  --chip-height: calc(1.25 * var(--numen-chrome-unit));
  --chip-text: var(--numen-chrome-text-2);
  --chip-pad: var(--numen-chrome-inset);
  display: inline-flex;
  align-items: center;
  gap: calc(var(--chip-pad) / 2);
  box-sizing: border-box;
  height: var(--chip-height);
  padding: 0 var(--chip-pad);
  background: var(--numen-surface);
  border: none;
  border-radius: var(--numen-chrome-radius);
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

.todos-status__popover .todos-status__heading {
  margin: 0;
  padding: 0.25rem 0.5rem;
  font-size: var(--numen-text-1);
  font-weight: 600;
  color: var(--numen-hushed);
}

.todos-status__popover .todos-status__list {
  margin: 0;
  padding: 0;
  list-style: none;
}

.todos-status__popover .todos-status__row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.25rem 0.5rem;
  font-size: var(--numen-text-2);
  border-radius: var(--numen-radius);
}

.todos-status__popover .todos-status__row:hover {
  background: var(--numen-selection-away);
}

.todos-status__popover .todos-status__tick {
  display: inline-flex;
  flex: none;
}

.todos-status__popover .todos-status__row input[type='checkbox'][aria-disabled='true'] {
  cursor: not-allowed;
  opacity: 0.5;
}

.todos-status__popover .todos-status__row input[type='checkbox'] {
  flex: none;
  inline-size: 1em;
  block-size: 1em;
  margin: 0;
  accent-color: var(--numen-accent);
}

.todos-status__popover .todos-status__text {
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

.todos-status__popover .todos-status__due,
.todos-status__popover .todos-status__note {
  flex: none;
  color: var(--numen-hushed);
}
</style>
