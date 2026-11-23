<script setup lang="ts">
/**
 * The identity of this device: its fingerprints when it has one, and otherwise the way to make one
 * (write down twelve words, confirm three of them) or to restore one from its words.
 */
import { Button } from '@numen/ui'
import type { InstallationIdentity } from '../../types'
import { useIdentitySection } from '../../model/useIdentitySection'
import { WORDS as words } from '../../words'
import { DevicesSection } from '@/features/identity-link'
import IdentityDetails from './IdentityDetails.vue'
import IdentityRestore from './IdentityRestore.vue'
import IdentitySetup from './IdentitySetup.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ identity: InstallationIdentity }>()

/* --------------------------------- State ---------------------------------- */
const section = useIdentitySection(() => props.identity)
const {
  identity: current,
  step,
  recoveryWords,
  call,
  confirmation,
  typing,
  deletion,
  actions,
} = section
</script>

<template>
  <section class="settings-card identity-section" :aria-label="words.identitySection">
    <template v-if="current">
      <IdentityDetails
        :info="current"
        :is-busy="call.isBusy.value"
        :is-asking-delete="deletion.isAskingDelete.value"
        @ask="deletion.askDelete"
        @confirm="deletion.confirmDelete"
        @cancel="deletion.cancelDelete"
      />
      <DevicesSection v-if="!current.isRevoked" :calls="identity.devices" has-identity />
    </template>

    <template v-else>
      <div class="identity-section__empty">
        <p class="identity-section__title">{{ words.identityNone }}</p>
        <p v-if="step === 'idle'" class="identity-section__detail">
          {{ words.identityNoneDetail }}
        </p>
      </div>

      <DevicesSection v-if="step === 'idle'" :calls="identity.devices" :has-identity="false" />

      <div v-if="step === 'idle'" class="identity-section__actions">
        <Button
          type="button"
          variant="solid"
          :disabled="call.isBusy.value"
          @click="actions.startSetup"
        >
          {{ words.identityCreate }}
        </Button>
        <Button
          type="button"
          variant="outline"
          :disabled="call.isBusy.value"
          @click="actions.startRestore"
        >
          {{ words.identityRestore }}
        </Button>
      </div>

      <IdentitySetup
        v-else-if="step === 'setup'"
        :recovery-words="recoveryWords"
        :positions="confirmation.positions.value"
        :entered="confirmation.entered.value"
        :is-busy="call.isBusy.value"
        @change="actions.setEntered"
        @keep="actions.keep"
        @cancel="actions.cancel"
      />

      <IdentityRestore
        v-else
        :text="typing.restoreText.value"
        :is-busy="call.isBusy.value"
        @change="typing.setRestoreText"
        @restore="actions.restore"
        @cancel="actions.cancel"
      />
    </template>

    <p v-if="call.problem.value !== ''" role="alert" class="identity-section__problem">
      {{ call.problem.value }}
    </p>
  </section>
</template>

<style scoped>
.identity-section {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: var(--settings-card-pad);
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
