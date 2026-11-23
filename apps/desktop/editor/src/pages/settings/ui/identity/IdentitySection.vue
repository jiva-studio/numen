<script setup lang="ts">
/**
 * The identity of this device: its fingerprints and devices when it has one, and otherwise the way
 * to make one (a dialog of two steps: write down twelve words, check three of them) or to restore
 * one from its words.
 */
import { Button, IslandCard } from '@numen/ui'
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
const { identity: current, step, setup, call, confirmation, typing, deletion, actions } = section
const { stage, recoveryWords } = setup
</script>

<template>
  <IdentityDetails
    v-if="current"
    :info="current"
    :problem="call.problem.value"
    :is-busy="call.isBusy.value"
    :is-asking-delete="deletion.isAskingDelete.value"
    @ask="deletion.askDelete"
    @confirm="deletion.confirmDelete"
    @cancel="deletion.cancelDelete"
  />

  <IslandCard v-else :title="words.identitySection">
    <template v-if="step !== 'restore'" #aside>
      <Button
        type="button"
        variant="ghost"
        size="small"
        :disabled="call.isBusy.value"
        @click="actions.startRestore"
      >
        {{ words.identityRestore }}
      </Button>
      <Button
        type="button"
        variant="solid"
        size="small"
        :disabled="call.isBusy.value"
        @click="actions.startSetup"
      >
        {{ words.identityCreate }}
      </Button>
    </template>
    <div class="identity-section__body">
      <p class="identity-section__detail">{{ words.identityNoneDetail }}</p>

      <IdentityRestore
        v-if="step === 'restore'"
        :text="typing.restoreText.value"
        :is-busy="call.isBusy.value"
        @change="typing.setRestoreText"
        @restore="actions.restore"
        @cancel="actions.cancel"
      />

      <p
        v-if="call.problem.value !== '' && step !== 'setup'"
        role="alert"
        class="identity-section__problem"
      >
        {{ call.problem.value }}
      </p>
    </div>

    <IdentitySetup
      v-if="step === 'setup'"
      :stage="stage"
      :recovery-words="recoveryWords"
      :positions="confirmation.positions.value"
      :entered="confirmation.entered.value"
      :is-complete="confirmation.isComplete.value"
      :problem="call.problem.value"
      :is-busy="call.isBusy.value"
      @change="actions.setEntered"
      @check="actions.showCheck"
      @back="actions.showWords"
      @keep="actions.keep"
      @cancel="actions.cancel"
    />
  </IslandCard>
</template>

<style scoped>
.identity-section__body {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding-block: calc(var(--numen-card-pad) - var(--numen-card-inset));
}

.identity-section__detail,
.identity-section__problem {
  margin: 0;
}

.identity-section__problem {
  font-size: var(--numen-text-1);
  color: var(--numen-alarm);
}
</style>
