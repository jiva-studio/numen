<script setup lang="ts">
import { computed } from 'vue'
import {
  AudioLines,
  Book,
  BookOpen,
  Bot,
  FilePlus,
  FileText,
  Folder,
  FolderOpen,
  FolderTree,
  Layers,
  ScrollText,
  Search,
  Settings,
  Waypoints,
} from '@lucide/vue'
import { Prose, Tree } from '@numen/ui'
import type { Frame, TabIcon } from './frame'
import { DEFAULT_NOTE, getNoteTitle, NOTES, OPEN, ROWS, VAULT_NAME } from './vault'

const props = defineProps<{ frame: Frame }>()

const TAB_ICONS: Record<TabIcon, typeof FileText> = {
  note: FileText,
  book: BookOpen,
  plex: Waypoints,
  agent: Bot,
  audio: AudioLines,
  logs: ScrollText,
  search: Search,
  settings: Settings,
  deck: Layers,
}

const hasFiles = computed(() => props.frame.files !== false)
const isTab = computed(() => props.frame.place === 'tab')
const hasNote = computed(() => !isTab.value || props.frame.note !== undefined)
const notePath = computed(() => props.frame.notePath ?? DEFAULT_NOTE)
const noteTitle = computed(() => getNoteTitle(notePath.value))
const noteText = computed(() => NOTES[notePath.value])
const selected = computed(
  () => props.frame.selected ?? (hasNote.value ? notePath.value : undefined),
)
const open = computed(() => {
  const folder = selected.value?.split('/').slice(0, -1).join('/')
  return folder && !OPEN.includes(folder) ? [...OPEN, folder] : OPEN
})
const storyTabs = computed(() => [
  props.frame.tab ?? { title: 'Untitled', icon: 'note' },
  ...(props.frame.otherTabs ?? []),
])
const share = computed(() => props.frame.share ?? 0.6)
const title = computed(() => `numen — ${props.frame.title ?? VAULT_NAME}`)

const getRowIcon = (id: string, isOpen: boolean) => {
  if (ROWS.some((row) => row.id === id && row.hasChildren)) return isOpen ? FolderOpen : Folder
  if (/\.(pdf|epub)$/.test(id)) return Book
  if (id.endsWith('.mp3')) return AudioLines
  return FileText
}
</script>

<template>
  <div class="window-frame numen">
    <slot v-if="frame.place === 'title-bar'" />
    <header v-else class="window-frame__title-bar">
      <span class="window-frame__controls" aria-hidden="true">
        <i class="window-frame__close" />
        <i class="window-frame__minimize" />
        <i class="window-frame__maximize" />
      </span>
      <span class="window-frame__commands">
        <FolderTree :size="15" />
        <Waypoints :size="15" />
        <FilePlus :size="15" />
      </span>
      <span class="window-frame__title">{{ title }}</span>
      <span class="window-frame__status">
        <slot v-if="frame.place === 'title-status'" />
      </span>
    </header>

    <div class="window-frame__body">
      <section v-if="hasFiles" class="window-frame__pane window-frame__pane--files">
        <div class="window-frame__strip">
          <span class="window-frame__tab" data-showing>
            <FolderTree class="window-frame__tab-icon" />
            Files
            <b>×</b>
          </span>
        </div>
        <div class="window-frame__content">
          <div v-if="frame.place === 'files'" class="window-frame__fill"><slot /></div>
          <div v-else class="window-frame__fill">
            <Tree
              :rows="ROWS"
              :open="open"
              :selected="selected ? [selected] : []"
              name="Vault files"
            >
              <template #icon="{ id, isOpen }">
                <component :is="getRowIcon(id, isOpen)" class="window-frame__row-icon" />
              </template>
            </Tree>
          </div>
        </div>
      </section>

      <section
        v-if="!isTab || frame.note === 'before'"
        class="window-frame__pane"
        :style="{ flexGrow: isTab ? 1 - share : 1 }"
      >
        <div class="window-frame__strip">
          <span class="window-frame__tab" data-showing>
            <FileText class="window-frame__tab-icon" />
            {{ noteTitle }}
            <b>×</b>
          </span>
        </div>
        <div class="window-frame__content window-frame__note"><Prose :text="noteText" /></div>
      </section>

      <section
        v-if="isTab"
        class="window-frame__pane"
        :style="{ flexGrow: frame.note ? share : 1 }"
      >
        <div class="window-frame__strip">
          <span
            v-for="(tab, index) in storyTabs"
            :key="tab.title"
            class="window-frame__tab"
            :data-showing="index === 0 || undefined"
          >
            <component :is="TAB_ICONS[tab.icon]" class="window-frame__tab-icon" />
            {{ tab.title }}
            <b v-if="index === 0">×</b>
          </span>
        </div>
        <div class="window-frame__content">
          <div class="window-frame__fill"><slot /></div>
        </div>
      </section>

      <section
        v-if="isTab && frame.note === 'after'"
        class="window-frame__pane"
        :style="{ flexGrow: 1 - share }"
      >
        <div class="window-frame__strip">
          <span class="window-frame__tab" data-showing>
            <FileText class="window-frame__tab-icon" />
            {{ noteTitle }}
            <b>×</b>
          </span>
        </div>
        <div class="window-frame__content window-frame__note"><Prose :text="noteText" /></div>
      </section>
    </div>

    <slot v-if="frame.place === 'status-bar'" />
  </div>
