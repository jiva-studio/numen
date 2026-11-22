<script setup lang="ts">
/**
 * One license card on the licenses page: the title and status of the license, the plugins that
 * declare it and what each plugin unlocks, the key input (or masked hint when activated), and the
 * contextual Buy, Activate or Deactivate action.
 */
import { Button } from '@numen/ui'
import type { LicenseInfo } from '@/entities/license'
import type { Result } from '@numen/result'
import { useLicenseCard } from '../../model/useLicensesPage'
import { WORDS as words } from '../../words'
import LicenseBuyMenu from './LicenseBuyMenu.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  license: LicenseInfo
  activate: (licenseId: string, key: string) => Promise<Result<LicenseInfo, string>>
  deactivate: (licenseId: string) => Promise<Result<LicenseInfo, string>>
}>()

/* --------------------------------- State ---------------------------------- */
const card = useLicenseCard({
  getLicense: () => props.license,
  activate: props.activate,
  deactivate: props.deactivate,
})
const { draftKey, isBusy, problem } = card.state
const { displayTitle, statusText, contributors, hasDraftKey, buyOptions } = card.view

/* -------------------------------- Handlers -------------------------------- */
function onInputKey(event: Event): void {
  const target = event.target as HTMLInputElement | null
  card.actions.setDraftKey(target?.value ?? '')
}

function onSubmitActivate(): void {
  card.actions.submitActivate()
}

function onDeactivate(): void {
  card.actions.submitDeactivate()
}
</script>

<template>
  <section class="settings-card license-card" :aria-label="displayTitle">
    <div class="license-card__header">
      <h3 class="license-card__title">{{ displayTitle }}</h3>
      <span
        class="license-card__badge"
        :class="{
          'license-card__badge--active': license.status === 'active',
          'license-card__badge--warn':
            license.status === 'expired' || license.status === 'unverified',
        }"
      >
        {{ statusText }}
      </span>
    </div>

    <ul v-if="contributors.length > 0" class="license-card__contributors">
      <li
        v-for="contributor in contributors"
        :key="contributor.pluginId"
        class="license-card__contributor"
      >
        <span v-if="contributor.pluginName !== ''" class="license-card__plugin">
          {{ contributor.pluginName }}
        </span>
        <span v-if="contributor.description !== ''" class="license-card__description">
          {{
            contributor.pluginName !== ''
              ? ` — ${contributor.description}`
              : contributor.description
          }}
        </span>
      </li>
    </ul>

    <div v-if="license.hasKey" class="license-card__controls">
      <input
        class="license-card__input license-card__input--readonly"
        type="text"
        readonly
        :value="license.keyHint"
        :aria-label="words.licenseKeyLabel(displayTitle)"
      />
      <Button
        type="button"
        variant="outline"
        class="license-card__button"
        :disabled="isBusy"
        @click="onDeactivate"
      >
        {{ isBusy ? words.licenseDeactivating : words.licenseDeactivate }}
      </Button>
    </div>

    <div v-else class="license-card__controls">
      <input
        class="license-card__input"
        type="text"
        autocomplete="off"
        spellcheck="false"
        :value="draftKey"
        :disabled="isBusy"
        :placeholder="words.licenseKeyPlaceholder"
        :aria-label="words.licenseKeyLabel(displayTitle)"
        @input="onInputKey"
        @keydown.enter.prevent="onSubmitActivate"
      />
      <Button
        v-if="hasDraftKey"
        type="button"
        variant="solid"
        class="license-card__button license-card__button--solid"
        :disabled="isBusy"
        @click="onSubmitActivate"
      >
        {{ isBusy ? words.licenseActivating : words.licenseActivate }}
      </Button>
      <LicenseBuyMenu v-else-if="buyOptions.length > 0" :options="buyOptions" />
    </div>

    <p v-if="problem !== ''" role="alert" class="license-card__problem">
      {{ problem }}
    </p>
  </section>
</template>

<style scoped>
.license-card {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: var(--settings-card-pad);
}

.license-card__header {
  display: flex;
  align-items: baseline;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.license-card__title {
  margin: 0;
  font-size: var(--numen-text-2);
  font-weight: 600;
  color: var(--numen-ink);
}

.license-card__badge {
  margin-inline-start: auto;
  padding: 0.125rem 0.5rem;
  border-radius: var(--numen-radius-tight);
  background-color: var(--numen-bubble-bg);
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  font-weight: 500;
  white-space: nowrap;
}

.license-card__badge--active {
  background-color: color-mix(in oklab, var(--numen-accent), transparent 82%);
  color: var(--numen-ink);
}

.license-card__badge--warn {
  background-color: var(--numen-alarm-bg);
  color: var(--numen-alarm);
}

.license-card__contributors {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  font-size: var(--numen-text-1);
  color: var(--numen-hushed);
}

.license-card__contributor {
  line-height: var(--numen-line-height);
}

.license-card__plugin {
  font-weight: 500;
  color: var(--numen-ink);
}

.license-card__controls {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.license-card__input {
  flex: 1;
  box-sizing: border-box;
  appearance: none;
  min-inline-size: 0;
  block-size: var(--numen-action-size);
  margin: 0;
  padding: 0 0.625rem;
  background-color: var(--numen-field-bg);
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  color: var(--numen-ink);
  font-family: var(--numen-font-mono);
  font-size: var(--numen-text-2);
  line-height: var(--numen-line-height);
}

.license-card__button {
  box-sizing: border-box;
  block-size: var(--numen-action-size);
  margin: 0;
}

.license-card__button--solid {
  border: var(--numen-stroke) solid transparent;
  background-clip: padding-box;
}

.license-card__input:focus {
  outline: var(--numen-stroke) solid var(--numen-accent);
}

.license-card__input--readonly {
  color: var(--numen-hushed);
  background-color: var(--numen-bubble-bg);
}

.license-card__problem {
  margin: 0;
  font-size: var(--numen-text-1);
  color: var(--numen-alarm);
}
</style>
