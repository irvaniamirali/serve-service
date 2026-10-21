import { afterEach, describe, expect, it, vi } from 'vitest'
import { api, ApiError } from './client'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('API client', () => {
  it('serializes application creation and uses the actual API route', async () => {
    const application = {
      id: 12,
      name: 'storefront',
      repository_url: 'https://github.com/acme/storefront.git',
      branch: 'main',
      created_at: '2026-10-21T10:00:00Z',
      updated_at: '2026-10-21T10:00:00Z',
    }
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(application), { status: 201 }),
    )
    vi.stubGlobal('fetch', fetchMock)

    await expect(api.createApplication({
      name: application.name,
      repository_url: application.repository_url,
      branch: application.branch,
    })).resolves.toEqual(application)

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/applications', {
      method: 'POST',
      body: JSON.stringify({
        name: application.name,
        repository_url: application.repository_url,
        branch: application.branch,
      }),
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json',
      },
    })
  })

  it('preserves the backend error message and code for non-success responses', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(
      new Response(JSON.stringify({
        error: {
          code: 'application_name_conflict',
          message: 'An application with this name already exists',
        },
      }), { status: 409 }),
    ))

    await expect(api.createApplication({
      name: 'duplicate',
      repository_url: 'https://github.com/acme/duplicate.git',
      branch: 'main',
    })).rejects.toMatchObject({
      name: 'ApiError',
      status: 409,
      code: 'application_name_conflict',
      message: 'An application with this name already exists',
    } satisfies Partial<ApiError>)
  })

  it('requests deployment history with its pagination parameters', async () => {
    const history = { deployments: [], limit: 20, offset: 0 }
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(history), { status: 200 }),
    )
    vi.stubGlobal('fetch', fetchMock)

    await expect(api.listDeployments(7)).resolves.toEqual(history)
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/applications/7/deployments?limit=20&offset=0',
      { headers: { Accept: 'application/json' } },
    )
  })

  it('queues a deployment at the supported application endpoint', async () => {
    const deployment = {
      id: 18,
      application_id: 7,
      status: 'queued',
      commit_sha: 'abcdef0123456789abcdef0123456789abcdef01',
      created_at: '2026-10-21T10:00:00Z',
      updated_at: '2026-10-21T10:00:00Z',
    }
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(deployment), { status: 201 }),
    )
    vi.stubGlobal('fetch', fetchMock)

    await expect(api.createDeployment(7, deployment.commit_sha)).resolves.toEqual(deployment)
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/applications/7/deployments', {
      method: 'POST',
      body: JSON.stringify({ commit_sha: deployment.commit_sha }),
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json',
      },
    })
  })
})
