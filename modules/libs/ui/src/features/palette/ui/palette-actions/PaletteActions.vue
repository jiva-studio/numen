<script setup lang="ts">
/**
 * The action panel: everything the lit item offers, by name, in a list of its
 * own with a field of its own.
 *
 * The keyboard stands here for as long as the panel does; nothing it is given
 * reaches the palette underneath.
 *
 * `data-actions` names each part: `panel`, `list`, `name`, `hint`, `silence`
 * and `search`. An action is an option.
 */
import { computed, nextTick, ref, useId, useTemplateRef, watch } from 'vue'
import ActionRow from './ActionRow.vue'
import { findKeptAction, placeActions, type ActionWords } from '../../lib/actions'
import { getSteppedIndex, type PaletteAction } from '../../lib/item'
import { isActionsChord } from '../../lib/keys'

const props = defineProps<{
  /** What the lit item offers, in the order it offers them. */
  actions: readonly PaletteAction[]
  words: ActionWords
}>()

const open = defineModel<boolean>('open', { default: false })

const emit = defineEmits<{
  /** An action was chosen. The identifier is the caller's, handed back as given. */
  (event: 'choose', action: string): void
}>()

const uid = useId()
const actionName = (pos: number): string => `${uid}-action-${pos}`

const searchInput = useTemplateRef<HTMLInputElement>('searchInput')

/** What is typed here, which narrows the actions and nothing else. */
const searchQuery = ref('')

/** The action the panel is on, by its identity. */
const activeActionId = ref('')

const actions = computed(() => placeActions(props.actions, searchQuery.value))

/** Which row that action relies on, counted over the actions the words left. */
const here = computed(() =>
  actions.value.findIndex((one) => one.action.id === activeActionId.value),
)

const goTo = (to: number) => {
  activeActionId.value = actions.value[to]?.action.id ?? ''
}

/**
 * Where the pointer was. Only a pointer that has actually moved lights what it
 * is over: a list scrolling under a pointer standing still reports a move too.
 */
let pointerPosition = { x: -1, y: -1 }

const onRowPointerMove = (to: number, event: PointerEvent) => {
  if (event.clientX === pointerPosition.x && event.clientY === pointerPosition.y) return
  pointerPosition = { x: event.clientX, y: event.clientY }
  goTo(to)
}

/** What is lit is brought into sight. Only a key does this. */
const reveal = async () => {
  await nextTick()
  actionRows.get(activeActionId.value)?.element?.scrollIntoView?.({ block: 'nearest' })
}

/** A row standing now, which carries the element it is drawn as. */
interface RowHandle {
  readonly element: HTMLElement | null
}

/** The rows as they are drawn, each under the action it stands for. */
const actionRows = new Map<string, RowHandle>()

const hold = (action: string, row: unknown): void => {
  if (row) actionRows.set(action, row as RowHandle)
  else actionRows.delete(action)
}

const run = (to: number) => {
  const selectedAction = actions.value[to]
  if (!selectedAction) return
  emit('choose', selectedAction.action.id)
  open.value = false
}

/** A fresh list keeps the action the panel was on, wherever the words put it. */
watch(actions, (now) => {
  if (open.value) goTo(findKeptAction(now, activeActionId.value))
})

// It opens on the first action there is, with an empty field and the keyboard
// in it, and leaves nothing standing behind it.
watch(
  open,
  async (now) => {
    if (!now) {
      activeActionId.value = ''
      return
    }
    searchQuery.value = ''
    goTo(findKeptAction(actions.value, ''))
    await nextTick()
    searchInput.value?.focus()
  },
  { immediate: true },
)

const onKey = (event: KeyboardEvent) => {
  const step = (delta: number, from = here.value) => {
    event.preventDefault()
    goTo(getSteppedIndex(actions.value.length, from, delta))
    void reveal()
  }
  if (isActionsChord(event) || event.key === 'Escape') {
    event.preventDefault()
    open.value = false
  } else if (event.key === 'ArrowDown') step(1)
  else if (event.key === 'ArrowUp') step(-1)
  else if (event.key === 'Home') step(1, -1)
  else if (event.key === 'End') step(-1, 0)
  else if (event.key === 'Enter') {
    event.preventDefault()
    run(here.value)
  }
  // The keyboard stays in this field for as long as the panel stands.
  else if (event.key === 'Tab') event.preventDefault()
}
</script>

<template>
  <div
    class="actions panel-numen flex flex-col"
    data-actions="panel"
    role="dialog"
    aria-modal="true"
    :aria-label="words.name"
    @keydown.stop="onKey"
    @pointerdown.stop
  >
    <div
      :id="`${uid}-actions`"
      class="actions__list min-h-0 flex-1"
      data-actions="list"
      role="listbox"
      :aria-label="words.name"
    >
      <ActionRow
        v-for="row in actions"
        :id="actionName(row.at)"
        :ref="(element) => hold(row.action.id, element)"
        :key="row.action.id"
        :row="row"
        :is-current="row.at === here"
        @pointermove="onRowPointerMove(row.at, $event)"
        @pointerdown.prevent
        @click="run(row.at)"
      />
    </div>

    <p
      v-if="!actions.length"
      class="actions__silence text-hushed px-2 py-1.5"
      data-actions="silence"
    >
      {{ words.silence }}
    </p>

    <input
      ref="searchInput"
      v-model="searchQuery"
      class="actions__search w-full bg-transparent"
      data-actions="search"
      type="text"
      role="combobox"
      autocomplete="off"
      spellcheck="false"
      :placeholder="words.placeholder"
      :aria-label="words.placeholder"
      :aria-expanded="actions.length !== 0"
      :aria-controls="`${uid}-actions`"
      :aria-activedescendant="here >= 0 ? actionName(here) : undefined"
    />
  </div>
</template>

<style scoped>
.actions {
  /* How wide the panel is and how much of the palette its list takes. */
  --panel-width: 17.5rem;
  --panel-tallest: 15rem;

  position: absolute;
  inset-block-end: var(--numen-inset);
  inset-inline-end: var(--numen-inset);
  inline-size: var(--panel-width);
  max-inline-size: calc(100% - 2 * var(--numen-inset));
  max-block-size: calc(100% - 2 * var(--numen-inset));
}

.actions__list {
  max-block-size: var(--panel-tallest);
  padding: var(--numen-field-padding);
  overflow-y: auto;
  overscroll-behavior: contain;
}

.actions__silence {
  margin: 0;
}

.actions__search {
  block-size: var(--numen-field-min);
  padding-inline: var(--numen-field-text-inset);
  border: 0;
  border-block-start: var(--numen-stroke) solid var(--numen-panel-border);
  color: inherit;
  font: inherit;
  outline: none;
}

.actions__search::placeholder {
  color: var(--numen-edge-label);
}
</style>
