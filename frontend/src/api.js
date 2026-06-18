const API_BASE = import.meta.env.VITE_API_BASE || 'http://localhost:8080/api'
const TOKEN_KEY = 'opscore.token'

export function getToken() {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token) {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY)
}

export async function api(path, options = {}) {
  const token = getToken()
  const response = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(options.headers || {})
    }
  })
  const payload = await response.json().catch(() => ({}))
  if (!response.ok) {
    throw new Error(payload.error || '请求失败')
  }
  return payload
}

export async function login(username, password) {
  const payload = await api('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username, password })
  })
  setToken(payload.token)
  return payload
}

