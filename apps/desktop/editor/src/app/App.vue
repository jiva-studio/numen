<script setup lang="ts">
/**
 * The window: one vault, and tabs to divide the screen between.
 *
 * The window's structure is made by `createWindow` and handed in. This file holds what the user
 * sees of it, and the binding between the two.
 */
import { WorkspaceLayout } from '@numen/workbench'
import { SlotViewHost } from '@numen/plugin-loader'
import { LoadingIndicator } from '@numen/ui'
import '@numen/ui/styles.css'
import '@numen/workbench/styles.css'
import { UnsavedChangesPrompt } from '@/features/file-conflict'
import CoreFailureNotice from './CoreFailureNotice.vue'
import { CommandPalette } from '@/features/command-palette'
import { TabLinkIndicator } from '@/features/tab-linking'
import { WelcomeScreen } from '@/pages/welcome'
import { BootstrapFailure } from '@/features/app-start'
import { getUserAgent } from './window/getUserAgent'
import type { WindowParts } from './window/createWindow'
import { WORDS as words } from '@/shared/ui/words'
import { WINDOW_WORDS } from './window/words'
import { WORDS as note } from '@/entities/note'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  parts: WindowParts
}>()

/* --------------------------------- State ---------------------------------- */
/** What the notes still unwritten are put in: the window's words and a note's. */
const unsaved = {
  going: words.going,
  keep: note.keep,
  take: note.take,
  later: words.later,
}

const { tabs, tabParts, vault, getTarget, palettes, views, tabMenu, bootstrap } = props.parts
const { windowTabs, tabLinks, tabOpeners } = tabs
const { isHighlighted } = tabs.linkSession
const { layout } = windowTabs.panes
const { detachTab } = tabParts.moves
const { getTabIcon } = tabParts.openTabs
const { isReady: isLayoutReady } = tabParts.layout
const { openCurrentVault } = tabParts
const { commands, search } = palettes.palette
const { runCommand, runChoice } = palettes.palette.runs
const { commandDeps } = palettes
const { pluginWays } = palettes.pluginPalette
const { titleBarView, statusBarView, windowEdge } = views
const { fileFlush, getTitle } = vault.editing
const { failure } = vault.window.health
const { vaultList } = vault.vaults
const { isStarted, failure: bootstrapFailure, retry: retryBootstrap } = bootstrap
</script>

<template>
  <!-- The window is drawn once the plugins of the window slots have started or the wait for them has
       passed, so the layout does not move when the bar is drawn. -->
  <LoadingIndicator v-if="!isStarted" class="starting" :label="WINDOW_WORDS.appName" />

  <main v-else :class="{ 'is-highlighted': isHighlighted }">
    <SlotViewHost :view="titleBarView" />

    <CoreFailureNotice :failure="failure" />

    <WorkspaceLayout
      v-model="layout"
      class="below"
      :tabs="windowTabs.tabs.tabs.value"
      :window-edge="windowEdge"
      :tab-menu="tabMenu.menu.value"
      @close="windowTabs.closeTab"
      @show="windowTabs.panes.onTabShown"
      @tab-context-menu="tabMenu.open"
      @tab-menu-choose="tabMenu.choose"
      @tab-menu-dismiss="tabMenu.dismiss"
      @tab-menu-highlight="tabMenu.highlight"
      @tear-off="detachTab"
    >
      <template #tab-menu-icon="{ id }">
        <component
          :is="tabMenu.getIcon(id)"
          v-if="tabMenu.getIcon(id)"
          class="tab-menu-icon"
          aria-hidden="true"
        />
      </template>

      <template #icon="{ id }">
        <component :is="getTabIcon(id)" v-if="getTabIcon(id)" class="tab-icon" />
        <TabLinkIndicator
          :is-linked="tabLinks.getGroupOf(id) !== null"
          :color-index="tabLinks.getGroupOf(id)?.colorIndex"
        />
      </template>

      <template #tab="{ id }">
        <component
          :is="windowTabs.tabs.getTab(id)!.kind.pane"
          v-if="windowTabs.tabs.getTab(id)"
          :state="windowTabs.tabs.getTab(id)!.state"
        />

        <div v-else />
      </template>

      <template #silence>
        <BootstrapFailure
          v-if="bootstrapFailure"
          :message="bootstrapFailure"
          @retry="retryBootstrap"
        />
        <WelcomeScreen
          v-else-if="isLayoutReady || !vaultList.showing"
          :vault-list="vaultList"
          :tabs="windowTabs.tabs.tabs.value"
          :commands="commands"
          :search="search"
          :doing="commandDeps"
          :get-target="getTarget"
          :plugin-ways="pluginWays"
          :run-command="runCommand"
          :open-current-vault="openCurrentVault"
          :agent="getUserAgent()"
        />
        <div v-else />
      </template>
    </WorkspaceLayout>

    <SlotViewHost :view="statusBarView" />

    <UnsavedChangesPrompt
      :conflicts="fileFlush.conflicts.value"
      :called="getTitle"
      :words="unsaved"
    />

    <CommandPalette
      :commands="commands"
      :search="search"
      :doing="commandDeps"
      :get-target="getTarget"
      :places="{ findFileIcon: tabOpeners.findFileIcon }"
      :run-choice="runChoice"
    />
  </main>
</template>

<style scoped>
main {
  display: flex;
  flex-direction: column;
  height: 100vh;
}

/* Another window offers to move a tab into this one. */
main.is-highlighted::after {
  --ring-width: 0.1875rem;
  content: '';
  position: fixed;
  inset: 0;
  z-index: var(--numen-lift-menu);
  border: var(--ring-width) solid var(--numen-accent);
  border-radius: var(--numen-window-radius);
  pointer-events: none;
}

.starting {
  height: 100vh;
}

.below {
  flex: 1;
  min-height: 0;
}

.tab-menu-icon {
  --size: 0.875rem;
  inline-size: var(--size);
  block-size: var(--size);
  stroke-width: 1.75;
}

/* Lucide draws on a 24 grid, and the stroke is given in those units. */
.tab-icon {
  inline-size: 100%;
  block-size: 100%;
  stroke-width: 1.75;
  opacity: 0.75;
}
</style>
