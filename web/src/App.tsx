import { useEffect, useState } from 'react'
import { Link, Outlet, Route, Routes, useParams } from 'react-router-dom'
import { apiAuth } from './api/client'
import Dashboard from './pages/Dashboard'
import Hub from './pages/Hub'
import MemoryDocs from './pages/MemoryDocs'
import RunDetail from './pages/RunDetail'
import Terminal from './pages/Terminal'
import './App.css'

function Layout() {
  const { workspaceId } = useParams()
  const [token, setToken] = useState(apiAuth.getToken())

  useEffect(() => {
    apiAuth.setToken(token)
  }, [token])

  const runsLink = workspaceId ? `/hub/${workspaceId}` : '/'
  const memoryLink = workspaceId ? `/hub/${workspaceId}/memory` : '/memory'
  const terminalLink = workspaceId ? `/hub/${workspaceId}/terminal` : '/terminal'

  return (
    <div className="app">
      <a className="skip-link" href="#main-content">
        Skip to content
      </a>
      <div className="shell">
        <header className="topbar">
          <div className="brand">
            <Link to="/">autocodex</Link>
            <span>Control Deck</span>
          </div>
          <nav className="nav" aria-label="Primary">
            <Link to={runsLink}>Runs</Link>
            <Link to={memoryLink}>Memory</Link>
            <Link to={terminalLink}>Terminal</Link>
            <Link to="/hub">Hub</Link>
            <a href="https://github.com/oodaris/autocodex" target="_blank" rel="noreferrer">
              GitHub
            </a>
          </nav>
          <form
            className="token"
            aria-label="API token"
            onSubmit={(event) => event.preventDefault()}
          >
            <label htmlFor="api-token">API token</label>
            <input
              className="visually-hidden"
              type="text"
              name="username"
              aria-label="Username"
              autoComplete="username"
              tabIndex={-1}
            />
            <input
              id="api-token"
              type="password"
              name="api-token"
              autoComplete="new-password"
              placeholder="optional"
              className="control-input token-input"
              value={token}
              onChange={(event) => setToken(event.target.value)}
            />
          </form>
        </header>
        <main id="main-content" className="content">
          <Outlet />
        </main>
      </div>
    </div>
  )
}

function NotFound() {
  return (
    <div className="page">
      <div className="panel">
        <h2>Page not found</h2>
        <p>The page you requested does not exist.</p>
        <Link className="link" to="/">
          Go back to the dashboard
        </Link>
      </div>
    </div>
  )
}

export default function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route index element={<Dashboard />} />
        <Route path="memory" element={<MemoryDocs />} />
        <Route path="terminal" element={<Terminal />} />
        <Route path="hub" element={<Hub />} />
        <Route path="hub/:workspaceId" element={<Dashboard />} />
        <Route path="hub/:workspaceId/memory" element={<MemoryDocs />} />
        <Route path="hub/:workspaceId/terminal" element={<Terminal />} />
        <Route path="runs/:runId" element={<RunDetail />} />
        <Route path="hub/:workspaceId/runs/:runId" element={<RunDetail />} />
        <Route path="*" element={<NotFound />} />
      </Route>
    </Routes>
  )
}
