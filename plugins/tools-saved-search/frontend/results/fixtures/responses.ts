/**
 * Invented saved searches over harbor logbooks, read from the fixtures of testdata/api and varied
 * for the tests and the stories.
 */
import response from '../../../testdata/api/GetResults.response.json'
import type {
  GetResultsResponse,
  ListSearchFunctionsResponse,
  ResultsChange,
  SearchResult,
} from '../api/types'

/** The path of the saved search note every fixture answers for. */
export const NOTE_PATH = 'Harbor/Harbor master - winter storms.md'

/** The cached results of the note: two passages and a whole file, computed at 09:30 UTC on 14 November 2026. */
export const HARBOR = response as GetResultsResponse

/** The first notice of the watch: the record as it is. */
export const COMPUTED: ResultsChange = { computedAt: HARBOR.computedAt, isRunning: false }

/** A fourth result, which the next run finds. */
export const LIGHTHOUSE_RESULT: SearchResult = {
  path: 'Logbooks/1912.pdf',
  title: '1912',
  kind: 'pdf',
  place: { page: 3 },
  text: 'The keeper lit the lamp early; the harbor master had the boats hauled up.',
  start: 5120,
  length: 940,
  line: 41,
  uri: 'numen:Logbooks/1912.pdf?start=5120&length=940',
  score: 0.77,
}

/** A second passage of the logbook of 1911. */
export const SECOND_1911_RESULT: SearchResult = {
  path: 'Logbooks/1911.pdf',
  title: '1911',
  kind: 'pdf',
  place: { page: 52 },
  text: 'Ice on the quay; no boat went out for a week.',
  start: 190400,
  length: 820,
  line: 1460,
  uri: 'numen:Logbooks/1911.pdf?start=190400&length=820',
  score: 0.71,
}

/** The results after the next run: the three of HARBOR and one more. */
export const HARBOR_REFRESHED: GetResultsResponse = {
  ...HARBOR,
  computedAt: '2026-11-14T10:05:00Z',
  results: [...HARBOR.results, LIGHTHOUSE_RESULT],
}

/** The results of HARBOR with a second passage of the logbook of 1911. */
export const HARBOR_TWO_LOGBOOK_PASSAGES: GetResultsResponse = {
  ...HARBOR,
  results: [...HARBOR.results, SECOND_1911_RESULT],
}

/** A note never computed yet. */
export const NEVER_COMPUTED: GetResultsResponse = {
  ...HARBOR,
  computedAt: '',
  order: '',
  results: [],
}

/** A run that found nothing. */
export const NOTHING_FOUND: GetResultsResponse = { ...HARBOR, results: [] }

/** A note whose query the core rejects: the problem points into the query and the last good results stay. */
export const MISSPELLED: GetResultsResponse = {
  ...HARBOR,
  query: 'kind in ["note", "pdf"]\n&& meanin("the harbor master in a winter storm")',
  failedAt: '2026-11-14T09:41:00Z',
  failure: 'the query has problems',
  errors: [{ line: 2, column: 4, message: 'unknown function meanin (did you mean meaning?)' }],
}

/** A warning of the core about the run. */
export const WARNING = 'the query matched more than 10000 documents; the first 10000 were searched'

/** A run in fused order, with a warning. */
export const FUSED: GetResultsResponse = { ...HARBOR, order: 'fused', warnings: [WARNING] }

/** The reason of a failed last run. */
export const FAILURE_REASON = 'the core did not answer'

/** A last run that failed at 09:41 UTC; the results are those of the good run at 09:30. */
export const FAILED_LAST_RUN: GetResultsResponse = {
  ...HARBOR,
  failedAt: '2026-11-14T09:41:00Z',
  failure: FAILURE_REASON,
}

/** The error of a run whose reranking service did not answer. */
export const RERANKER_BUSY = 'no reranking service answered: Local reranker: busy'

/** A last run that failed at 09:41 UTC on a busy reranking service; the results are those of 09:30. */
export const FAILED_ON_RERANKER: GetResultsResponse = {
  ...HARBOR,
  failedAt: '2026-11-14T09:41:00Z',
  failure: RERANKER_BUSY,
  errors: [{ line: 0, column: 0, message: RERANKER_BUSY }],
}

/** A run in progress over the results of 09:30. */
export const RUNNING: GetResultsResponse = { ...HARBOR, isRunning: true }

/** Results a run will replace. */
export const STALE: GetResultsResponse = { ...HARBOR, isStale: true }

/** The catalog of the search language: three functions, one of them from a plugin, and two stages. */
export const FUNCTIONS: ListSearchFunctionsResponse = {
  functions: [
    {
      name: 'words',
      kind: 'function',
      signatures: ['words(text)'],
      description: 'Passages that hold every word of the text.',
      example: 'words("lighthouse keeper")',
    },
    {
      name: 'meaning',
      kind: 'function',
      signatures: ['meaning(text)'],
      description: 'Passages close in meaning to the text.',
      example: 'meaning("a night at sea")',
    },
    {
      name: 'tags',
      kind: 'function',
      signatures: ['tags(name)'],
      description: 'Notes that carry the tag.',
      example: 'tags("harbor")',
    },
    {
      name: 'top',
      kind: 'stage',
      signatures: ['.top(count)'],
      description: 'Keeps the first results.',
      example: 'words("gale").top(10)',
    },
    {
      name: 'rerank',
      kind: 'stage',
      signatures: ['.rerank(text)'],
      description: 'Orders the results by a reranking service.',
      example: 'words("gale").rerank("storm damage")',
    },
  ],
}
