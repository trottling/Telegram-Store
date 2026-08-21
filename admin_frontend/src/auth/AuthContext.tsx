import {createContext, type ReactNode, useCallback, useContext, useEffect, useState} from 'react'
import {exchangeInitData, getMe, logoutRequest} from '../api/resources'
import {clearToken, getToken, setToken} from '../api/client'
import type {AdminUser} from '../types/api'

interface AuthContextValue {
    admin: AdminUser | null
    loading: boolean
    // login меняет initData от Telegram на токен и подтверждает через GET /api/auth/me.
    login: (initData: string) => Promise<void>
    logout: () => void
    refresh: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined)

export function AuthProvider({children}: { children: ReactNode }) {
    const [admin, setAdmin] = useState<AdminUser | null>(null)
    const [loading, setLoading] = useState(true)

    const refresh = useCallback(async () => {
        if (!getToken()) {
            setAdmin(null)
            setLoading(false)
            return
        }
        try {
            const me = await getMe()
            setAdmin(me)
        } catch {
            setAdmin(null)
        } finally {
            setLoading(false)
        }
    }, [])

    useEffect(() => {
        refresh()
    }, [refresh])

    const login = useCallback(async (initData: string) => {
        const {token} = await exchangeInitData(initData) // кидает ApiError(401), если initData невалидна или не админ
        setToken(token)
        const me = await getMe()
        setAdmin(me)
    }, [])

    const logout = useCallback(() => {
        // Best-effort — сессия чистится локально в любом случае.
        logoutRequest().catch(() => undefined)
        clearToken()
        setAdmin(null)
    }, [])

    return (
        <AuthContext.Provider value={{admin, loading, login, logout, refresh}}>{children}</AuthContext.Provider>
    )
}

export function useAuth(): AuthContextValue {
    const ctx = useContext(AuthContext)
    if (!ctx) throw new Error('useAuth must be used within AuthProvider')
    return ctx
}
