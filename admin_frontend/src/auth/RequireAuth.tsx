import type {ReactNode} from 'react'
import {Navigate} from 'react-router-dom'
import {Spin} from 'antd'
import {useAuth} from './AuthContext'

export function RequireAuth({children}: { children: ReactNode }) {
    const {admin, loading} = useAuth()

    if (loading) {
        return (
            <div style={{display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh'}}>
                <Spin size="large"/>
            </div>
        )
    }
    if (!admin) {
        return <Navigate to="/start?to=admin" replace/>
    }
    return <>{children}</>
}
