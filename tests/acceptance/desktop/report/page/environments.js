/* global createElement, createStatusDot, state, writeDetail */
/* exported pickRun, createEnvironments */
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
    document.querySelector('.environments [aria-pressed="true"]')?.focus()
  })
  return group
}
