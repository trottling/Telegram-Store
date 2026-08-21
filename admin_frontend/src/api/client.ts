// Обёртка над fetch: добавляет токен к каждому запросу, на 401 чистит его и уводит на /start.
import type {ErrorResponse} from '../types/api'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080'
const TOKEN_STORAGE_KEY = 'tg_store_admin_token'

export function getToken(): string | null {
    return localStorage.getItem(TOKEN_STORAGE_KEY)
}

export function setToken(token: string) {
    localStorage.setItem(TOKEN_STORAGE_KEY, token)
}

export function clearToken() {
    localStorage.removeItem(TOKEN_STORAGE_KEY)
}

export class ApiError extends Error {
    status: number
    code: string

    constructor(status: number, body: ErrorResponse) {
        super(body.message || `request failed with status ${status}`)
        this.status = status
        this.code = body.code
    }
}

interface RequestOptions {
    method?: 'GET' | 'POST' | 'PUT' | 'DELETE'
    body?: unknown
    query?: Record<string, string | number | undefined | null>
}

function buildQuery(query?: RequestOptions['query']): string {
    if (!query) return ''
    const params = new URLSearchParams()
    for (const [key, value] of Object.entries(query)) {
        if (value !== undefined && value !== null && value !== '') {
            params.set(key, String(value))
        }
    }
    const qs = params.toString()
    return qs ? `?${qs}` : ''
}

export async function apiRequest<T>(path: string, options: RequestOptions = {}): Promise<T> {
    const token = getToken()
    const headers: Record<string, string> = {}
    if (token) headers['Authorization'] = `Bearer ${token}`
    if (options.body !== undefined) headers['Content-Type'] = 'application/json'

    const res = await fetch(`${API_BASE_URL}${path}${buildQuery(options.query)}`, {
        method: options.method ?? 'GET',
        headers,
        body: options.body !== undefined ? JSON.stringify(options.body) : undefined,
    })

    if (res.status === 401) {
        clearToken()
        if (window.location.pathname !== '/start') {
            window.location.href = '/start?to=admin'
        }
        throw new ApiError(401, {code: 'unauthorized', message: 'session expired'})
    }

    if (res.status === 204) {
        return undefined as T
    }

    const data = await res.json().catch(() => ({}))
    if (!res.ok) {
        throw new ApiError(res.status, data as ErrorResponse)
    }
    return data as T
}
