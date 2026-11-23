<script setup lang="ts">
/**
 * The identity of this device: its fingerprints, devices and recovery kit when it has one, and
 * otherwise the way to make one (a dialog of two steps: write down twelve words, check three of
 * them) or to restore one from its words or from two sheets of a kit.
 */
import { computed } from 'vue'
import { FileText, KeyRound, Plus } from '@lucide/vue'
import { IslandCard } from '@numen/ui'
import type { Printer } from '@/pages/settings/api/print'
import type { InstallationIdentity } from '../../types'
import { useIdentityCall } from '../../model/useIdentityCall'
import { useIdentitySection } from '../../model/useIdentitySection'
import { useRecoveryKit } from '../../model/useRecoveryKit'
import { useSheetFields } from '../../model/useSheetFields'
import { WORDS as words } from '../../words'
import CardAction from './CardAction.vue'
import IdentityDetails from './IdentityDetails.vue'
import IdentityRestore from './IdentityRestore.vue'
import IdentityRestoreSheets from './IdentityRestoreSheets.vue'
import IdentitySetup from './IdentitySetup.vue'
import RecoveryKitSection from './RecoveryKitSection.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ identity: InstallationIdentity; printer: Printer }>()

/* --------------------------------- State ---------------------------------- */
const call = useIdentityCall()
const fields = useSheetFields()
const kit = useRecoveryKit(
  () => props.identity,
  () => props.printer,
  call,
  fields,
)
const section = useIdentitySection(() => props.identity, call, fields, kit.actions.showAfterSetup)
const { identity: current, step, setup, confirmation, restoring, deletion, actions } = section
const { stage, recoveryWords } = setup
const isRestoring = computed(() => step.value === 'restore' || step.value === 'restoreSheets')
</script>

<template>
  <template v-if="current">
    <IdentityDetails
      :info="current"
      :problem="call.problem.value"
      :is-busy="call.isBusy.value"
      :is-asking-delete="deletion.isAskingDelete.value"
      @ask="deletion.askDelete"
      @confirm="deletion.confirmDelete"
      @cancel="deletion.cancelDelete"
    />

    <RecoveryKitSection :model="kit" :texts="fields.texts.value" :is-busy="call.isBusy.value" />
  </template>

  <IslandCard v-else :title="words.identitySection">
    <template v-if="!isRestoring" #aside>
      <CardAction :is-disabled="call.isBusy.value" @click="actions.startRestore('restore')">
        <KeyRound :size="14" aria-hidden="true" />
        {{ words.identityRestore }}
      </CardAction>
      <CardAction :is-disabled="call.isBusy.value" @click="actions.startRestore('restoreSheets')">
        <FileText :size="14" aria-hidden="true" />
        {{ words.identityRestoreSheets }}
      </CardAction>
      <CardAction is-primary :is-disabled="call.isBusy.value" @click="actions.startSetup">
        <Plus :size="14" aria-hidden="true" />
        {{ words.identityCreate }}
      </CardAction>
    </template>
    <div class="identity-section__body">
      <p class="identity-section__detail">{{ words.identityNoneDetail }}</p>
      <p class="identity-section__detail">{{ words.identityNoneWords }}</p>

      <IdentityRestore
        v-if="step === 'restore'"
        :text="restoring.typing.restoreText.value"
        :is-busy="call.isBusy.value"
        @change="restoring.typing.setRestoreText"
        @restore="restoring.restoreWords"
        @cancel="actions.cancel"
      />

      <IdentityRestoreSheets
        v-else-if="step === 'restoreSheets'"
        :texts="fields.texts.value"
        :is-busy="call.isBusy.value"
        @change="restoring.setSheetText"
        @restore="restoring.restoreSheets"
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
