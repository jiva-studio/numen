<script setup lang="ts">
/**
 * The identity of this device: its fingerprints, devices and recovery kit when it has one, and
 * otherwise the way to make one (a dialog of two steps: write down twelve words, check three of them)
 * or to restore one in a dialog from its words or from two sheets of a kit.
 */
import { computed, ref } from 'vue'
import { Keyboard, KeyRound, Plus, QrCode, RotateCcw, ScanQrCode } from '@lucide/vue'
import { CardActions, IslandCard, Toast, type CardActionItem } from '@numen/ui'
import type { InstallationIdentity } from '../../types'
import { useIdentitySection } from '../../model/useIdentitySection'
import { useRecoveryKit } from '../../model/useRecoveryKit'
import { WORDS as words } from '../../words'
import { JoinDialog } from '@/features/identity-link'
import { KeyFileHost } from '@/features/key-file'
import IdentityDetails from './IdentityDetails.vue'
import RestoreSheetsDialog from './RestoreSheetsDialog.vue'
import RestoreTypedDialog from './RestoreTypedDialog.vue'
import RestoreWordsDialog from './RestoreWordsDialog.vue'
import IdentitySetup from './IdentitySetup.vue'
import RecoveryKitDialog from './RecoveryKitDialog.vue'
import RecoveryKitRow from './RecoveryKitRow.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ identity: InstallationIdentity }>()

/* --------------------------------- State ---------------------------------- */
const kit = useRecoveryKit(() => props.identity)
const section = useIdentitySection(() => props.identity, kit.actions.hold)
const { identity: current, step, setup, call, confirmation, restore, deletion, actions } = section
const { words: restoring, typed: typing, scanning } = restore
const { stage, recoveryWords } = setup
const isJoining = ref(false)

const noneActions = computed<readonly CardActionItem[]>(() => {
  const isDisabled = call.isBusy.value
  return [
    {
      id: 'restore',
      label: words.identityRestore,
      icon: RotateCcw,
      isDisabled,
      choices: [
        {
          id: 'restore-words',
          label: words.identityRestoreFromWords,
          detail: words.identityRestoreFromWordsDetail,
          icon: KeyRound,
        },
        {
          id: 'restore-sheets',
          label: words.identityRestoreKit,
          detail: words.identityRestoreScanDetail,
          icon: ScanQrCode,
        },
        {
          id: 'restore-typed',
          label: words.identityRestoreKit,
          detail: words.identityRestoreTypeDetail,
          icon: Keyboard,
        },
      ],
    },
    { id: 'join', label: words.identityJoin, icon: QrCode, isDisabled },
    { id: 'create', label: words.identityCreate, icon: Plus, isDisabled, isPrimary: true, isLabelKept: true },
  ]
})

function onChooseNone(id: string): void {
  if (id === 'restore-words' || id === 'restore-sheets' || id === 'restore-typed') actions.startRestore(id)
  else if (id === 'join') isJoining.value = true
  else actions.startSetup()
}
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
        :is-busy="call.isBusy.value || kit.call.isBusy.value"
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
      <CardActions
        :items="noneActions"
        :more-label="words.moreActions"
        @choose="onChooseNone"
      />
    </template>
    <div class="identity-section__body">
      <p class="identity-section__detail">{{ words.identityNoneDetail }}</p>
      <p class="identity-section__detail">{{ words.identityNoneWords }}</p>

      <JoinDialog
        v-if="isJoining && step === 'idle'"
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

    <RestoreWordsDialog
      v-if="step === 'restore-words'"
      :text="restoring.text.value"
      :reading="restoring.reading.value"
      :problem="call.problem.value"
      :is-busy="call.isBusy.value"
      @change="restoring.setText"
      @restore="actions.restore"
      @cancel="actions.cancel"
    />

    <RestoreSheetsDialog
      v-if="step === 'restore-sheets'"
      :slots="scanning.slots.value"
      :note="scanning.note.value"
      :problem="call.problem.value"
      :is-busy="call.isBusy.value"
      @scanned="scanning.scan"
      @unavailable="actions.startRestore('restore-typed')"
      @cancel="actions.cancel"
    />

    <RestoreTypedDialog
      v-if="step === 'restore-typed'"
      :sheets="typing.sheets.value"
      :reading="typing.reading.value"
      :problem="call.problem.value"
      :is-busy="call.isBusy.value"
      @change="typing.setSheet"
      @restore="actions.restoreTyped"
      @cancel="actions.cancel"
    />
  </IslandCard>

  <KeyFileHost :key-file="identity.keyFile" />
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
