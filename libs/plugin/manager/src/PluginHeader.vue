<script setup lang="ts">
/**
 * The header of a plugin page: the category icon, the name, the id, the version, the state and the
 * description, and on the right the enabled switch of an installed plugin or the install button.
 */
import { computed } from 'vue'
import { Button, Switch } from '@numen/ui'
import { getCategoryIcon } from './categoryIcons'
import { getPluginCategory } from './filter'
import { isInstalled, type ManagedPlugin } from './plugins'
import type { PluginState } from './state'
import { PLUGIN_WORDS as words } from './words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  plugin: ManagedPlugin
  state: PluginState
  isEnabled: boolean
  isPending: boolean
  canInstall: boolean
}>()

const emit = defineEmits<{
  (e: 'toggle-enabled', isEnabled: boolean): void
  (e: 'install'): void
}>()

/* --------------------------------- State ---------------------------------- */
const icon = computed(() => getCategoryIcon(getPluginCategory(props.plugin)))

/* -------------------------------- Handlers -------------------------------- */
function onToggleEnabled(isEnabled: boolean): void {
  emit('toggle-enabled', isEnabled)
}

function onInstall(): void {
  emit('install')
}
</script>

<template>
  <div class="plugin-header">
    <component
      :is="icon"
      class="plugin-header__glyph"
      :size="32"
      :stroke-width="1.5"
      aria-hidden="true"
    />
    <div class="plugin-header__heading">
      <h2 class="plugin-header__title">{{ props.plugin.name }}</h2>
      <code class="plugin-header__id">{{ props.plugin.id }}</code>
    </div>
    <div class="plugin-header__control">
      <Switch
        v-if="isInstalled(props.plugin)"
        :model-value="props.isEnabled"
        :is-busy="props.isPending"
        :aria-label="words.pluginSwitch(props.plugin.name)"
        @update:model-value="onToggleEnabled"
      />
      <Button v-else-if="props.canInstall" size="small" @click="onInstall">
        {{ words.pluginInstall }}
      </Button>
    </div>
    <p class="plugin-header__meta">
      <span v-if="props.plugin.version">{{ words.pluginVersion(props.plugin.version) }}</span>
      <span class="plugin-header__state">
        <span
          class="plugin-header__dot"
          :class="`plugin-header__dot--${props.state}`"
          aria-hidden="true"
        />
        {{ words.pluginState[props.state] }}
      </span>
    </p>
    <p v-if="props.plugin.description" class="plugin-header__description">
      {{ props.plugin.description }}
    </p>
  </div>
</template>

<style scoped>
.plugin-header {
  --glyph-size: 2rem;
  display: grid;
  grid-template-columns: var(--glyph-size) minmax(0, 1fr) auto;
  gap: 0.375rem 1rem;
  align-items: baseline;
}

.plugin-header__glyph {
  grid-row: span 3;
  align-self: start;
  inline-size: var(--glyph-size);
  block-size: var(--glyph-size);
  color: var(--numen-hushed);
}

.plugin-header__heading {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 0.25rem 0.75rem;
  min-inline-size: 0;
}

.plugin-header__title {
  margin: 0;
  color: var(--numen-ink);
  font-size: var(--numen-text-3);
  font-weight: 600;
  line-height: 1.3;
}

.plugin-header__id {
  color: var(--numen-hushed);
  font-family: var(--numen-font-mono);
  font-size: var(--numen-text-1);
}

.plugin-header__control {
  display: flex;
  align-self: center;
}

.plugin-header__meta,
.plugin-header__description {
  grid-column: 2 / -1;
  margin: 0;
  color: var(--numen-hushed);
}

.plugin-header__meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0 0.75rem;
  font-size: var(--numen-text-2);
  font-variant-numeric: tabular-nums;
}

.plugin-header__state {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
}

.plugin-header__dot {
  inline-size: 0.375rem;
  block-size: 0.375rem;
  border-radius: 50%;
  background-color: var(--numen-hushed);
}

.plugin-header__dot--stopped,
.plugin-header__dot--error {
  background-color: var(--numen-alarm);
}

.plugin-header__description {
  max-inline-size: 34rem;
}
</style>
