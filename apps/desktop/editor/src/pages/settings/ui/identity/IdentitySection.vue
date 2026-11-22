<script setup lang="ts">
/**
 * The identity of this device: its fingerprints when it has one, and otherwise the way to make one
 * (write down twelve words, confirm three of them) or to restore one from its words.
 */
import { Button } from '@numen/ui'
import type { InstallationIdentity } from '../../types'
import { useIdentitySection } from '../../model/useIdentitySection'
import { WORDS as words } from '../../words'
import { ConfirmDialog } from '../confirm-dialog'
import ConfirmWords from './ConfirmWords.vue'
import RecoveryWords from './RecoveryWords.vue'
import RestoreWords from './RestoreWords.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ identity: InstallationIdentity }>()

/* --------------------------------- State ---------------------------------- */
const section = useIdentitySection(() => props.identity)
const { step, recoveryWords, positions, entered, restoreText, problem, isBusy, isAskingDelete } =
  section.state
const { identity } = section.view
const actions = section.actions
</script>

<template>
  <section class="settings-card identity-section" :aria-label="words.identitySection">
    <template v-if="identity">
      <dl class="identity-section__facts">
        <dt>{{ words.identityAccount }}</dt>
        <dd class="identity-section__fingerprint">{{ identity.accountFingerprint }}</dd>
        <dt>{{ words.identityThisDevice }}</dt>
        <dd>
          <span class="identity-section__device">{{ identity.deviceName }}</span>
          <span class="identity-section__fingerprint">{{ identity.deviceFingerprint }}</span>
        </dd>
      </dl>
      <div class="identity-section__actions">
        <Button type="button" variant="outline" :disabled="isBusy" @click="actions.askDelete">
          {{ words.identityDelete }}
        </Button>
      </div>
      <ConfirmDialog
        v-if="isAskingDelete"
        tone="danger"
        :text="words.identityDeleteWarning"
        :label="words.identityDeleteTitle"
        :confirm-text="words.identityDeleteConfirm"
        :cancel-text="words.identityDeleteCancel"
        @confirm="actions.confirmDelete"
        @cancel="actions.cancelDelete"
      />
    </template>

    <template v-else>
      <div class="identity-section__empty">
        <p class="identity-section__title">{{ words.identityNone }}</p>
        <p v-if="step === 'idle'" class="identity-section__detail">{{ words.identityNoneDetail }}</p>
      </div>

      <div v-if="step === 'idle'" class="identity-section__actions">
        <Button type="button" variant="solid" :disabled="isBusy" @click="actions.startSetup">
          {{ words.identityCreate }}
        </Button>
        <Button type="button" variant="outline" :disabled="isBusy" @click="actions.startRestore">
          {{ words.identityRestore }}
        </Button>
      </div>

      <template v-else-if="step === 'setup'">
        <RecoveryWords :recovery-words="recoveryWords" />
        <ConfirmWords
          :positions="positions"
          :entered="entered"
          :is-disabled="isBusy"
          @change="actions.setEntered"
          @submit="actions.keep"
        />
        <div class="identity-section__actions">
          <Button type="button" variant="solid" :disabled="isBusy" @click="actions.keep">
            {{ isBusy ? words.identityKeeping : words.identityKeep }}
          </Button>
          <Button type="button" variant="outline" :disabled="isBusy" @click="actions.cancel">
            {{ words.identityCancel }}
          </Button>
        </div>
      </template>

      <template v-else>
        <RestoreWords
          :text="restoreText"
          :is-disabled="isBusy"
          @change="actions.setRestoreText"
        />
        <div class="identity-section__actions">
          <Button type="button" variant="solid" :disabled="isBusy" @click="actions.restore">
            {{ isBusy ? words.identityRestoring : words.identityRestoreAction }}
          </Button>
          <Button type="button" variant="outline" :disabled="isBusy" @click="actions.cancel">
            {{ words.identityCancel }}
          </Button>
        </div>
      </template>
    </template>

    <p v-if="problem !== ''" role="alert" class="identity-section__problem">{{ problem }}</p>
  </section>
</template>

<style scoped>
.identity-section {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: var(--settings-card-pad);
}

.identity-section__facts {
  display: grid;
  grid-template-columns: max-content 1fr;
  gap: 0.375rem 1rem;
  margin: 0;
}

.identity-section__facts dt {
  color: var(--numen-hushed);
}

.identity-section__facts dd {
  display: flex;
  flex-wrap: wrap;
  gap: 0.25rem 0.75rem;
  margin: 0;
}

.identity-section__fingerprint {
  font-family: var(--numen-font-mono);
}

.identity-section__empty {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.identity-section__title {
  margin: 0;
  font-weight: 500;
}

.identity-section__detail {
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}

.identity-section__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.identity-section__problem {
  margin: 0;
  font-size: var(--numen-text-1);
  color: var(--numen-alarm);
}
</style>
