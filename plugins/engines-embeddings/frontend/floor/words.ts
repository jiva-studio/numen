/**
 * Strings of the search relevance panel: the threshold, its source, the noise measured on the
 * vault, and the query check.
 */
export const CALIBRATION_MIN_NOTES = 20

// Stryker disable StringLiteral: interface copy
export const WORDS = {
  title: 'Search relevance',
  modeLabel: 'Relevance threshold',
  modeAuto: 'Auto',
  modeManual: 'Manual',
  thresholdLabel: 'Threshold',
  recalculate: 'Recalculate',
  recalculating: 'Calculating…',
  sourceLabel: 'Source',
  sourceCalibrated: 'Calibrated on your vault',
  sourceModel: 'Model profile',
  sourceDefault: 'Default',
  sourceFile: 'Settings file',
  sourceManual: 'Set by you',
  fileNote: 'The settings file sets this threshold for every model. Edit the file to change it.',
  noiseLabel: 'Noise (unrelated pairs)',
  noiseNone: 'Not measured yet',
  histogramAlt: 'Distribution of similarities between unrelated texts',
  histogramMarker: 'Threshold',
  tooFew: `Not enough notes yet (needs ≥ ${CALIBRATION_MIN_NOTES})`,
  pending: 'Calibration pending',
  sliderLabel: 'Manual threshold',
  sliderHint: 'Applies to the current model only. The marker shows the automatic value.',
  automaticMarker: 'Automatic value',
  probeLabel: 'Check a query',
  probePlaceholder: 'Type a query',
  probeButton: 'Check',
  probing: 'Checking…',
  loading: 'Loading…',
  retry: 'Try again',
  loadFailed: 'The relevance settings could not be loaded.',
  actionFailed: 'The change could not be applied.',
  probeEmpty: 'The vault has no indexed passages yet.',
}

export const SLIDER_MIN = 0.05
export const SLIDER_MAX = 0.95
export const SLIDER_STEP = 0.01
