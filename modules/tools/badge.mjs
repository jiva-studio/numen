import { execFileSync } from 'node:child_process'

/**
 * Updates a badge JSON file in the `badges` branch of the current repository.
 * Uses `gh api` with GH_TOKEN / GITHUB_TOKEN.
 *
 * @param {string} filename - e.g. 'coverage.json' or 'mutation.json'
 * @param {object} payload - Shields.io endpoint object: { schemaVersion: 1, label, message, color }
 */
export const updateBadgeBranch = async (filename, payload) => {
  const contentStr = JSON.stringify(payload, null, 2) + '\n'
  const token = process.env.GH_TOKEN || process.env.GITHUB_TOKEN
  const repo = process.env.GITHUB_REPOSITORY || 'jiva-studio/numen'

  if (!token) {
    console.log(`[badge] No GH_TOKEN provided; skipping remote update for ${filename}.`)
    return
  }

  try {
    let sha = null
    try {
      const existing = JSON.parse(
        execFileSync('gh', ['api', `repos/${repo}/contents/${filename}?ref=badges`], {
          encoding: 'utf8',
          env: { ...process.env, GH_TOKEN: token },
        }),
      )
      sha = existing.sha
    } catch {
      // File does not exist yet on branch
    }

    const args = [
      'api',
      '--method',
      'PUT',
      `repos/${repo}/contents/${filename}`,
      '-f',
      `message=chore: update ${filename} badge`,
      '-f',
      `content=${Buffer.from(contentStr).toString('base64')}`,
      '-f',
      'branch=badges',
    ]
    if (sha) {
      args.push('-f', `sha=${sha}`)
    }

    execFileSync('gh', args, {
      encoding: 'utf8',
      env: { ...process.env, GH_TOKEN: token },
    })
    console.log(`[badge] Successfully updated ${filename} on branch badges.`)
  } catch (err) {
    console.error(`[badge] Failed to update ${filename} on branch badges:`, err.message)
    throw err
  }
}
