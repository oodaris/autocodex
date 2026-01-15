import { Link, Outlet, Route, Routes } from 'react-router-dom'
import Dashboard from './pages/Dashboard'
import RunDetail from './pages/RunDetail'
import './App.css'

function Layout() {
  return (
    <div className="app">
      <div className="shell">
        <header className="topbar">
          <div className="brand">
            <Link to="/">Autocodex</Link>
            <span>Control Deck</span>
          </div>
          <nav className="nav">
            <Link to="/">Runs</Link>
            <a href="https://github.com/oodaris/autocodex" target="_blank" rel="noreferrer">
              GitHub
            </a>
          </nav>
        </header>
        <Outlet />
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
        <Route path="runs/:runId" element={<RunDetail />} />
        <Route path="*" element={<NotFound />} />
      </Route>
    </Routes>
  )
}
