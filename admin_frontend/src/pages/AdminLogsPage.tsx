import {useEffect, useState} from 'react'
import {InputNumber, Space, Table, Tag, Typography} from 'antd'
import {listAdminLogs} from '../api/resources'
import type {AdminLog} from '../types/api'

const {Title, Text} = Typography

const actionColor: Record<string, string> = {
    ban: 'red',
    unban: 'green',
    balance_add: 'gold',
    make_admin: 'blue',
    revoke_admin: 'volcano',
    product_create: 'cyan',
    product_update: 'geekblue',
    product_delete: 'red',
    product_add_items: 'cyan',
    category_create: 'cyan',
    category_update: 'geekblue',
    category_delete: 'red',
}

export function AdminLogsPage() {
    const [logs, setLogs] = useState<AdminLog[]>([])
    const [total, setTotal] = useState(0)
    const [page, setPage] = useState(1)
    const pageSize = 20
    const [adminId, setAdminId] = useState<number | null>(null)
    const [loading, setLoading] = useState(false)

    useEffect(() => {
        setLoading(true)
        listAdminLogs({
            offset: (page - 1) * pageSize,
            limit: pageSize,
            admin_id: adminId ?? undefined,
        })
            .then((res) => {
                setLogs(res.items)
                setTotal(res.total)
            })
            .finally(() => setLoading(false))
    }, [page, adminId])

    const columns = [
        {title: 'ID', dataIndex: 'id', key: 'id', width: 80},
        {title: 'Админ', dataIndex: 'admin_id', key: 'admin_id', width: 140},
        {
            title: 'Действие',
            dataIndex: 'action',
            key: 'action',
            render: (v: string) => <Tag color={actionColor[v] ?? 'default'}>{v}</Tag>,
        },
        {
            title: 'Цель',
            dataIndex: 'target_id',
            key: 'target_id',
            width: 100,
            render: (v: number | null | undefined) => v ?? '—',
        },
        {
            title: 'Детали',
            dataIndex: 'details',
            key: 'details',
            render: (v: unknown) =>
                v ? <Text code style={{fontSize: 12}}>{JSON.stringify(v)}</Text> : '—',
        },
        {title: 'Дата', dataIndex: 'created_at', key: 'created_at', render: (v: string) => new Date(v).toLocaleString()},
    ]

    return (
        <div>
            <div style={{display: 'flex', justifyContent: 'space-between', marginBottom: 16}}>
                <Title level={3}>Логи администраторов</Title>
                <Space>
                    <InputNumber
                        placeholder="ID администратора"
                        style={{width: 180}}
                        value={adminId}
                        onChange={(v) => {
                            setAdminId(v)
                            setPage(1)
                        }}
                    />
                </Space>
            </div>
            <Table
                rowKey="id"
                loading={loading}
                columns={columns}
                dataSource={logs}
                pagination={{current: page, pageSize, total, onChange: setPage, showSizeChanger: false}}
            />
        </div>
    )
}
