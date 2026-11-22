<script setup lang="ts">
/**
 * The identity of this device: its fingerprints when it has one, and otherwise the way to make one
 * (write down twelve words, confirm three of them) or to restore one from its words.
 */
import { Button } from '@numen/ui'
import type { InstallationIdentity } from '../../types'
import { useIdentitySection } from '../../model/useIdentitySection'
import { WORDS as words } from '../../words'
import IdentityDetails from './IdentityDetails.vue'
import IdentityRestore from './IdentityRestore.vue'
import IdentitySetup from './IdentitySetup.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ identity: InstallationIdentity }>()

/* --------------------------------- State ---------------------------------- */
const section = useIdentitySection(() => props.identity)
const { step, recoveryWords, positions, entered, restoreText, problem, isBusy, isAskingDelete } =
  section.state
const { identity: current, actions } = section
</script>

<template>
  <section class="settings-card identity-section" :aria-label="words.identitySection">
    <IdentityDetails
      v-if="current"
      :info="current"
      :is-busy="isBusy"
      :is-asking-delete="isAskingDelete"
      @ask="actions.askDelete"
      @confirm="actions.confirmDelete"
      @cancel="actions.cancelDelete"
    />

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

      <IdentitySetup
        v-else-if="step === 'setup'"
        :recovery-words="recoveryWords"
        :positions="positions"
        :entered="entered"
        :is-busy="isBusy"
        @change="actions.setEntered"
        @keep="actions.keep"
        @cancel="actions.cancel"
      />

      <IdentityRestore
        v-else
        :text="restoreText"
        :is-busy="isBusy"
        @change="actions.setRestoreText"
        @restore="actions.restore"
        @cancel="actions.cancel"
      />
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
