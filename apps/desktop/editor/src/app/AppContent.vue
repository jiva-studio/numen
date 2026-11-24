<script setup lang="ts">
/**
 * The content between the title bar and the status bar: the workspace layout with the tabs of the
 * window, and the screen shown when no tab is open.
 */
import { WorkspaceLayout } from '@numen/workbench'
import { TabLinkIndicator } from '@/features/tab-linking'
import { WelcomeScreen } from '@/pages/welcome'
import { BootstrapFailure } from '@/features/app-start'
import { getUserAgent } from './window/getUserAgent'
import type { WindowParts } from './window/createWindow'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  parts: WindowParts
}>()

/* --------------------------------- State ---------------------------------- */
const { tabs, tabParts, vault, getTarget, palettes, views, tabMenu, bootstrap } = props.parts
const { windowTabs, tabLinks } = tabs
const { layout } = windowTabs.panes
const { detachTab } = tabParts.moves
const { getTabIcon } = tabParts.openTabs
const { isReady: isLayoutReady } = tabParts.layout
const { openCurrentVault } = tabParts
const { commands, search } = palettes.palette
const { runCommand } = palettes.palette.runs
const { commandDeps } = palettes
const { pluginWays } = palettes.pluginPalette
const { windowEdge } = views
const { vaultList } = vault.vaults
const { failure: bootstrapFailure, retry: retryBootstrap } = bootstrap
</script>

<template>
  <WorkspaceLayout
    v-model="layout"
    class="fill"
    :tabs="windowTabs.tabs.tabs.value"
    :window-edge="windowEdge"
    :tab-menu="tabMenu.menu.value"
    @close="windowTabs.panes.closeTab"
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
</template>

<style scoped>

.fill {
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
