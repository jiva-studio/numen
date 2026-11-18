<script setup lang="ts">
/**
 * One license card on the licenses page: the title and status of the license, the plugins that
 * declare it and what each plugin unlocks, the key input (or masked hint when activated), and the
 * contextual Buy, Activate or Deactivate action.
 */
import { computed, ref, useTemplateRef } from 'vue'
import { ChevronDown, ExternalLink } from '@lucide/vue'
import { Button, runInBackground, useOutsidePress } from '@numen/ui'
import type { LicenseInfo } from '@/entities/license'
import type { Result } from '@/shared/lib/result'
import { formatError } from '@/shared/api/errors'
import { WORDS as words } from '../../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  license: LicenseInfo
  activate: (licenseId: string, key: string) => Promise<Result<LicenseInfo, string>>
  deactivate: (licenseId: string) => Promise<Result<LicenseInfo, string>>
}>()

/* --------------------------------- State ---------------------------------- */
const draftKey = ref('')
const isBusy = ref(false)
const isBuyOpen = ref(false)
const problem = ref('')
const buyGroup = useTemplateRef<HTMLElement>('buyGroup')

const displayTitle = computed(() =>
  props.license.title !== '' ? props.license.title : props.license.id,
)

const statusText = computed(() => {
  switch (props.license.status) {
    case 'active':
      return words.licenseStatusActive
    case 'expired':
      return words.licenseStatusExpired
    case 'unverified':
      return words.licenseStatusUnverified
    case 'inactive':
      return words.licenseStatusInactive
  }
})

const activeProviderName = computed(() => {
  const providerId = props.license.activeProvider
  if (providerId === '') return ''
  const match = props.license.providers.find((one) => one.provider === providerId)
  return match?.providerName ?? providerId
})

const activationSubtitle = computed(() =>
  props.license.hasKey
    ? words.licenseActivatedOn(props.license.instanceName, activeProviderName.value)
    : '',
)

const hasDraftKey = computed(() => draftKey.value.trim() !== '')

const buyOptions = computed(() =>
  props.license.providers.filter((one) => one.checkoutUrl !== ''),
)

/* --------------------------------- Hooks ---------------------------------- */
useOutsidePress(
  buyGroup,
  () => {
    isBuyOpen.value = false
  },
  () => isBuyOpen.value,
)

/* -------------------------------- Handlers -------------------------------- */
function onInputKey(event: Event): void {
  const target = event.target as HTMLInputElement | null
  draftKey.value = target?.value ?? ''
  if (problem.value !== '') problem.value = ''
}

function onSubmitActivate(): void {
  const trimmed = draftKey.value.trim()
  if (trimmed === '' || isBusy.value || props.license.hasKey) return
  isBusy.value = true
  problem.value = ''
  runInBackground(
    props
      .activate(props.license.id, trimmed)
      .then((answer) => {
        if (!answer.ok) {
          problem.value = answer.error
          return
        }
        draftKey.value = ''
      })
      .finally(() => {
        isBusy.value = false
      }),
    (caughtError) => {
      problem.value = formatError(caughtError)
    },
  )
}

function onDeactivate(): void {
  if (isBusy.value || !props.license.hasKey) return
  isBusy.value = true
  problem.value = ''
  runInBackground(
    props
      .deactivate(props.license.id)
      .then((answer) => {
        if (!answer.ok) {
          problem.value = answer.error
        }
      })
      .finally(() => {
        isBusy.value = false
      }),
    (caughtError) => {
      problem.value = formatError(caughtError)
    },
  )
}

function onToggleBuy(): void {
  isBuyOpen.value = !isBuyOpen.value
}

function onCloseBuy(): void {
  isBuyOpen.value = false
}
</script>

