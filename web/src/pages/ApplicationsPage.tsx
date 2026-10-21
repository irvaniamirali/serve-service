import { useState, type FormEvent } from 'react'
import { ArrowRight, Plus, Search } from 'lucide-react'
import { api, ApiError } from '@/api/client'
import { ApplicationDetails } from '@/components/ApplicationDetails'
import { Notice } from '@/components/Notice'
import { Button } from '@/components/ui/button'
import { Card, CardDescription, CardHeader, CardPanel, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { parsePositiveId } from '@/lib/parse-positive-id'

interface ApplicationsPageProps {
  applicationId: number | null
  onApplicationSelected: (id: number | null) => void
}

function errorMessage(error: unknown): string {
  return error instanceof ApiError
    ? error.message
    : 'Could not reach the API. Check the backend and try again.'
}

export function ApplicationsPage({
  applicationId,
  onApplicationSelected,
}: ApplicationsPageProps) {
  const [lookupId, setLookupId] = useState(applicationId ? String(applicationId) : '')
  const [name, setName] = useState('')
  const [repositoryURL, setRepositoryURL] = useState('')
  const [branch, setBranch] = useState('main')
  const [submitting, setSubmitting] = useState(false)
  const [lookupError, setLookupError] = useState('')
  const [createError, setCreateError] = useState('')

  function selectApplication(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setLookupError('')
    const id = parsePositiveId(lookupId)
    if (id === null) {
      setLookupError('Enter a positive application ID.')
      return
    }
    onApplicationSelected(id)
  }

  async function createApplication(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSubmitting(true)
    setCreateError('')
    try {
      const created = await api.createApplication({
        name,
        repository_url: repositoryURL,
        branch,
      })
      setLookupId(String(created.id))
      onApplicationSelected(created.id)
      setName('')
      setRepositoryURL('')
      setBranch('main')
    } catch (error) {
      setCreateError(errorMessage(error))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <section aria-labelledby="applications-title">
      <div className="page-heading">
        <div>
          <p className="eyebrow">SOURCE CONFIGURATION</p>
          <h1 id="applications-title">Applications</h1>
          <p className="page-subtitle">
            Register a repository or retrieve an application by its API-issued ID.
          </p>
        </div>
        {applicationId && (
          <span className="context-pill">Application #{applicationId} selected</span>
        )}
      </div>

      <div className="feature-grid">
        <Card className="content-card">
          <CardHeader>
            <div className="card-heading-icon"><Plus aria-hidden="true" size={16} /></div>
            <CardTitle>Register application</CardTitle>
            <CardDescription>Create a record with the repository and branch configuration.</CardDescription>
          </CardHeader>
          <CardPanel>
            <form className="form-stack" onSubmit={createApplication}>
              <label className="field" htmlFor="application-name">
                Application name
                <Input
                  autoComplete="off"
                  className="text-input"
                  id="application-name"
                  nativeInput
                  onChange={(event) => setName(event.target.value)}
                  placeholder="storefront"
                  required
                  value={name}
                />
              </label>
              <label className="field" htmlFor="repository-url">
                Repository URL
                <Input
                  autoComplete="url"
                  className="text-input"
                  id="repository-url"
                  nativeInput
                  onChange={(event) => setRepositoryURL(event.target.value)}
                  placeholder="https://github.com/acme/storefront.git"
                  required
                  type="url"
                  value={repositoryURL}
                />
              </label>
              <label className="field" htmlFor="application-branch">
                Branch
                <Input
                  autoComplete="off"
                  className="text-input"
                  id="application-branch"
                  nativeInput
                  onChange={(event) => setBranch(event.target.value)}
                  placeholder="main"
                  required
                  value={branch}
                />
              </label>
              {createError && <Notice kind="error">{createError}</Notice>}
              <Button className="form-submit" disabled={submitting} loading={submitting} type="submit">
                Register application <ArrowRight aria-hidden="true" size={15} />
              </Button>
            </form>
          </CardPanel>
        </Card>

        <Card className="content-card">
          <CardHeader>
            <div className="card-heading-icon"><Search aria-hidden="true" size={16} /></div>
            <CardTitle>Open an application</CardTitle>
            <CardDescription>Retrieve a saved application by its database ID.</CardDescription>
          </CardHeader>
          <CardPanel>
            <form className="lookup-form" onSubmit={selectApplication}>
              <label className="field field-grow" htmlFor="application-lookup">
                Application ID
                <Input
                  className="text-input mono-input"
                  id="application-lookup"
                  inputMode="numeric"
                  nativeInput
                  onChange={(event) => setLookupId(event.target.value)}
                  placeholder="e.g. 12"
                  value={lookupId}
                />
              </label>
              <Button className="lookup-submit" variant="outline" type="submit">Retrieve</Button>
            </form>
            {lookupError && <Notice kind="error">{lookupError}</Notice>}
            <div className="api-gap">
              <span className="gap-indicator" aria-hidden="true">i</span>
              <p>
                Application listing is unavailable: the API has no <code>GET /applications</code> endpoint.
                This view only shows records returned by create or retrieve requests.
              </p>
            </div>
          </CardPanel>
        </Card>
      </div>

      {applicationId !== null && (
        <div className="application-workspace">
          <ApplicationDetails
            applicationId={applicationId}
            onApplicationUnavailable={() => onApplicationSelected(null)}
          />
        </div>
      )}
    </section>
  )
}
