<script setup lang="ts">
/** A titled group: a card whose tinted header holds the title and may carry an aside at its right, with the rows below it. The body pads like a card of settings rows. The aside shows the labels of its actions while they fit, then only their icons, then only a "..." menu. */
import { nextTick, onBeforeUnmount, onMounted, ref, useTemplateRef } from 'vue'

defineProps<{ title: string }>()

const LEVELS = ['full', 'icons', 'menu'] as const
const header = useTemplateRef<HTMLElement>('header')
const level = ref<(typeof LEVELS)[number]>('full')
let watcher: ResizeObserver | undefined

async function fit(): Promise<void> {
  const element = header.value
  if (element === null) return
  for (const next of LEVELS) {
    level.value = next
    await nextTick()
    if (element.scrollWidth <= element.clientWidth) return
  }
}

onMounted(() => {
  fit().catch(() => undefined)
  if (typeof ResizeObserver === 'undefined' || header.value === null) return
  watcher = new ResizeObserver(() => fit().catch(() => undefined))
  watcher.observe(header.value)
})

onBeforeUnmount(() => watcher?.disconnect())
</script>

<template>
  <section class="island-card numen" :class="`island-card--${level}`">
    <header ref="header" class="island-card__header">
      <h3 class="island-card__title">{{ title }}</h3>
      <div v-if="$slots.aside" class="island-card__aside"><slot name="aside" /></div>
    </header>
    <div class="island-card__body">
      <slot />
    </div>
  </section>
</template>

<style scoped>
.island-card {
  overflow: hidden;
  box-sizing: border-box;
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius);
  background-color: var(--numen-raised);
}

.island-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  padding: 0.375rem var(--numen-card-pad);
  background-color: color-mix(in oklab, var(--numen-raised), var(--numen-ink) 6%);
}

.island-card__title {
  margin: 0;
  flex: 0 1 auto;
  min-inline-size: 5rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: var(--numen-text-2);
  font-weight: 600;
  color: var(--numen-ink);
}

.island-card__aside {
  display: flex;
  flex: none;
  align-items: center;
  gap: 0.5rem;
  font-size: var(--numen-text-2);
  color: var(--numen-hushed);
}

.island-card__body {
  box-sizing: border-box;
  padding: var(--numen-card-inset) var(--numen-card-pad);
}
</style>

<style>
/* The last action moves right so its box stands as far from the right edge as from the top. */
.island-card__aside:has(button, [role="button"]) {
  margin-inline-end: calc(0.375rem - var(--numen-card-pad));
}

.island-card--icons .card-action__label,
.island-card--menu .card-action__label,
.island-card--menu .card-action--secondary {
  display: none;
}

.island-card--menu .card-action--more {
  display: inline-flex;
}
</style>
