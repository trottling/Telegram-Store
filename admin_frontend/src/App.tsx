import {lazy, Suspense} from 'react'
import {Navigate, Route, Routes} from 'react-router-dom'
import {ConfigProvider, Spin} from 'antd'
import {AuthProvider} from './auth/AuthContext'
import {RequireAuth} from './auth/RequireAuth'
import {Layout} from './components/Layout'
import {LoginPage} from './pages/LoginPage'

// Ленивая загрузка — каждая страница своим чанком.
const CategoriesPage = lazy(() => import('./pages/CategoriesPage').then((m) => ({default: m.CategoriesPage})))
const ProductsPage = lazy(() => import('./pages/ProductsPage').then((m) => ({default: m.ProductsPage})))
const UsersPage = lazy(() => import('./pages/UsersPage').then((m) => ({default: m.UsersPage})))
const UserDetailPage = lazy(() => import('./pages/UserDetailPage').then((m) => ({default: m.UserDetailPage})))
const PurchasesPage = lazy(() => import('./pages/PurchasesPage').then((m) => ({default: m.PurchasesPage})))
const PurchaseDetailPage = lazy(() => import('./pages/PurchaseDetailPage').then((m) => ({default: m.PurchaseDetailPage})))
const ReplenishmentsPage = lazy(() => import('./pages/ReplenishmentsPage').then((m) => ({default: m.ReplenishmentsPage})))
const AdminLogsPage = lazy(() => import('./pages/AdminLogsPage').then((m) => ({default: m.AdminLogsPage})))
const SettingsPage = lazy(() => import('./pages/SettingsPage').then((m) => ({default: m.SettingsPage})))

function Protected({children}: { children: React.ReactNode }) {
    return (
        <RequireAuth>
            <Layout>{children}</Layout>
        </RequireAuth>
    )
}

function PageFallback() {
    return (
        <div style={{display: 'flex', justifyContent: 'center', padding: 48}}>
            <Spin size="large"/>
        </div>
    )
}

export default function App() {
    return (
        <ConfigProvider theme={{token: {colorPrimary: '#1677ff'}}}>
            <AuthProvider>
                <Suspense fallback={<PageFallback/>}>
                    <Routes>
                        <Route path="/login" element={<LoginPage/>}/>
                        <Route path="/categories" element={<Protected><CategoriesPage/></Protected>}/>
                        <Route path="/products" element={<Protected><ProductsPage/></Protected>}/>
                        <Route path="/users" element={<Protected><UsersPage/></Protected>}/>
                        <Route path="/users/:telegramId" element={<Protected><UserDetailPage/></Protected>}/>
                        <Route path="/purchases" element={<Protected><PurchasesPage/></Protected>}/>
                        <Route path="/purchases/:id" element={<Protected><PurchaseDetailPage/></Protected>}/>
                        <Route path="/replenishments" element={<Protected><ReplenishmentsPage/></Protected>}/>
                        <Route path="/admin-logs" element={<Protected><AdminLogsPage/></Protected>}/>
                        <Route path="/settings" element={<Protected><SettingsPage/></Protected>}/>
                        <Route path="*" element={<Navigate to="/categories" replace/>}/>
                    </Routes>
                </Suspense>
            </AuthProvider>
        </ConfigProvider>
    )
}
