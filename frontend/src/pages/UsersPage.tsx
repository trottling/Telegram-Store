import {useEffect, useState} from 'react'
import {Input, Table, Tag, Typography} from 'antd'
import {useNavigate} from 'react-router-dom'
import {listUsers} from '../api/resources'
import type {AdminUser} from '../types/api'
import {roleColor, roleLabel} from '../types/role'

const {Title} = Typography

export function UsersPage() {
    const navigate = useNavigate()
    const [users, setUsers] = useState<AdminUser[]>([])
    const [total, setTotal] = useState(0)
    const [page, setPage] = useState(1)
    const [search, setSearch] = useState('')
    const pageSize = 20
    const [loading, setLoading] = useState(false)

    useEffect(() => {
        setLoading(true)
        listUsers({offset: (page - 1) * pageSize, limit: pageSize})
            .then((res) => {
                setUsers(res.items)
                setTotal(res.total)
            })
            .finally(() => setLoading(false))
    }, [page])

    const filtered = search
        ? users.filter(
            (u) => u.username.toLowerCase().includes(search.toLowerCase()) || String(u.telegram_id).includes(search),
        )
        : users

    const columns = [
        {title: 'Telegram ID', dataIndex: 'telegram_id', key: 'telegram_id'},
        {title: 'Username', dataIndex: 'username', key: 'username', render: (v: string) => v || '—'},
        {title: 'Баланс', dataIndex: 'balance', key: 'balance', render: (v: number) => v.toFixed(2)},
        {
            title: 'Статус',
            key: 'status',
            render: (_: unknown, row: AdminUser) => <Tag color={roleColor[row.role]}>{roleLabel[row.role]}</Tag>,
        },
    ]

    return (
        <div>
            <div style={{display: 'flex', justifyContent: 'space-between', marginBottom: 16}}>
                <Title level={3}>Пользователи</Title>
                <Input.Search
                    placeholder="Поиск по username или ID"
                    style={{width: 280}}
                    onChange={(e) => setSearch(e.target.value)}
                />
            </div>
            <Table
                rowKey="telegram_id"
                loading={loading}
                columns={columns}
                dataSource={filtered}
                onRow={(row) => ({onClick: () => navigate(`/users/${row.telegram_id}`), style: {cursor: 'pointer'}})}
                pagination={{current: page, pageSize, total, onChange: setPage, showSizeChanger: false}}
            />
        </div>
    )
}
