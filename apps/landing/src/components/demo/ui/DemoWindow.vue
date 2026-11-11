<script setup lang="ts">
/**
 * The landing page's picture of the Numen window: the workspace of panes and tabs over fixed example
 * data. It reads and changes no vault.
 */
import { WorkspaceLayout } from '@numen/workbench'
import '@numen/ui/styles.css'
import '@numen/workbench/styles.css'
import { ENTROPY_BODY } from '../lib/note'
import { PDF_TAB_PATH } from '../lib/tabs'
import { useDemoWindow } from '../model/useDemoWindow'
import AgentView from './AgentView.vue'
import DeckView from './DeckView.vue'
import FilesView from './FilesView.vue'
import LinkView from './LinkView.vue'
import NoteView from './NoteView.vue'
import PdfView from './PdfView.vue'
import PlexView from './PlexView.vue'
import RecordingView from './RecordingView.vue'
import { getTabIcon } from './icons'

/* --------------------------------- State ---------------------------------- */
const { agent, player, workspace, tabs, close, getKind } = useDemoWindow()
</script>

<template>
  <div class="demo-window numen" tabindex="-1" @contextmenu.prevent>
    <WorkspaceLayout v-model="workspace" :tabs="tabs" :minimum="140" @close="close">
      <template #icon="{ id }">
        <component :is="getTabIcon(getKind(id) ?? 'note')" class="tab-icon" />
      </template>

      <template #tab="{ id }">
        <FilesView v-if="getKind(id) === 'files'" />
        <PlexView v-else-if="getKind(id) === 'plex'" />
        <NoteView v-else-if="getKind(id) === 'note'" :body="ENTROPY_BODY" />
        <PdfView v-else-if="getKind(id) === 'pdf'" :path="PDF_TAB_PATH" />
        <RecordingView v-else-if="getKind(id) === 'recording'" :player="player" />
        <LinkView v-else-if="getKind(id) === 'link'" />
        <DeckView v-else-if="getKind(id) === 'deck'" />
        <AgentView v-else-if="getKind(id) === 'agent'" :agent="agent" />
      </template>
    </WorkspaceLayout>
  </div>
</template>

<style scoped>
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

.tab-icon {
  inline-size: 0.9rem;
  block-size: 0.9rem;
  flex-shrink: 0;
  stroke-width: 1.75;
  opacity: 0.75;
}

.demo-window :deep(.tab__icon) {
  inline-size: auto;
  min-inline-size: 0.9rem;
}

.demo-window :deep(.workspace__dragged) {
  z-index: 60;
}
</style>
