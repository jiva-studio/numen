<script setup lang="ts">
/**
 * The identity of this device: its fingerprints, devices and recovery kit when it has one, and
 * otherwise the way to make one (a dialog of two steps: write down twelve words, check three of them)
 * or to restore one in a dialog from its words or from two sheets of a kit.
 */
import { ref } from 'vue'
import { KeyRound, Plus, QrCode } from '@lucide/vue'
import { IslandCard, Toast } from '@numen/ui'
import type { InstallationIdentity } from '../../types'
import { useIdentitySection } from '../../model/useIdentitySection'
import { useRecoveryKit } from '../../model/useRecoveryKit'
import { WORDS as words } from '../../words'
import { LinkDialog } from '@/features/identity-link'
import CardAction from './CardAction.vue'
import IdentityDetails from './IdentityDetails.vue'
import IdentityRestoreDialog from './IdentityRestoreDialog.vue'
import IdentitySetup from './IdentitySetup.vue'
import RecoveryKitDialog from './RecoveryKitDialog.vue'
import RecoveryKitRow from './RecoveryKitRow.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ identity: InstallationIdentity }>()

/* --------------------------------- State ---------------------------------- */
const kit = useRecoveryKit(() => props.identity)
const section = useIdentitySection(() => props.identity, kit.actions.hold)
const { identity: current, step, setup, call, confirmation, restoring, deletion, actions } = section
const { stage, recoveryWords } = setup
const isJoining = ref(false)
</script>

<template>
  <template v-if="current">
    <IdentityDetails
      :info="current"
      :problem="call.problem.value"
      :is-busy="call.isBusy.value || kit.call.isBusy.value"
      :is-asking-delete="deletion.isAskingDelete.value"
      :devices="identity.devices"
      @ask="deletion.askDelete"
      @confirm="deletion.confirmDelete"
      @cancel="deletion.cancelDelete"
    >
      <RecoveryKitRow
        :problem="kit.isAsking.value ? '' : kit.call.problem.value"
        :is-busy="kit.call.isBusy.value"
        @download="kit.actions.download"
      />
    </IdentityDetails>

    <RecoveryKitDialog
      v-if="kit.isAsking.value"
      :text="kit.proof.text.value"
      :reading="kit.proof.reading.value"
      :problem="kit.call.problem.value"
      :is-busy="kit.call.isBusy.value"
      @change="kit.proof.setText"
      @download="kit.actions.submit"
      @cancel="kit.actions.close"
    />

    <Toast :text="kit.isSaved.value ? words.kitSaved : ''" />
  </template>

  <IslandCard v-else :title="words.identitySection">
    <template #aside>
      <CardAction :is-disabled="call.isBusy.value" @click="actions.startRestore">
        <KeyRound :size="14" aria-hidden="true" />
        {{ words.identityRestore }}
      </CardAction>
      <CardAction :is-disabled="call.isBusy.value" @click="isJoining = true">
        <QrCode :size="14" aria-hidden="true" />
        {{ words.identityJoin }}
      </CardAction>
      <CardAction is-primary :is-disabled="call.isBusy.value" @click="actions.startSetup">
        <Plus :size="14" aria-hidden="true" />
        {{ words.identityCreate }}
      </CardAction>
    </template>
    <div class="identity-section__body">
      <p class="identity-section__detail">{{ words.identityNoneDetail }}</p>
      <p class="identity-section__detail">{{ words.identityNoneWords }}</p>

      <LinkDialog
        v-if="isJoining && step === 'idle'"
        role="joiner"
        :calls="identity.devices.link"
        @close="isJoining = false"
      />

      <p
        v-if="call.problem.value !== '' && step === 'idle'"
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
      :wrong="confirmation.wrong.value"
      :is-complete="confirmation.isComplete.value"
      :problem="call.problem.value"
      :is-busy="call.isBusy.value"
      @change="actions.setEntered"
      @check="setup.showCheck"
      @back="setup.showWords"
      @keep="actions.keep"
      @cancel="actions.cancel"
    />

    <IdentityRestoreDialog
      v-if="step === 'restore'"
      :text="restoring.text.value"
      :reading="restoring.reading.value"
      :problem="call.problem.value"
      :is-busy="call.isBusy.value"
      @change="restoring.setText"
      @restore="actions.restore"
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
  font-size: var(--numen-text-2);
  color: var(--numen-alarm);
}
</style>