<template>
  <section class="settings-card license-card" :aria-label="displayTitle">
    <div class="license-card__header">
      <div class="license-card__titles">
        <div class="license-card__title-row">
          <h3 class="license-card__title">{{ displayTitle }}</h3>
          <code class="license-card__id">{{ license.id }}</code>
        </div>
        <p v-if="activationSubtitle !== ''" class="license-card__subtitle">
          {{ activationSubtitle }}
        </p>
      </div>
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

    <ul v-if="license.contributors.length > 0" class="license-card__contributors">
      <li
        v-for="contributor in license.contributors"
        :key="contributor.pluginId"
        class="license-card__contributor"
      >
        <span class="license-card__plugin">{{ contributor.pluginName }}</span>
        <span v-if="contributor.description !== ''" class="license-card__description">
          — {{ contributor.description }}
        </span>
      </li>
    </ul>

    <div class="license-card__controls">
      <template v-if="license.hasKey">
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
          :disabled="isBusy"
          @click="onDeactivate"
        >
          {{ isBusy ? words.licenseDeactivating : words.licenseDeactivate }}
        </Button>
      </template>

      <template v-else>
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
          :disabled="isBusy"
          @click="onSubmitActivate"
        >
          {{ isBusy ? words.licenseActivating : words.licenseActivate }}
        </Button>
        <div
          v-else-if="buyOptions.length > 0"
          ref="buyGroup"
          class="license-card__buy"
          @keydown.esc="onCloseBuy"
        >
          <Button
            type="button"
            variant="outline"
            :aria-expanded="isBuyOpen"
            aria-haspopup="menu"
            @click="onToggleBuy"
          >
            {{ words.licenseBuy }}
            <ChevronDown :size="14" aria-hidden="true" />
          </Button>
          <div
            v-if="isBuyOpen"
            class="license-card__buy-menu"
            role="menu"
            :aria-label="words.licenseBuy"
          >
            <a
              v-for="option in buyOptions"
              :key="option.provider"
              class="license-card__buy-item"
              role="menuitem"
              :href="option.checkoutUrl"
              target="_blank"
              rel="noopener noreferrer"
              @click="onCloseBuy"
            >
              <span class="license-card__buy-item-body">
                <span class="license-card__buy-provider">{{ option.providerName }}</span>
                <span
                  v-if="option.paymentMethods.length > 0"
                  class="license-card__buy-methods"
                >
                  {{ option.paymentMethods.join(', ') }}
                </span>
              </span>
              <ExternalLink :size="14" class="license-card__buy-icon" aria-hidden="true" />
            </a>
          </div>
        </div>
      </template>
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
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.75rem;
}

.license-card__titles {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  min-inline-size: 0;
}

.license-card__title-row {
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

.license-card__id {
  font-family: var(--numen-font-mono);
  font-size: var(--numen-text-1);
  color: var(--numen-hushed);
}

.license-card__subtitle {
  margin: 0;
  font-size: var(--numen-text-1);
  color: var(--numen-hushed);
}

.license-card__badge {
  flex: none;
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
  min-inline-size: 0;
  block-size: var(--numen-action-size);
  padding: 0.25rem 0.625rem;
  background-color: var(--numen-field-bg);
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  color: var(--numen-ink);
  font-family: var(--numen-font-mono);
  font-size: var(--numen-text-2);
}

.license-card__input:focus {
  outline: var(--numen-stroke) solid var(--numen-accent);
}

.license-card__input--readonly {
  color: var(--numen-hushed);
  background-color: var(--numen-bubble-bg);
}

.license-card__buy {
  position: relative;
  flex: none;
}

.license-card__buy-menu {
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

.license-card__buy-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.375rem 0.5rem;
  border-radius: var(--numen-radius-popup-item);
  color: var(--numen-ink);
  text-decoration: none;
}

.license-card__buy-item:hover,
.license-card__buy-item:focus-visible {
  background-color: var(--numen-bubble-bg);
  outline: none;
}

.license-card__buy-item-body {
  display: flex;
  flex-direction: column;
  min-inline-size: 0;
}

.license-card__buy-provider {
  font-size: var(--numen-text-2);
  font-weight: 500;
}

.license-card__buy-methods {
  font-size: var(--numen-text-1);
  color: var(--numen-hushed);
}

.license-card__buy-icon {
  flex: none;
  color: var(--numen-hushed);
}

.license-card__problem {
  margin: 0;
  font-size: var(--numen-text-1);
  color: var(--numen-alarm);
}
</style>
