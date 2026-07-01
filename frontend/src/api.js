const API_BASE = import.meta.env?.VITE_API_BASE || '/api'
const TOKEN_KEY = 'opscore.token'
let sessionExpiredHandler = null

export class ApiError extends Error {
  constructor(message, status) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

export function setSessionExpiredHandler(handler) {
  sessionExpiredHandler = typeof handler === 'function' ? handler : null
}

export function getToken() {
  return sessionStorage.getItem(TOKEN_KEY)
}

export function setToken(token) {
  localStorage.removeItem(TOKEN_KEY)
  sessionStorage.setItem(TOKEN_KEY, token)
}

export function clearToken() {
  sessionStorage.removeItem(TOKEN_KEY)
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
    if (response.status === 401 && token) {
      clearToken()
      sessionExpiredHandler?.()
    }
    throw new ApiError(payload.error || '请求失败', response.status)
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
