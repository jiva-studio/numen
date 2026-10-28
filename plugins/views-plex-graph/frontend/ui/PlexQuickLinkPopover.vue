<script setup lang="ts">
/**
 * Native Plex-style thought creation box and autocomplete dropdown.
 */
import { computed, nextTick, onMounted, ref, useId, useTemplateRef, watch } from 'vue'
import { useOutsidePress } from '@numen/ui'
import type { PlexRelation } from '../plex'
import { hasExactNoteMatch, useSearchResults } from '../lib/search'
import { WORDS as words } from '../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  at: { readonly x: number; readonly y: number }
  relation: PlexRelation
  search: (query: string) => Promise<readonly { path: string; title: string }[]>
}>()

const emit = defineEmits<{
  (event: 'select-note', path: string): void
  (event: 'create-note', title: string): void
  (event: 'dismiss'): void
}>()

/* --------------------------------- State ---------------------------------- */
const input = useTemplateRef<HTMLInputElement>('input')
const popover = useTemplateRef<HTMLDivElement>('popover')

const queryFieldId = useId()

const query = ref('')
const highlightedIndex = ref(0)
const { results, isLoading, setQuery } = useSearchResults((text) => props.search(text))

const trimmedQuery = computed(() => query.value.trim())

const hasExactMatch = computed(() => hasExactNoteMatch(results.value, trimmedQuery.value))

const showCreateOption = computed(() => {
  if (!trimmedQuery.value) return false
  return !hasExactMatch.value
})

const totalItems = computed(() => results.value.length + (showCreateOption.value ? 1 : 0))

const positionStyle = computed(() => ({
  left: `${Math.round(props.at.x)}px`,
  top: `${Math.round(props.at.y)}px`,
  '--relation-color': `var(--numen-relation-${props.relation}, var(--numen-accent))`,
}))

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => {
  void nextTick(() => {
    input.value?.focus()
  })
})

useOutsidePress(popover, () => emit('dismiss'))

watch(query, (text) => {
  highlightedIndex.value = 0
  void setQuery(text)
})

/* -------------------------------- Handlers -------------------------------- */
function onKeyDown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    event.stopPropagation()
    event.preventDefault()
    emit('dismiss')
    return
  }

  if (event.key === 'ArrowDown') {
    event.preventDefault()
    if (totalItems.value > 0) {
      highlightedIndex.value = (highlightedIndex.value + 1) % totalItems.value
    }
    return
  }

  if (event.key === 'ArrowUp') {
    event.preventDefault()
    if (totalItems.value > 0) {
      highlightedIndex.value = (highlightedIndex.value - 1 + totalItems.value) % totalItems.value
    }
    return
  }

  if (event.key === 'Enter') {
    event.preventDefault()
    submitHighlighted()
  }
}

function onSelectResult(path: string) {
  emit('select-note', path)
}

function onCreateAction() {
  if (trimmedQuery.value.length > 0) {
    emit('create-note', trimmedQuery.value)
  }
}

/* -------------------------------- Helpers --------------------------------- */
function submitHighlighted() {
  if (highlightedIndex.value < results.value.length) {
    const selectedItem = results.value[highlightedIndex.value]
    if (selectedItem) {
      onSelectResult(selectedItem.path)
    }
  } else if (showCreateOption.value) {
    onCreateAction()
  }
}
</script>

<template>
  <div
    ref="popover"
    class="plex-quick-link"
    :class="`plex-quick-link--${relation}`"
    :style="positionStyle"
    role="dialog"
    aria-label="New thought"
    :aria-busy="isLoading"
  >
    <!-- Plex Thought Node Box -->
    <div class="plex-quick-link__node">
      <label :for="queryFieldId" class="sr-only">{{ words.quickLinkLabel }}</label>
      <input
        :id="queryFieldId"
        ref="input"
        v-model="query"
        type="text"
        class="plex-quick-link__input"
        :placeholder="words.quickLinkPlaceholder"
        @keydown="onKeyDown"
      />
    </div>

    <!-- Dropdown Autocomplete Menu -->
    <div
      v-if="results.length > 0 || showCreateOption"
      class="plex-quick-link__dropdown"
      role="listbox"
    >
      <button
        v-for="(result, index) in results"
        :key="result.path"
        type="button"
        class="plex-quick-link__option"
        :class="{
          'plex-quick-link__option--active': highlightedIndex === index,
        }"
        role="option"
        :aria-selected="highlightedIndex === index"
        @click="onSelectResult(result.path)"
        @pointerenter="highlightedIndex = index"
      >
        <span class="plex-quick-link__option-title">{{ result.title || result.path }}</span>
      </button>

      <button
        v-if="showCreateOption"
        type="button"
        class="plex-quick-link__option plex-quick-link__option--create"
        :class="{ 'plex-quick-link__option--active': highlightedIndex === results.length }"
        role="option"
        :aria-selected="highlightedIndex === results.length"
        @click="onCreateAction"
        @pointerenter="highlightedIndex = results.length"
      >
        <span class="plex-quick-link__option-title">{{ words.createNote(trimmedQuery) }}</span>
      </button>
    </div>
  </div>
</template>

<style scoped>
.plex-quick-link {
  position: absolute;
  z-index: 20;
  inline-size: 0;
  block-size: 0;
  pointer-events: auto;
  font-family: var(--numen-font-sans);
  --plex-quick-link-width: 14rem;
  --plex-quick-link-height: 2rem;
}
.plex-quick-link__node,
.plex-quick-link__dropdown {
  background: var(--numen-surface-raised, var(--numen-surface));
  border: var(--numen-stroke, 1px) solid var(--relation-color, var(--numen-rule));
  border-radius: var(--radius, 0.375rem);
}
.plex-quick-link__node {
  position: absolute;
  margin-inline-start: calc(var(--plex-quick-link-width) / -2);
  margin-block-start: calc(var(--plex-quick-link-height) / -2);
  inline-size: var(--plex-quick-link-width);
  block-size: var(--plex-quick-link-height);
  padding-inline: var(--numen-node-padding, 0.625rem);
  display: flex;
  align-items: center;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.14);
}
.plex-quick-link__input,
.plex-quick-link__option {
  border: none;
  background: transparent;
  color: var(--numen-ink);
  font: inherit;
  font-size: var(--numen-font-size, 0.8125rem);
}
.plex-quick-link__input {
  inline-size: 100%;
  outline: none;
  padding: 0;
}
.plex-quick-link__input::placeholder {
  color: var(--numen-hushed);
  opacity: 0.85;
}
.plex-quick-link__dropdown {
  position: absolute;
  top: calc(var(--plex-quick-link-height) / 2 + 0.375rem);
  inset-inline-start: 50%;
  margin-inline-start: calc(var(--plex-quick-link-width) / -2);
  inline-size: var(--plex-quick-link-width);
  max-block-size: 12rem;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  padding: 0.25rem;
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.2);
  gap: 0.125rem;
}
.plex-quick-link__option {
  display: flex;
  align-items: center;
  padding: 0.3125rem 0.5rem;
  border-radius: 0.25rem;
  cursor: pointer;
}
.plex-quick-link__option--active {
  background: var(--numen-accent);
  color: var(--numen-accent-ink);
}
.plex-quick-link__option + .plex-quick-link__option--create {
  border-top: var(--numen-stroke, 1px) solid var(--numen-rule);
  margin-top: 0.125rem;
  padding-top: 0.375rem;
}
.plex-quick-link__option-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
