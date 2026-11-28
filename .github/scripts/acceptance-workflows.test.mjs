import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import { load } from 'js-yaml'

function readWorkflow(name) {
  return load(readFileSync(new URL('../workflows/' + name + '.yml', import.meta.url), 'utf8'))
}

test('acceptance groups and E2E use the same nightly build artifact', () => {
  const { jobs } = readWorkflow('reports')
  assert.equal(jobs.build.uses, './.github/workflows/desktop-e2e-build.yml')
  for (const name of ['acceptance', 'live-sync', 'stand', 'e2e']) {
    assert.deepEqual(jobs[name].needs, ['build'])
    assert.equal(jobs[name].with['artifact-name'], '${{ needs.build.outputs.artifact-name }}')
  }
  for (const name of ['acceptance', 'live-sync', 'stand']) {
    assert.ok(jobs.report.needs.includes(name))
    assert.ok(jobs.summary.needs.includes(name))
  }
})

test('nightly release requires success from offline acceptance, E2E, live and stand', () => {
  const { jobs, on } = readWorkflow('reports')
  const condition = readWorkflow('nightly').jobs.release.if
  for (const [result, job] of Object.entries({ linux: 'acceptance', e2e: 'e2e', live: 'live-sync', stand: 'stand' })) {
    assert.match(condition, new RegExp("&& needs\\.tests\\.outputs\\." + result + " == 'success'"))
    assert.equal(on.workflow_call.outputs[result].value, '${{ jobs.summary.outputs.' + result + ' }}')
    assert.equal(jobs.summary.outputs[result], '${{ needs.' + job + '.result }}')
  }
})

test('real acceptance jobs download the build and publish separate reports on failure', () => {
  for (const [workflow, job] of [['live', 'live'], ['stand', 'stand']]) {
    const { jobs } = readWorkflow(workflow)
    const steps = jobs[job].steps
    assert.ok(steps.some((step) => step.uses === './.github/actions/fetch-e2e-build'))
    const blob = steps.find((step) => step.with?.name === 'acceptance-blobs-' + workflow + '-linux')
    assert.match(blob?.if ?? '', /!cancelled\(\)/u)
  }
})
