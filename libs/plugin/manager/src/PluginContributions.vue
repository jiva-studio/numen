<script setup lang="ts">
/**
 * What a plugin's manifest declares, one row per section: the label of the section, then each
 * entry with its title, its id or target where that differs, and the shortcut of a command.
 */
import { KeyCap } from '@numen/ui'
import { parseKeystroke } from './keystroke'
import type { PluginContribution } from './pluginInfo'
import { PLUGIN_WORDS as words } from './words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  contributions: readonly PluginContribution[]
  isApple: boolean
}>()
</script>

<template>
  <dl class="settings-card plugin-contributions">
    <div v-for="group in props.contributions" :key="group.kind" class="plugin-contributions__row">
      <dt class="plugin-contributions__kind">{{ words.contributionKinds[group.kind] }}</dt>
      <dd
        v-for="(item, index) in group.items"
        :key="`${index}-${item.text}`"
        class="plugin-contributions__item"
      >
        <span>{{ item.text }}</span>
        <code v-if="item.detail && item.detail !== item.text" class="plugin-contributions__detail">
          {{ item.detail }}
        </code>
        <KeyCap v-if="item.keys" :keys="parseKeystroke(item.keys, props.isApple)" />
      </dd>
    </div>
  </dl>
</template>

<style scoped>
.plugin-contributions {
  margin: 0;
}

.plugin-contributions__row {
  display: grid;
  grid-template-columns: 10rem minmax(0, 1fr);
  align-items: baseline;
  gap: 0.375rem 1rem;
  box-sizing: border-box;
  min-block-size: var(--settings-row-min);
  padding-block: var(--settings-row-pad);
  border-block-end: var(--numen-stroke) solid var(--numen-rule);
}

.plugin-contributions__row:last-child {
  border-block-end: none;
}

.plugin-contributions__kind {
  grid-column: 1;
  color: var(--numen-hushed);
}

.plugin-contributions__item {
  display: flex;
  flex-wrap: wrap;
  grid-column: 2;
  align-items: baseline;
  gap: 0.25rem 0.625rem;
  min-inline-size: 0;
  margin: 0;
  color: var(--numen-ink);
}

.plugin-contributions__detail {
  color: var(--numen-hushed);
  font-family: var(--numen-font-mono);
  font-size: var(--numen-text-1);
  overflow-wrap: anywhere;
}

@container settings (max-width: 34rem) {
  .plugin-contributions__row {
    grid-template-columns: minmax(0, 1fr);
  }

  .plugin-contributions__kind,
  .plugin-contributions__item {
    grid-column: 1;
  }
}
</style>
