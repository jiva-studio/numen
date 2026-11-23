<script setup lang="ts">
/**
 * The identity of this device: its fingerprints and the recovery kit when it has one, and otherwise
 * the way to make one (write down twelve words, confirm three of them) or to restore one from its
 * words or from two sheets of a kit.
 */
import { Button } from '@numen/ui'
import type { Printer } from '@/shared/api/print'
import type { InstallationIdentity } from '../../types'
import { useIdentityCall } from '../../model/useIdentityCall'
import { useIdentitySection } from '../../model/useIdentitySection'
import { useRecoveryKit } from '../../model/useRecoveryKit'
import { useSheetFields } from '../../model/useSheetFields'
import { WORDS as words } from '../../words'
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
const kit = useRecoveryKit(() => props.identity, () => props.printer, call, fields)
const section = useIdentitySection(() => props.identity, call, fields, kit.actions.showAfterSetup)
const {
  identity: current,
  step,
  recoveryWords,
  confirmation,
  restoring,
  deletion,
  actions,
} = section
</script>

<template>
  <section class="settings-card identity-section" :aria-label="words.identitySection">
    <IdentityDetails
      v-if="current"
      :info="current"
      :is-busy="call.isBusy.value"
      :is-asking-delete="deletion.isAskingDelete.value"
      @ask="deletion.askDelete"
      @confirm="deletion.confirmDelete"
      @cancel="deletion.cancelDelete"
    />

    <template v-else>
      <div class="identity-section__empty">
        <p class="identity-section__title">{{ words.identityNone }}</p>
        <p v-if="step === 'idle'" class="identity-section__detail">
          {{ words.identityNoneDetail }}
        </p>
      </div>

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
          @click="actions.startRestore('restore')"
        >
          {{ words.identityRestore }}
        </Button>
        <Button
          type="button"
          variant="outline"
          :disabled="call.isBusy.value"
          @click="actions.startRestore('restoreSheets')"
        >
          {{ words.identityRestoreSheets }}
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
        v-else-if="step === 'restore'"
        :text="restoring.typing.restoreText.value"
        :is-busy="call.isBusy.value"
        @change="restoring.typing.setRestoreText"
        @restore="restoring.restoreWords"
        @cancel="actions.cancel"
      />

      <IdentityRestoreSheets
        v-else
        :texts="fields.texts.value"
        :is-busy="call.isBusy.value"
        @change="restoring.setSheetText"
        @restore="restoring.restoreSheets"
        @cancel="actions.cancel"
      />
    </template>

    <RecoveryKitSection
      v-if="current"
      :model="kit"
      :texts="fields.texts.value"
      :is-busy="call.isBusy.value"
    />

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
