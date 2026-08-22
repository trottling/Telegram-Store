import {useEffect, useRef, useState} from 'react'
import {useNavigate, useSearchParams} from 'react-router-dom'
import {useRawInitData} from '@tma.js/sdk-react'
import {Alert, Spin, Typography} from 'antd'
import {useAuth} from '../auth/AuthContext'

const {Text} = Typography

// StartPage — единственная точка входа для обоих inline-кнопок из /admin
// (?to=admin и ?to=stats, см. bot/keyboards.buildAdminKb). Меняет initData,
// которую Telegram кладёт в URL при запуске Mini App, на сессионный токен, и
// сама решает, куда открывать дальше:
//   to=admin — переход внутри этого же SPA (токен уже в localStorage);
//   to=stats — полная навигация на /stats/: Grafana спрятана за Caddy'шным
//     forward_auth, который проверяет ту же сессионную cookie, что только что
//     выставил /api/auth/exchange (см. handlers.Exchange, setSessionCookie) —
//     отдельного моста для Grafana не нужно, она просто позади того же токена.
// Вне Telegram initData нет вообще — тут не форма логина, а тупик с понятной
// причиной; отдельной страницы с ручным вводом кода больше не существует.
//
// to=admin пропускает обмен, если сессия уже валидна (AuthProvider уже
// подтвердил токен из localStorage через GET /api/auth/me) — иначе каждый
// повторный тап по кнопке в Telegram бил бы по POST /api/auth/exchange, а он
// под RateLimitExchange (10/мин на IP): десяток открытий панели подряд —
// и админ на минуту заперт снаружи собственной панели. to=stats всегда бьёт
// в обмен заново — session-cookie отдельно от localStorage-токена, и дешевле
// гарантированно освежить её, чем рисковать редиректной петлёй со /stats.
export function StartPage() {
    const [params] = useSearchParams()
    const to = params.get('to') === 'stats' ? 'stats' : 'admin'
    const initData = useRawInitData()
    const {admin, loading, login} = useAuth()
    const navigate = useNavigate()
    const [error, setError] = useState<string | null>(null)
    const started = useRef(false)

    useEffect(() => {
        if (loading || started.current) return
        started.current = true

        if (to === 'admin' && admin) {
            navigate('/categories', {replace: true})
            return
        }

        if (!initData) {
            setError('Откройте эту ссылку из Telegram — отправьте боту /admin.')
            return
        }

        login(initData)
            .then(() => {
                if (to === 'stats') {
                    window.location.href = '/stats/'
                } else {
                    navigate('/categories', {replace: true})
                }
            })
            .catch(() => setError('Не удалось войти: доступ только для администраторов.'))
    }, [loading, admin, initData, login, navigate, to])

    if (error) {
        return (
            <div style={{display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh', background: '#f0f2f5', padding: 24}}>
                <Alert type="error" message={error} showIcon style={{maxWidth: 420}}/>
            </div>
        )
    }

    return (
        <div style={{display: 'flex', flexDirection: 'column', gap: 16, justifyContent: 'center', alignItems: 'center', height: '100vh'}}>
            <Spin size="large"/>
            <Text type="secondary">Входим…</Text>
        </div>
    )
}
