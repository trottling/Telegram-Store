import type {ReactNode} from 'react'
import {Button, Layout as AntLayout, Menu, Typography} from 'antd'
import {AppstoreOutlined, BarChartOutlined, FileTextOutlined, LogoutOutlined, ShoppingCartOutlined, ShoppingOutlined, UserOutlined,} from '@ant-design/icons'
import {useLocation, useNavigate} from 'react-router-dom'
import {useAuth} from '../auth/AuthContext'

const {Header, Sider, Content} = AntLayout
const {Text} = Typography

const menuItems = [
    {key: '/categories', icon: <AppstoreOutlined/>, label: 'Категории'},
    {key: '/products', icon: <ShoppingOutlined/>, label: 'Товары'},
    {key: '/users', icon: <UserOutlined/>, label: 'Пользователи'},
    {key: '/purchases', icon: <ShoppingCartOutlined/>, label: 'Покупки'},
    {key: '/stats', icon: <BarChartOutlined/>, label: 'Статистика'},
    {key: '/admin-logs', icon: <FileTextOutlined/>, label: 'Логи админов'},
]

export function Layout({children}: { children: ReactNode }) {
    const navigate = useNavigate()
    const location = useLocation()
    const {admin, logout} = useAuth()

    const selectedKey = menuItems.find((item) => location.pathname.startsWith(item.key))?.key ?? '/categories'

    return (
        <AntLayout style={{minHeight: '100vh'}}>
            <Sider breakpoint="lg" collapsedWidth="0">
                <div style={{color: '#fff', padding: 16, fontWeight: 600, fontSize: 16}}>TG-Store Admin</div>
                <Menu
                    theme="dark"
                    mode="inline"
                    selectedKeys={[selectedKey]}
                    items={menuItems}
                    onClick={({key}) => navigate(key)}
                />
            </Sider>
            <AntLayout>
                <Header style={{background: '#fff', display: 'flex', justifyContent: 'flex-end', alignItems: 'center', gap: 16, padding: '0 24px'}}>
                    <Text>{admin?.username || admin?.telegram_id}</Text>
                    <Button icon={<LogoutOutlined/>} onClick={logout}>
                        Выйти
                    </Button>
                </Header>
                <Content style={{margin: 24}}>{children}</Content>
            </AntLayout>
        </AntLayout>
    )
}
