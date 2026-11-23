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
const {
  identity: current,
  step,
  stage,
  recoveryWords,
  call,
  confirmation,
  typing,
  deletion,
  actions,
} = section
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
    <template #aside>{{ words.identityNone }}</template>
    <div class="identity-section__body">
      <p v-if="step !== 'restore'" class="identity-section__detail">
        {{ words.identityNoneDetail }}
      </p>

      <div v-if="step !== 'restore'" class="identity-section__actions">
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

      <IdentityRestore
        v-else
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
      :problem="call.problem.value"
      :is-busy="call.isBusy.value"
      @change="actions.setEntered"
      @check="actions.showCheck"
      @show-words="actions.showWords"
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
  padding-block: 0.375rem;
}

.identity-section__detail,
.identity-section__problem {
  margin: 0;
}

.identity-section__actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
}

.identity-section__problem {
  font-size: var(--numen-text-1);
  color: var(--numen-alarm);
}
</style>
