import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import Login from './pages/Login';
import Queue from './pages/Queue';
import CaseDetail from './pages/CaseDetail';
import { isAuthenticated } from './api/client';
import './App.css';

function PrivateRoute({ children }: { children: React.ReactNode }) {
  return isAuthenticated() ? children : <Navigate to="/login" replace />;
}

function App() {
  return (
    <BrowserRouter>
      <div className="app">
        <header className="app-header">
          <h1>Lantern Moderation Console</h1>
        </header>
        <main>
          <Routes>
            <Route path="/login" element={<Login />} />
            <Route
              path="/"
              element={
                <PrivateRoute>
                  <Queue />
                </PrivateRoute>
              }
            />
            <Route
              path="/cases/:caseId"
              element={
                <PrivateRoute>
                  <CaseDetail />
                </PrivateRoute>
              }
            />
          </Routes>
        </main>
      </div>
    </BrowserRouter>
  );
}

export default App;
