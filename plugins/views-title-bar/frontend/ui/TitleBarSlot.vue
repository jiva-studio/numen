<script setup lang="ts">
/**
 * One slot of the bar: its commands as buttons, its submenus and its commands with items as buttons
 * that open them, its views in their places and its separators as gaps, each out of the area that
 * drags the window.
 */
import type { Component } from 'vue'
import type { SlotCommand } from '@numen/plugin-sdk'
import { formatShortcut } from '../lib/shortcut'
import { formatTooltip } from '../lib/title'
import { useIsDrawn } from '../model/useIsDrawn'
import TitleBarMenuButton from './TitleBarMenuButton.vue'
import type { TitleBarSlotProps } from './TitleBarSlot.types'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<TitleBarSlotProps>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (
    event: 'run-command',
    command: string,
    itemId?: string,
    values?: Readonly<Record<string, string>>,
  ): void
}>()

/* --------------------------------- State ---------------------------------- */
const isDrawn = useIsDrawn(
  () => props.getKey,
  () => props.hidden,
)

/* -------------------------------- Handlers -------------------------------- */
function onRunCommand(
  command: string,
  itemId?: string,
  values?: Readonly<Record<string, string>>,
): void {
  if (values) emit('run-command', command, itemId, values)
  else if (itemId === undefined) emit('run-command', command)
  else emit('run-command', command, itemId)
}

/* -------------------------------- Helpers --------------------------------- */
function getTooltip(entry: SlotCommand): string {
  return formatTooltip(entry.title, formatShortcut(entry.shortcut, props.platform))
}
</script>

<template>
  <div class="title-bar__slot" :data-title-bar-slot="name">
    <template v-for="entry in entries" :key="getKey(entry)">
      <template v-if="isDrawn(entry)">
        <span
          v-if="entry.kind === 'separator'"
          class="title-bar__gap"
          data-title-bar-gap
          :data-title-bar-entry="getKey(entry)"
          aria-hidden="true"
          style="--wails-draggable: no-drag"
        />
        <TitleBarMenuButton
          v-else-if="entry.kind === 'command' && entry.hasItems"
          :entries="[entry]"
          :label="entry.title"
          :icon="entry.icon"
          :tooltip="getTooltip(entry)"
          is-item-list
          :list-items="listItems"
          :report-error="reportError"
          :data-title-bar-entry="getKey(entry)"
          :data-title-bar-command="entry.command"
          @run-command="onRunCommand"
        />
        <button
          v-else-if="entry.kind === 'command'"
          type="button"
          class="title-bar__button"
          :class="{ 'title-bar__button--text': entry.icon === undefined }"
          :data-title-bar-entry="getKey(entry)"
          :data-title-bar-command="entry.command"
          data-title-bar-focusable
          :aria-label="entry.title"
          :title="getTooltip(entry)"
          style="--wails-draggable: no-drag"
          @click="onRunCommand(entry.command, undefined, entry.values)"
        >
          <component :is="entry.icon as Component" v-if="entry.icon !== undefined" :size="16" />
          <template v-else>{{ entry.title }}</template>
        </button>
        <TitleBarMenuButton
          v-else-if="entry.kind === 'submenu'"
          :entries="entry.entries"
          :label="entry.title"
          :icon="entry.icon"
          :data-title-bar-entry="getKey(entry)"
          :data-title-bar-submenu="entry.submenu"
          :list-items="listItems"
          :report-error="reportError"
          @run-command="onRunCommand"
        />
        <div
          v-else
          class="title-bar__view"
          :data-title-bar-entry="getKey(entry)"
          :data-title-bar-view="entry.key"
          style="--wails-draggable: no-drag"
        >
          <component :is="entry.component as Component" />
        </div>
      </template>
    </template>
  </div>
</template>
