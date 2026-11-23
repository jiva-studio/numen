export interface IdentitySetupProps {
  stage: 'words' | 'check'
  recoveryWords: readonly string[]
  positions: readonly number[]
  entered: Readonly<Record<number, string>>
  wrong: readonly number[]
  isComplete: boolean
  problem: string
  isBusy: boolean
}
