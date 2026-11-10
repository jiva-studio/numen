<script setup lang="ts">
/**
 * The landing page's simulation of the Numen window: the workspace of panes and tabs, the tab menu and
 * the palette, over a vault held in memory.
 */
import { ref, type Component } from 'vue'
import { FileText, Link2, Link2Off } from '@lucide/vue'
import { Menu } from '@numen/ui'
import { Palette, WorkspaceLayout, commandKeyChord, type ActionWords } from '@numen/workbench'
import '@numen/ui/styles.css'
import '@numen/workbench/styles.css'
import { NAMING_COMMAND, KEYBOARD } from '../lib/palette'
import { getPdfPathOfTab, getKindOfTab } from '../lib/tabs'
import { createDemoWindow } from '../model/window'
import { UNLINK_ITEM } from '../model/tabMenu'
import AgentView from './AgentView.vue'
import DeckView from './DeckView.vue'
import FilesView from './FilesView.vue'
import LinkView from './LinkView.vue'
import NoteView from './NoteView.vue'
import PdfView from './PdfView.vue'
import PlexView from './PlexView.vue'
import RecordingView from './RecordingView.vue'
import { getCommandIcon, getFileIcon, getTabIcon } from './icons'

/* --------------------------------- State ---------------------------------- */
const demo = createDemoWindow()
const { parts, pdf, actions } = demo
const { vault, layout, files, plex, tabMenu, palette, agent, deck, player, links, commands } = parts
const root = ref<HTMLElement | null>(null)
const ACTION_KEY = commandKeyChord(KEYBOARD)
const ACTION_WORDS: ActionWords = { name: 'Actions', placeholder: 'Search actions', silence: 'No actions found' }

/* -------------------------------- Handlers -------------------------------- */
function onKeydown(event: KeyboardEvent) {
  const key = event.key.toLowerCase()
  if (!(event.metaKey || event.ctrlKey) || (key !== 'k' && key !== 'p')) return
  event.preventDefault()
  palette.actions.toggle(key === 'k' ? 'commands' : 'search')
}

/* -------------------------------- Helpers --------------------------------- */
function getMenuIcon(id: string): Component | null {
  if (id === UNLINK_ITEM) return Link2Off
  return tabMenu.isLinkTarget(id) ? getTabIcon(id) : null
}

function getPaletteIcon(id: string): Component | null {
  const naming = palette.state.naming.value
  if (palette.state.panel.value === 'naming') return getCommandIcon(naming ? NAMING_COMMAND[naming.role] : '') ?? FileText
  if (palette.state.panel.value === 'commands') return getCommandIcon(id)
  return getFileIcon(id, Boolean(vault.read.getNote(id)))
}
</script>

<template>
  <div ref="root" class="demo-window numen" tabindex="-1" @keydown="onKeydown" @contextmenu.prevent>
    <WorkspaceLayout
      v-model="layout.state.held.value"
      :tabs="layout.state.tabs.value"
      :minimum="140"
      :create-id="layout.actions.createPaneId"
      @activate="actions.activateTab"
      @close="layout.actions.close"
      @tab-context-menu="tabMenu.open"
    >
      <template #icon="{ id }">
        <component :is="getTabIcon(id)" class="tab-icon" />
        <Link2 v-if="links.getGroupOf(id)" class="tab-link" :data-color="links.getGroupOf(id)?.colorIndex" aria-label="Linked tab" />
      </template>

      <template #tab="{ id }">
        <FilesView v-if="getKindOfTab(id) === 'files'" :files="files" :vault="vault" @run="commands.runInTree" />
        <PlexView
          v-else-if="getKindOfTab(id) === 'plex'"
          :layout="plex.layout.value"
          @go="plex.go"
          @open="layout.actions.openVaultFile($event)"
          @run="commands.run"
        />
        <NoteView
          v-else-if="getKindOfTab(id) === 'note' && layout.actions.getNoteOfTab(id)"
          :body="layout.actions.getNoteOfTab(id)?.body ?? ''"
          @update:body="actions.writeBody(id, $event)"
          @open="actions.followLink($event, id)"
        />
        <PdfView
          v-else-if="getKindOfTab(id) === 'reader:pdf'"
          :path="getPdfPathOfTab(id)"
          :page="pdf.pages[getPdfPathOfTab(id)] ?? 0"
          @go="pdf.goToPage(getPdfPathOfTab(id), $event)"
        />
        <RecordingView v-else-if="getKindOfTab(id) === 'recording'" :player="player" />
        <LinkView v-else-if="getKindOfTab(id) === 'link'" />
        <DeckView v-else-if="getKindOfTab(id) === 'deck'" :deck="deck" />
        <AgentView v-else-if="getKindOfTab(id) === 'agent'" :agent="agent" @open="actions.openTurnSource" />
      </template>
    </WorkspaceLayout>

    <Menu
      v-if="tabMenu.tab.value"
      :items="tabMenu.items.value"
      :at="tabMenu.at.value"
      has-groups
      is-open
      @choose="tabMenu.choose"
      @dismiss="tabMenu.close"
    >
      <template #icon="{ id }">
        <component :is="getMenuIcon(id)" v-if="getMenuIcon(id)" class="tab-menu-icon" aria-hidden="true" />
      </template>
      <template #silence>No open tabs to link with</template>
    </Menu>

    <Palette
      v-if="palette.state.panel.value"
      v-model="palette.state.typed.value"
      :groups="palette.view.groups.value"
      :from="root"
      :to="root ?? 'body'"
      :crumb="palette.view.crumb.value"
      :step="palette.view.step.value"
      is-open
      :placeholder="palette.view.placeholder.value"
      name="Search the vault"
      :action-words="ACTION_WORDS"
      :action-key="ACTION_KEY"
      @choose="actions.choosePalette"
      @back="palette.actions.dismiss"
      @dismiss="palette.actions.dismiss"
    >
      <template #icon="{ id }">
        <component :is="getPaletteIcon(id)" v-if="getPaletteIcon(id)" class="command-icon" />
      </template>
    </Palette>
  </div>
</template>

<style scoped>
.tab-menu-icon {
  inline-size: 0.875rem;
  block-size: 0.875rem;
  stroke-width: 1.75;
}

.demo-window {
  position: relative;
  width: 100%;
  height: 100%;
  border-radius: 0 0 0.45rem 0.45rem;
  overflow: hidden;
  background: var(--numen-surface);
  color: var(--numen-ink);
  font-family: var(--numen-font-sans);
  text-align: left;
  user-select: none;
  -webkit-user-select: none;
}

.demo-window :deep(.palette) {
  position: absolute;
  --drop: 4.5rem;
  --tallest: 20rem;
}

.command-icon {
  inline-size: 100%;
  block-size: 100%;
  stroke-width: 1.75;
  opacity: 0.75;
}

.tab-icon {
  inline-size: 0.9rem;
  block-size: 0.9rem;
  flex-shrink: 0;
  stroke-width: 1.75;
  opacity: 0.75;
}

.tab-link {
  inline-size: 0.85rem;
  block-size: 0.85rem;
  margin-inline-start: 0.125rem;
  stroke-width: 2.25;
  color: var(--numen-relation-parent);
}

.tab-link[data-color='2'] {
  color: var(--numen-relation-child);
}

.tab-link[data-color='3'] {
  color: var(--numen-relation-jump);
}

.tab-link[data-color='4'] {
  color: var(--numen-relation-sibling);
}

.demo-window :deep(.tab__icon) {
  inline-size: auto;
  min-inline-size: 0.9rem;
}

.demo-window :deep(.workspace__dragged) {
  z-index: 60;
}
</style>
