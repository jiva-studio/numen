/**
 * The state of a masked field a secret value is entered into. Without a submit function the value
 * is emitted and the field cleared. With one, the field keeps the value until submit resolves
 * true, sends no second value while one is in flight, and sends a failed value again only when
 * asked to commit it again.
 */
import { computed, ref } from 'vue'
import { reportFailure, runInBackground } from '@numen/ui'

export interface SecretValueEntryDeps {
  readonly getSubmit: () => ((value: string) => Promise<boolean>) | undefined
  readonly emitCommit: (value: string) => void
}

export function useSecretValueEntry(deps: SecretValueEntryDeps) {
  const entry = ref('')
  const isSending = ref(false)
  const failedValue = ref<string | null>(null)
  const hasFailed = computed(() => failedValue.value !== null)

  async function submitValue(
    submit: (value: string) => Promise<boolean>,
    value: string,
  ): Promise<void> {
    if (isSending.value || value === failedValue.value) return
    isSending.value = true
    try {
      if (await submit(value)) entry.value = ''
      else failedValue.value = value
    } finally {
      isSending.value = false
    }
  }

  function commit(): void {
    const value = entry.value.trim()
    if (value === '') return
    const submit = deps.getSubmit()
    if (submit) {
      runInBackground(submitValue(submit, value), reportFailure)
      return
    }
    entry.value = ''
    deps.emitCommit(value)
  }

  function commitAgain(): void {
    failedValue.value = null
    commit()
  }

  function clearFailure(): void {
    failedValue.value = null
  }

  return { entry, hasFailed, commit, commitAgain, clearFailure }
}
