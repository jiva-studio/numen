<script setup lang="ts">
/**
 * An entry row of the settings list that opens a page of settings: System by default, others by props.
 * The `trail` slot holds tags drawn before the chevron.
 */
import { computed } from 'vue'
import { ChevronRight } from '@lucide/vue'
import { getCategoryIcon } from './categoryIcons'
import { PLUGIN_WORDS as words } from './words'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<{ name?: string; detail?: string; page?: string }>(), {
  name: words.systemSettings,
  detail: words.systemSettingsDetail,
  page: 'system',
})

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (e: 'select'): void
}>()

/* --------------------------------- State ---------------------------------- */
const icon = computed(() => getCategoryIcon(props.page))

/* -------------------------------- Handlers -------------------------------- */
function onClick(): void {
  emit('select')
}

function onKeydown(e: KeyboardEvent): void {
  if (e.key === 'Enter' || e.key === ' ') {
    e.preventDefault()
    emit('select')
  }
}
</script>

<template>
  <div
    class="system-row"
    role="button"
    tabindex="0"
    :aria-label="props.name"
    @click="onClick"
    @keydown="onKeydown"
  >
    <span class="system-row__glyph" aria-hidden="true">
      <component :is="icon" :size="20" :stroke-width="1.75" />
    </span>
    <div class="system-row__main">
      <span class="system-row__name">{{ props.name }}</span>
      <p v-if="props.detail" class="system-row__description">{{ props.detail }}</p>
    </div>
    <slot name="trail" />
    <ChevronRight class="system-row__chevron" :size="16" aria-hidden="true" />
  </div>
</template>

<style scoped>
.system-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  box-sizing: border-box;
  min-block-size: var(--settings-row-min, 3.25rem);
  padding: var(--settings-row-pad, 0.625rem) var(--settings-card-pad, 1rem);
  border-block-start: var(--numen-stroke) solid var(--numen-rule);
  cursor: pointer;
  user-select: none;
  transition: background-color var(--numen-motion-hover) ease;
}

.system-row:first-child {
  border-block-start: none;
}

.system-row:hover {
  background-color: color-mix(in oklab, var(--numen-raised), var(--numen-ink) 5%);
}

.system-row:focus-visible {
  outline: var(--numen-stroke) solid var(--numen-accent);
  outline-offset: calc(-1 * var(--numen-stroke));
}

.system-row__glyph {
  display: grid;
  place-items: center;
  flex: none;
  color: var(--numen-hushed);
}

.system-row__main {
  display: flex;
  flex-direction: column;
  flex: 1 1 auto;
  min-inline-size: 0;
}

.system-row__name {
  color: var(--numen-ink);
  font-weight: 500;
}

.system-row__description {
  margin: 0;
  color: var(--numen-hushed);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.system-row__chevron {
  color: var(--numen-hushed);
}
</style>
