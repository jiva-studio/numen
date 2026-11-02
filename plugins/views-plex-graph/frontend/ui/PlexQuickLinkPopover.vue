<script setup lang="ts">
/**
 * Native Plex-style thought creation combobox: the input and its options in one frame.
 */
import { computed, nextTick, onMounted, ref, useId, useTemplateRef, watch } from 'vue'
import { useOutsidePress } from '@numen/ui'
import type { PlexRelation } from '../plex'
import { findExactNote } from '../lib/search'
import { useSearchResults } from '../model/useSearchResults'
import { WORDS as words } from '../words'

type Row =
  | { readonly key: string; readonly kind: 'create' }
  | {
      readonly key: string
      readonly kind: 'note'
      readonly path: string
      readonly title: string
    }

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
const FIRST_KEY = ''

const input = useTemplateRef<HTMLInputElement>('input')
const popover = useTemplateRef<HTMLDivElement>('popover')

const queryFieldId = useId()
const listboxId = useId()

const query = ref('')
const activeKey = ref(FIRST_KEY)
const { results, isLoading, setQuery } = useSearchResults((text) => props.search(text))

const trimmedQuery = computed(() => query.value.trim())

const exactNote = computed(() => findExactNote(results.value, trimmedQuery.value))

const toRow = (note: { path: string; title: string }, key: string): Row => ({
  key,
  kind: 'note',
  path: note.path,
  title: note.title || note.path,
})

const rows = computed<readonly Row[]>(() => {
  if (!trimmedQuery.value) return []
  const exact = exactNote.value
  return [
    exact ? toRow(exact, FIRST_KEY) : { key: FIRST_KEY, kind: 'create' },
    ...results.value.filter((note) => note !== exact).map((note) => toRow(note, note.path)),
  ]
})

const activeIndex = computed(() => {
  const index = rows.value.findIndex((row) => row.key === activeKey.value)
  return index < 0 ? 0 : index
})

const isOpen = computed(() => rows.value.length > 0)

const positionStyle = computed(() => ({
  left: `${Math.round(props.at.x)}px`,
  top: `${Math.round(props.at.y)}px`,
  '--relation-color': `var(--numen-relation-${props.relation}, var(--numen-accent))`,
}))

/* --------------------------------- Hooks ---------------------------------- */
onMounted(async () => {
  await nextTick(() => {
    input.value?.focus()
  })
})

useOutsidePress(popover, () => emit('dismiss'))

watch(query, (text) => {
  activeKey.value = FIRST_KEY
  return setQuery(text)
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
    moveActive(1)
    return
  }

  if (event.key === 'ArrowUp') {
    event.preventDefault()
    moveActive(-1)
    return
  }

  if (event.key === 'Enter') {
    event.preventDefault()
    const row = rows.value[activeIndex.value]
    if (row) choose(row)
  }
}

function choose(row: Row) {
  const exact = exactNote.value
  if (row.kind === 'create' && exact) {
    emit('select-note', exact.path)
  } else if (row.kind === 'create') {
    emit('create-note', trimmedQuery.value)
  } else {
    emit('select-note', row.path)
  }
}

/* -------------------------------- Helpers --------------------------------- */
function moveActive(step: number) {
  const count = rows.value.length
  if (count === 0) return
  const next = rows.value[(activeIndex.value + step + count) % count]
  if (next) activeKey.value = next.key
}

function getOptionId(index: number): string {
  return `${listboxId}-${index}`
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
    <div class="plex-quick-link__frame">
      <div class="plex-quick-link__node">
        <label :for="queryFieldId" class="sr-only">{{ words.quickLinkLabel }}</label>
        <input
          :id="queryFieldId"
          ref="input"
          v-model="query"
          type="text"
          role="combobox"
          class="plex-quick-link__input"
          :placeholder="words.quickLinkPlaceholder"
          aria-autocomplete="list"
          :aria-expanded="isOpen"
          :aria-controls="listboxId"
          :aria-activedescendant="isOpen ? getOptionId(activeIndex) : undefined"
          @keydown="onKeyDown"
        />
      </div>

      <div class="plex-quick-link__panel" :data-open="isOpen">
        <div class="plex-quick-link__panel-inner">
          <div v-if="isOpen" :id="listboxId" class="plex-quick-link__dropdown" role="listbox">
            <button
              v-for="(row, index) in rows"
              :id="getOptionId(index)"
              :key="row.key"
              type="button"
              tabindex="-1"
              class="plex-quick-link__option"
              :class="{
                'plex-quick-link__option--create': row.kind === 'create',
                'plex-quick-link__option--active': activeIndex === index,
              }"
              role="option"
              :aria-selected="activeIndex === index"
              @click="choose(row)"
              @pointerenter="activeKey = row.key"
            >
              <span class="plex-quick-link__option-title">
                {{ row.kind === 'create' ? words.createNote(trimmedQuery) : row.title }}
              </span>
            </button>
          </div>
        </div>
      </div>
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
.plex-quick-link__frame {
  position: absolute;
  margin-inline-start: calc(var(--plex-quick-link-width) / -2);
  margin-block-start: calc(var(--plex-quick-link-height) / -2);
  inline-size: var(--plex-quick-link-width);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--numen-raised);
  border: var(--numen-stroke) solid var(--relation-color);
  border-radius: var(--numen-radius-panel);
  box-shadow: var(--numen-panel-shadow);
}
.plex-quick-link__node {
  flex: none;
  block-size: var(--plex-quick-link-height);
  padding-inline: var(--numen-node-padding);
  display: flex;
  align-items: center;
}
.plex-quick-link__input,
.plex-quick-link__option {
  border: none;
  background: transparent;
  color: var(--numen-ink);
  font: inherit;
  font-size: var(--numen-font-size);
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
.plex-quick-link__panel {
  display: grid;
  grid-template-rows: 0fr;
  transition: grid-template-rows var(--numen-motion) ease;
}
@media (prefers-reduced-motion: reduce) {
  .plex-quick-link__panel {
    transition: none;
  }
}
.plex-quick-link__panel[data-open='true'] {
  grid-template-rows: 1fr;
}
.plex-quick-link__panel-inner {
  min-block-size: 0;
  overflow: hidden;
}
.plex-quick-link__dropdown {
  max-block-size: 12rem;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  padding: 0.25rem;
  gap: 0.125rem;
  border-top: var(--numen-stroke) solid var(--numen-rule);
}
.plex-quick-link__option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  padding: 0.3125rem 0.5rem;
  border-radius: var(--numen-radius-tight);
  cursor: pointer;
  text-align: start;
}
.plex-quick-link__option--active {
  background: var(--numen-accent);
  color: var(--numen-accent-ink);
}
.plex-quick-link__option-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.plex-quick-link__option-hint {
  flex: none;
  font-size: var(--numen-text-1);
  opacity: 0.75;
}
</style>
