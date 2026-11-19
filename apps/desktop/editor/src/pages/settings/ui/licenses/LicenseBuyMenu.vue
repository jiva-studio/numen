<script setup lang="ts">
/**
 * The Buy dropdown button on a license card: opens a menu of the declared checkout providers with
 * their payment methods and external checkout links.
 */
import { ref, useTemplateRef } from 'vue'
import { ChevronDown, ExternalLink } from '@lucide/vue'
import { Button, useOutsidePress } from '@numen/ui'
import type { LicenseProviderOptionInfo } from '@/entities/license'
import { WORDS as words } from '../../words'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  options: readonly LicenseProviderOptionInfo[]
}>()

/* --------------------------------- State ---------------------------------- */
const isOpen = ref(false)
const buyGroup = useTemplateRef<HTMLElement>('buyGroup')

/* --------------------------------- Hooks ---------------------------------- */
useOutsidePress(
  buyGroup,
  () => {
    isOpen.value = false
  },
  () => isOpen.value,
)

/* -------------------------------- Handlers -------------------------------- */
function onToggle(): void {
  isOpen.value = !isOpen.value
}

function onClose(): void {
  isOpen.value = false
}
</script>

<template>
  <div ref="buyGroup" class="license-buy" @keydown.esc="onClose">
    <Button
      type="button"
      variant="outline"
      class="license-buy__button"
      :aria-expanded="isOpen"
      aria-haspopup="menu"
      @click="onToggle"
    >
      {{ words.licenseBuy }}
      <ChevronDown :size="14" aria-hidden="true" />
    </Button>
    <div v-if="isOpen" class="license-buy__menu" role="menu" :aria-label="words.licenseBuy">
      <a
        v-for="option in options"
        :key="option.provider"
        class="license-buy__item"
        role="menuitem"
        :href="option.checkoutUrl"
        target="_blank"
        rel="noopener noreferrer"
        @click="onClose"
      >
        <span class="license-buy__provider">{{ option.providerName }}</span>
        <span v-if="option.paymentMethods.length > 0" class="license-buy__methods">
          {{ option.paymentMethods.join(', ') }}
        </span>
        <ExternalLink :size="14" class="license-buy__icon" aria-hidden="true" />
      </a>
    </div>
  </div>
</template>

<style scoped>
.license-buy {
  position: relative;
  flex: none;
}

.license-buy__button {
  box-sizing: border-box;
  block-size: var(--numen-action-size);
  margin: 0;
}

.license-buy__menu {
  position: absolute;
  inset-block-start: calc(100% + 0.25rem);
  inset-inline-end: 0;
  z-index: var(--numen-lift-menu);
  display: flex;
  flex-direction: column;
  min-inline-size: 14rem;
  padding: 0.375rem;
  background-color: var(--numen-raised);
  border: var(--numen-stroke) solid var(--numen-panel-border);
  border-radius: var(--numen-radius-popup);
  box-shadow: var(--numen-panel-shadow);
}

.license-buy__item {
  display: grid;
  grid-template-columns: 1fr auto;
  align-items: center;
  column-gap: 0.75rem;
  padding: 0.375rem 0.5rem;
  border-radius: var(--numen-radius-popup-item);
  color: var(--numen-ink);
  text-decoration: none;
}

.license-buy__item:hover,
.license-buy__item:focus-visible {
  background-color: var(--numen-bubble-bg);
  outline: none;
}

.license-buy__provider {
  grid-column: 1;
  font-size: var(--numen-text-2);
  font-weight: 500;
}

.license-buy__methods {
  grid-column: 1;
  font-size: var(--numen-text-1);
  color: var(--numen-hushed);
}

.license-buy__icon {
  grid-column: 2;
  grid-row: 1 / span 2;
  color: var(--numen-hushed);
}
</style>
