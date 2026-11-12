<script setup lang="ts">
/**
 * The sections of a plugin page as a row of tabs over a hairline; the chosen tab is underlined in
 * the accent. The arrow keys move between the tabs and choose as they go.
 */
import { ref } from 'vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  tabs: readonly { readonly id: string; readonly text: string }[]
  label: string
}>()

const chosen = defineModel<string>({ required: true })

/* --------------------------------- State ---------------------------------- */
const buttons = ref<HTMLButtonElement[]>([])

/* -------------------------------- Handlers -------------------------------- */
function onKeydown(e: KeyboardEvent, index: number): void {
  const step = e.key === 'ArrowRight' ? 1 : e.key === 'ArrowLeft' ? -1 : 0
  if (step === 0) return
  e.preventDefault()
  const next = (index + step + props.tabs.length) % props.tabs.length
  const tab = props.tabs[next]
  if (!tab) return
  chosen.value = tab.id
  buttons.value[next]?.focus()
}
</script>

<template>
  <div class="plugin-tabs" role="tablist" :aria-label="props.label">
    <button
      v-for="(tab, index) in props.tabs"
      :key="tab.id"
      ref="buttons"
      type="button"
      role="tab"
      class="plugin-tabs__tab"
      :aria-selected="chosen === tab.id"
      :tabindex="chosen === tab.id ? 0 : -1"
      @click="chosen = tab.id"
      @keydown="onKeydown($event, index)"
    >
      {{ tab.text }}
    </button>
  </div>
</template>

<style scoped>
.plugin-tabs {
  display: flex;
  gap: 1.5rem;
  padding-inline: var(--settings-card-pad, 1rem);
  border-block-end: var(--numen-stroke) solid var(--numen-rule);
}

.plugin-tabs__tab {
  margin-block-end: calc(-1 * var(--numen-stroke));
  padding: 0.5rem 0;
  border: none;
  border-block-end: 2px solid transparent;
  background: none;
  color: var(--numen-hushed);
  font-family: inherit;
  font-size: var(--numen-text-2);
  font-weight: 500;
  cursor: pointer;
  transition: color var(--numen-motion-hover) ease;
}

.plugin-tabs__tab:hover {
  color: var(--numen-ink);
}

.plugin-tabs__tab[aria-selected='true'] {
  color: var(--numen-ink);
  border-block-end-color: var(--numen-accent);
}

.plugin-tabs__tab:focus-visible {
  outline: var(--numen-stroke) solid var(--numen-accent);
  outline-offset: -2px;
}
</style>
