/**
 * Invented saved searches over harbor logbooks, read from the fixtures of testdata/api and varied
 * for the tests and the stories.
 */
import response from '../../../testdata/api/GetResults.response.json'
import type { GetResultsResponse, ResultsChange, SearchResult } from '../api/types'

/** The path of the saved search note every fixture answers for. */
export const NOTE_PATH = 'Harbor/Harbor master - winter storms.md'

/** The cached results of the note: three results, computed at 09:30 UTC on 14 November 2026. */
export const HARBOR = response as GetResultsResponse

/** The first notice of the watch: the record as it is. */
export const COMPUTED: ResultsChange = { computedAt: HARBOR.computedAt, isRunning: false }

/** A fourth result, which the next run finds. */
export const LIGHTHOUSE_RESULT: SearchResult = {
  path: 'Logbooks/1912.pdf',
  title: '1912',
  place: { page: 3, pageCount: 180 },
  text: 'The keeper lit the lamp early; the harbor master had the boats hauled up.',
  start: 5120,
  length: 940,
  line: 41,
  uri: 'numen:Logbooks/1912.pdf?start=5120&length=940',
  score: 0.77,
  chunkCount: 1,
}

/** The results after the next run: the three of HARBOR and one more. */
export const HARBOR_REFRESHED: GetResultsResponse = {
  ...HARBOR,
  computedAt: '2026-11-14T10:05:00Z',
  results: [...HARBOR.results, LIGHTHOUSE_RESULT],
}

/** A note never computed yet. */
export const NEVER_COMPUTED: GetResultsResponse = {
  ...HARBOR,
  computedAt: '',
  order: '',
  candidateCount: 0,
  results: [],
}

/** A run that found nothing. */
export const NOTHING_FOUND: GetResultsResponse = { ...HARBOR, candidateCount: 0, results: [] }

/** A note whose query does not compile: the problem points into the file, and there is no result. */
export const MISSPELLED: GetResultsResponse = {
  ...HARBOR,
  query: 'kind in ["note", "pdf"]\n&& meanin("the harbor master in a winter storm")',
  computedAt: '',
  order: '',
  results: [],
  errors: [{ line: 9, column: 4, message: "undeclared reference to 'meanin'" }],
}

/** The warning of a run whose reranker did not answer. */
export const FUSED_WARNING = 'the reranking service did not answer; the results are in fused order'

/** A run that kept its results in fused order, with a warning. */
export const FUSED: GetResultsResponse = { ...HARBOR, order: 'fused', warnings: [FUSED_WARNING] }

/** The reason of a failed last run. */
export const FAILURE_REASON = 'the reranking service did not answer'

/** A last run that failed at 09:41 UTC; the results are those of the good run at 09:30. */
export const FAILED_LAST_RUN: GetResultsResponse = {
  ...HARBOR,
  failedAt: '2026-11-14T09:41:00Z',
  failure: FAILURE_REASON,
}

/** A run in progress over the results of 09:30. */
export const RUNNING: GetResultsResponse = { ...HARBOR, isRunning: true }

/** Results a run will replace. */
export const STALE: GetResultsResponse = { ...HARBOR, isStale: true }
