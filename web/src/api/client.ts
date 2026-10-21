export interface Application {
  id: number
  name: string
  repository_url: string
  branch: string
  created_at: string
  updated_at: string
}

export interface Deployment {
  id: number
  application_id: number
  status: string
  commit_sha?: string
  error_message?: string
  created_at: string
  updated_at: string
}

export interface DeploymentHistory {
  deployments: Deployment[]
  limit: number
  offset: number
}

export interface CreateApplicationInput {
  name: string
  repository_url: string
  branch: string
}

export interface ApiFailure {
  error?: {
    code?: string
    message?: string
  }
}

const apiBase = (import.meta.env.VITE_API_BASE_URL || '/api/v1').replace(/\/+$/, '')

export class ApiError extends Error {
  readonly status: number
  readonly code?: string

  constructor(message: string, status: number, code?: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

async function request<T>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  const response = await fetch(`${apiBase}${path}`, {
    ...init,
    headers: {
      Accept: 'application/json',
      ...(init?.body ? { 'Content-Type': 'application/json' } : {}),
      ...init?.headers,
    },
  })
  const body = await response.text()
  let data: unknown

  if (body) {
    try {
      data = JSON.parse(body)
    } catch {
      throw new ApiError('The server returned an unreadable response.', response.status)
    }
  }

  if (!response.ok) {
    const failure = data as ApiFailure | undefined
    throw new ApiError(
      failure?.error?.message || `Request failed with status ${response.status}.`,
      response.status,
      failure?.error?.code,
    )
  }

  if (data === undefined) {
    throw new ApiError('The server returned an empty response.', response.status)
  }

  return data as T
}

export const api = {
  createApplication(input: CreateApplicationInput): Promise<Application> {
    return request<Application>('/applications', {
      method: 'POST',
      body: JSON.stringify(input),
    })
  },

  getApplication(id: number, signal?: AbortSignal): Promise<Application> {
    return request<Application>(`/applications/${id}`, { signal })
  },

  createDeployment(
    applicationId: number,
    commitSHA: string,
  ): Promise<Deployment> {
    return request<Deployment>(`/applications/${applicationId}/deployments`, {
      method: 'POST',
      body: JSON.stringify(commitSHA ? { commit_sha: commitSHA } : {}),
    })
  },

  getDeployment(id: number, signal?: AbortSignal): Promise<Deployment> {
    return request<Deployment>(`/deployments/${id}`, { signal })
  },

  listDeployments(
    applicationId: number,
    limit = 20,
    offset = 0,
    signal?: AbortSignal,
  ): Promise<DeploymentHistory> {
    const query = new URLSearchParams({
      limit: String(limit),
      offset: String(offset),
    })
    return request<DeploymentHistory>(
      `/applications/${applicationId}/deployments?${query.toString()}`,
      { signal },
    )
  },
}
