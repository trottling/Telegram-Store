import {useEffect, useState} from 'react'
import {DatePicker, Select, Space, Table, Tag, Typography} from 'antd'
import {useNavigate} from 'react-router-dom'
import type {Dayjs} from 'dayjs'
import {listPurchases} from '../api/resources'
import type {PurchaseAdminItem, PurchaseStatus} from '../types/api'

const {Title} = Typography
const {RangePicker} = DatePicker

const statusColor: Record<PurchaseStatus, string> = {
    pending: 'gold',
    completed: 'green',
    cancelled: 'red',
}

export function PurchasesPage() {
    const navigate = useNavigate()
    const [purchases, setPurchases] = useState<PurchaseAdminItem[]>([])
    const [total, setTotal] = useState(0)
    const [page, setPage] = useState(1)
    const pageSize = 20
    const [status, setStatus] = useState<PurchaseStatus | undefined>(undefined)
    const [range, setRange] = useState<[Dayjs, Dayjs] | null>(null)
    const [loading, setLoading] = useState(false)

    useEffect(() => {
        setLoading(true)
        listPurchases({
            offset: (page - 1) * pageSize,
            limit: pageSize,
            status,
            from: range?.[0]?.startOf('day').toISOString(),
            to: range?.[1]?.endOf('day').toISOString(),
        })
            .then((res) => {
                setPurchases(res.items)
                setTotal(res.total)
            })
            .finally(() => setLoading(false))
    }, [page, status, range])

    const columns = [
        {title: 'ID', dataIndex: 'id', key: 'id', width: 80},
        {title: 'Пользователь', dataIndex: 'username', key: 'username', render: (v: string, r: PurchaseAdminItem) => v || r.user_id},
        {title: 'Товар', dataIndex: 'product_name', key: 'product_name'},
        {title: 'Сумма', dataIndex: 'amount', key: 'amount', render: (v: number) => v.toFixed(2)},
        {
            title: 'Статус',
            dataIndex: 'status',
            key: 'status',
            render: (v: PurchaseStatus) => <Tag color={statusColor[v]}>{v}</Tag>,
        },
        {title: 'Дата', dataIndex: 'created_at', key: 'created_at', render: (v: string) => new Date(v).toLocaleString()},
    ]

    return (
        <div>
            <div style={{display: 'flex', justifyContent: 'space-between', marginBottom: 16}}>
                <Title level={3}>Покупки</Title>
                <Space>
                    <Select
                        allowClear
                        placeholder="Статус"
                        style={{width: 160}}
                        value={status}
                        onChange={(v) => {
                            setStatus(v)
                            setPage(1)
                        }}
                        options={[
                            {value: 'pending', label: 'pending'},
                            {value: 'completed', label: 'completed'},
                            {value: 'cancelled', label: 'cancelled'},
                        ]}
                    />
                    <RangePicker
                        value={range}
                        onChange={(v) => {
                            setRange(v as [Dayjs, Dayjs] | null)
                            setPage(1)
                        }}
                    />
                </Space>
            </div>
            <Table
                rowKey="id"
                loading={loading}
                columns={columns}
                dataSource={purchases}
                onRow={(row) => ({onClick: () => navigate(`/purchases/${row.id}`), style: {cursor: 'pointer'}})}
                pagination={{current: page, pageSize, total, onChange: setPage, showSizeChanger: false}}
            />
        </div>
    )
}
