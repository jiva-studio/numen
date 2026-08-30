/** What the settings tab says: the groups it draws, and what each setting is. */
export const WORDS = {
  settings: 'Settings',
  /** Where the settings stand, said once at the head of the page. */
  file: 'One file, numen.json, in this machine’s configuration folder.',
  /** The groups, in the order they are drawn. */
  window: 'The window',
  naming: 'What a note is called',
  review: 'A day of review',
  indexing: 'Reading the vault',
  agent: 'Which agent answers',
  /** The window's own settings. */
  theme: 'Theme',
  themeDetail: 'The stylesheet the window wears.',
  shipped: 'Ships with numen',
  owned: 'Yours',
  mode: 'Light or dark',
  modeDetail: 'Which half of a colour pair every token is read as.',
  pinned: 'The theme worn declares light and dark itself.',
  system: 'System',
  light: 'Light',
  dark: 'Dark',
  interfaceScale: 'Interface size',
  interfaceScaleDetail: 'How large the chrome, the controls and the type in them are drawn. 1 is as designed.',
  textScale: 'Reading size',
  textScaleDetail: 'How large the text a person reads is set. 1 is as designed.',
  hanging: 'Parts under a node',
  hangingDetail: 'Whether a node hangs the headings of its note under its box.',
  parts: 'How many parts',
  partsDetail: 'How many of them stand under a node at once.',
  syncing: 'Sync title and filename',
  syncingDetail: 'Whether renaming either of the two brings the other into line.',
  dayStarts: 'A day begins at',
  dayStartsDetail:
    'The hour a day of review begins at, on the clock on the wall. An answer given before it is written into the day before.',
  embedding: 'What a vector is, and where it is made',
  embeddingDetail: 'The model the vault is indexed by, and the two stations that make its vectors.',
  proofreading: 'Putting a reading right',
  proofreadingDetail: 'What corrects a reading of a scanned page. Naming nothing here proofreads nothing.',
  recognition: 'Reading a scanned document',
  recognitionDetail: 'How a scanned page is read.',
  agentUse: 'The agent',
  agentUseDetail: 'Which agent answers in the panel, what it may reach, and whether the tools go on a port.',
  /**
   * A setting this window neither reads nor writes. It stands in the file, and
   * the file is where it is changed.
   */
  inTheFile: 'Set in numen.json',
}