</template>

<style>
.window-frame {
  display: flex;
  flex-direction: column;
  block-size: 100vh;
  overflow: hidden;
  background: var(--numen-surface);
  color: var(--numen-ink);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-font-size);
}

.window-frame__title-bar {
  position: relative;
  z-index: 20;
  display: grid;
  grid-template-columns: auto auto 1fr auto;
  align-items: center;
  flex: none;
  block-size: calc(2 * var(--numen-chrome-unit));
  background: var(--numen-titlebar);
  border-block-end: 1px solid var(--numen-rule);
  font-size: var(--numen-chrome-text-2);
  user-select: none;
}

.window-frame__controls {
  display: flex;
  gap: calc(0.5 * var(--numen-chrome-unit));
  padding-inline: calc(0.75 * var(--numen-chrome-unit));
}

.window-frame__controls i {
  inline-size: calc(0.75 * var(--numen-chrome-unit));
  block-size: calc(0.75 * var(--numen-chrome-unit));
  border-radius: 50%;
}

.window-frame__close {
  background: var(--numen-alarm);
}

.window-frame__minimize {
  background: var(--numen-caution-fg);
}

.window-frame__maximize {
  background: var(--numen-positive);
}

.window-frame__commands {
  display: flex;
  align-items: center;
  gap: calc(1 * var(--numen-chrome-unit));
  padding-inline: calc(0.5 * var(--numen-chrome-unit));
  color: var(--numen-ink);
}

.window-frame__title {
  text-align: center;
  white-space: nowrap;
}

.window-frame__status {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  block-size: 100%;
  min-inline-size: 7rem;
  padding-inline-end: calc(0.5 * var(--numen-chrome-unit));
}

.window-frame__body {
  display: flex;
  flex: 1 1 0;
  min-block-size: 0;
}

.window-frame__pane {
  display: flex;
  flex-direction: column;
  flex: 1 1 0;
  min-inline-size: 0;
  background: var(--numen-surface);
}

.window-frame__pane + .window-frame__pane {
  border-inline-start: var(--numen-stroke) solid var(--numen-rule);
}

.window-frame__pane--files {
  flex: 0 0 15.5rem;
}

.window-frame__strip {
  display: flex;
  flex: none;
  align-items: stretch;
  overflow: hidden;
  border-block-end: var(--numen-stroke) solid var(--numen-rule);
}

.window-frame__tab {
  display: flex;
  align-items: center;
  column-gap: 0.4rem;
  block-size: 1.4rem;
  padding-inline: 0.4rem;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  white-space: nowrap;
}

.window-frame__tab[data-showing] {
  background: var(--numen-raised);
  color: var(--numen-ink);
}

.window-frame__tab b {
  font-weight: normal;
  opacity: 0.55;
}

.window-frame__tab-icon {
  inline-size: 0.875rem;
  block-size: 0.875rem;
  stroke-width: 1.75;
  opacity: 0.75;
}

.window-frame__row-icon {
  flex: none;
  inline-size: 1rem;
  block-size: 1rem;
  color: var(--numen-hushed);
}

.window-frame__content {
  position: relative;
  flex: 1 1 0;
  min-block-size: 0;
}

.window-frame__fill {
  position: absolute;
  inset: 0;
  overflow: auto;
}

.window-frame__note {
  overflow: hidden;
  padding: 1.5rem 2rem;
}
</style>
