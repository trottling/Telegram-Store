import {useEffect, useRef, useState} from 'react'
import {useNavigate, useSearchParams} from 'react-router-dom'
import {useRawInitData} from '@tma.js/sdk-react'
import {Alert, Spin, Typography} from 'antd'
import {useAuth} from '../auth/AuthContext'

const {Text} = Typography

// dashboardUidPattern — тот же алфавит, что Grafana сама допускает для uid
// дашборда. dashboard берётся из query-параметра (см. ниже) и подставляется в
// window.location.href — без проверки формата туда можно было бы протащить
// "//evil.com" или "https://..." и получить открытый редирект.
const dashboardUidPattern = /^[a-zA-Z0-9_-]{1,40}$/

// StartPage — единственная точка входа для всех inline-кнопок из /admin
// (?to=admin и ?to=stats, см. bot/keyboards.buildAdminKb). Меняет initData,
// которую Telegram кладёт в URL при запуске Mini App, на сессионный токен, и
// сама решает, куда открывать дальше:
//   to=admin — переход внутри этого же SPA (токен уже в localStorage);
//   to=stats — полная навигация на конкретный дашборд Grafana (?dashboard=<uid>,
//     см. bot/keyboards.statsStartURL) или на /stats/ (список дашбордов), если
//     dashboard отсутствует или не прошёл dashboardUidPattern. Grafana спрятана
//     за Caddy'шным forward_auth, который проверяет ту же сессионную cookie,
//     что только что выставил /api/auth/exchange (см. handlers.Exchange,
//     setSessionCookie) — отдельного моста для Grafana не нужно, она просто
//     позади того же токена. Переход именно отсюда, а не прямой ссылкой на
//     Grafana из кнопки бота, важен: иначе первый неаутентифицированный тап
//     падал бы в сам forward_auth, а тот на отказе редиректит на голый
//     "/start?to=stats" без query-параметров (см. Caddyfile) — какой дашборд
//     был нужен, терялось бы.
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
    const dashboard = params.get('dashboard')
    const statsTarget = dashboard && dashboardUidPattern.test(dashboard) ? `/stats/d/${dashboard}/${dashboard}` : '/stats/'
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
                    window.location.href = statsTarget
                } else {
                    navigate('/categories', {replace: true})
                }
            })
            .catch(() => setError('Не удалось войти: доступ только для администраторов.'))
    }, [loading, admin, initData, login, navigate, to, statsTarget])

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
