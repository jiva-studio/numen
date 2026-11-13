<script setup lang="ts">
/**
 * The header of a plugin page: the category icon, the name, the id, the version and the
 * description. The state and the enabled switch stand in the toolbar of the page.
 */
import { computed } from 'vue'
import { getCategoryIcon } from './categoryIcons'
import { getPluginCategory } from './filter'
import type { ManagedPlugin } from './plugins'
import { PLUGIN_WORDS as words } from './words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  plugin: ManagedPlugin
}>()

/* --------------------------------- State ---------------------------------- */
const icon = computed(() => getCategoryIcon(getPluginCategory(props.plugin)))
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
    <p v-if="props.plugin.version" class="plugin-header__meta">
      {{ words.pluginVersion(props.plugin.version) }}
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
  grid-template-columns: var(--glyph-size) minmax(0, 1fr);
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

.plugin-header__meta,
.plugin-header__description {
  grid-column: 2;
  margin: 0;
  color: var(--numen-hushed);
}

.plugin-header__meta {
  font-size: var(--numen-text-2);
  font-variant-numeric: tabular-nums;
}

.plugin-header__description {
  max-inline-size: 34rem;
}
</style>
