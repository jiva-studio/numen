/* global createElement, createStatusDot, state, writeDetail */
/* exported createRunsNote, pickRun, createEnvironments */
const getFailedLabels = (scenario) =>
  scenario.runs.filter((run) => run.status === 'failed').map((run) => run.environment.label)

const createRunsNote = (scenario) => {
  const failed = getFailedLabels(scenario)
  if (failed.length === 0 || failed.length === scenario.runs.length) return null
  const text = failed.join(', ')
  return createElement('span', { class: 'runs-note', title: `Fails on ${text}` }, [text])
}

const pickRun = (scenario) =>
  scenario.runs.find((run) => run.environment.id === state.environment) ??
  scenario.runs.find((run) => run.status === 'failed') ??
  scenario.runs[0]

const createEnvironmentButton = (run, current) =>
  createElement(
    'button',
    {
      type: 'button',
      'data-environment': run.environment.id,
      'aria-pressed': String(run === current),
    },
    [createStatusDot(run.status), run.environment.label],
  )

const createEnvironments = (scenario, current) => {
  const group = createElement(
    'div',
    { class: 'environments', role: 'group', 'aria-label': 'Environment' },
    scenario.runs.map((run) => createEnvironmentButton(run, current)),
  )
  group.addEventListener('click', (event) => {
    const button = event.target.closest('button')
    if (!button) return
    state.environment = button.dataset.environment
    writeDetail(scenario)
  })
  return group
}
