<script setup lang="ts">
/* --------------------------------- Props ---------------------------------- */
import { computed, ref, useTemplateRef } from 'vue'
import { SearchableListPopup, Spinner, type ListItem, measureRect } from '@numen/ui'
import type { Position } from '@numen/ui'
import type { AgentModelOption } from '../../model/options'
import { browserTimer, useDelayedOpen } from '../../model/opening'
import type { AgentModelSelectorProps } from './AgentModelSelector.types'

const props = withDefaults(defineProps<AgentModelSelectorProps>(), {
  options: () => [],
  selectedAgentId: '',
  selectedModelId: '',
  isDisabled: false,
  emptyLabel: 'No agent installed',
  pendingAgentIds: () => [],
  loadingLabel: 'Loading…',
  failedAgentIds: () => [],
  failedLabel: 'Could not list models',
  timer: () => browserTimer,
})

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'select', option: AgentModelOption): void
  /** The person asked for the list of models; it opens once the models of the agents are listed. */
  (event: 'open'): void
}>()

/* --------------------------------- State ---------------------------------- */
const trigger = useTemplateRef<HTMLButtonElement>('trigger')
const menuPosition = ref<Position>({ x: 0, y: 0 })

const isReady = computed(() => props.pendingAgentIds.length === 0)
const { isOpen, isWaiting, isSpinnerShown, request, close } = useDelayedOpen(
  isReady,
  props.timer,
  () => {
    const rect = measureRect(trigger.value)
    if (rect) menuPosition.value = { x: rect.left, y: rect.bottom + 4 }
  },
)

const currentKey = computed(() => `${props.selectedAgentId}:${props.selectedModelId}`)

const currentOption = computed(() => {
  const match = props.options.find(
    (one) => one.agentId === props.selectedAgentId && one.modelId === props.selectedModelId,
  )
  if (match) return match
  return props.options.find((one) => one.isDefault) ?? props.options[0]
})

const label = computed(() => {
  const active = currentOption.value
  if (!active) return props.emptyLabel
  return `${active.agentTitle} · ${active.modelTitle}`
})

/**
 * An agent still being asked, or one that could not list its models, stands as one disabled row under its
 * name, in its place.
 */
const menuItems = computed<ListItem[]>(() => {
  const placed = new Set<string>()
  return props.options.flatMap((one) => {
    const isFailed = props.failedAgentIds.includes(one.agentId)
    if (!isFailed && !props.pendingAgentIds.includes(one.agentId)) {
      return [{ id: `${one.agentId}:${one.modelId}`, text: one.modelTitle, group: one.agentTitle }]
    }
    if (placed.has(one.agentId)) return []
    placed.add(one.agentId)
    const id = `${one.agentId}:${isFailed ? 'failed' : 'loading'}`
    const text = isFailed ? props.failedLabel : props.loadingLabel
    return [{ id, text, group: one.agentTitle, isDisabled: true }]
  })
})

/* -------------------------------- Handlers -------------------------------- */
function onToggle() {
  if (props.isDisabled || props.options.length === 0) return
  if (isOpen.value || isWaiting.value) {
    close()
    return
  }
  emit('open')
  request()
}

function onChoose(id: string) {
  close()
  const selectedOption = props.options.find((one) => `${one.agentId}:${one.modelId}` === id)
  if (selectedOption) emit('select', selectedOption)
}

function onDismiss() {
  close()
}
</script>

<template>
  <div class="model-selector relative inline-flex max-w-full min-w-0 items-center">
    <button
      ref="trigger"
      type="button"
      class="model-selector__pill bg-bubble text-ink text-small hover:bg-surface rounded-pill inline-flex h-7 max-w-full min-w-0 cursor-pointer items-center gap-1.5 px-2.5 font-medium transition-colors disabled:pointer-events-none disabled:opacity-50"
      :disabled="isDisabled || options.length === 0"
      :aria-busy="isWaiting"
      @click="onToggle"
    >
      <Spinner v-if="isSpinnerShown" class="model-selector__spinner" />
      <span class="model-selector__label min-w-0 flex-1 truncate">{{ label }}</span>
      <svg
        class="model-selector__chevron h-3.5 w-3.5 shrink-0 opacity-60"
        viewBox="0 0 16 16"
        fill="currentColor"
      >
        <path
          fill-rule="evenodd"
          d="M4.22 6.22a.75.75 0 0 1 1.06 0L8 8.94l2.72-2.72a.75.75 0 1 1 1.06 1.06l-3.25 3.25a.75.75 0 0 1-1.06 0L4.22 7.28a.75.75 0 0 1 0-1.06Z"
          clip-rule="evenodd"
        />
      </svg>
    </button>

    <SearchableListPopup
      v-if="isOpen"
      :items="menuItems"
      :at="menuPosition"
      :model-value="currentKey"
      :from="trigger"
      label="Model"
      placeholder="Search models"
      empty-text="No models found"
      :max-rows="12"
      @select="onChoose"
      @close="onDismiss"
    />
  </div>
</template>
