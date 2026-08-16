import {useEffect, useState} from 'react'
import {Select, Space, Table, Tag, Typography} from 'antd'
import {listReplenishments} from '../api/resources'
import type {Merchant, ReplenishmentAdminItem} from '../types/api'
import {merchantLabel, replenishmentStatusColor} from '../types/merchant'

const {Title} = Typography

const merchantOptions = (Object.keys(merchantLabel) as Merchant[]).map((m) => ({value: m, label: merchantLabel[m]}))

export function ReplenishmentsPage() {
    const [replenishments, setReplenishments] = useState<ReplenishmentAdminItem[]>([])
    const [total, setTotal] = useState(0)
    const [page, setPage] = useState(1)
    const pageSize = 20
    const [merchant, setMerchant] = useState<Merchant | undefined>(undefined)
    const [loading, setLoading] = useState(false)

    useEffect(() => {
        setLoading(true)
        listReplenishments({offset: (page - 1) * pageSize, limit: pageSize, merchant})
            .then((res) => {
                setReplenishments(res.items)
                setTotal(res.total)
            })
            .finally(() => setLoading(false))
    }, [page, merchant])

    const columns = [
        {title: 'ID', dataIndex: 'id', key: 'id', width: 80},
        {title: 'Пользователь', dataIndex: 'username', key: 'username', render: (v: string, r: ReplenishmentAdminItem) => v || r.user_id},
        {title: 'Мерчант', dataIndex: 'merchant', key: 'merchant', render: (v: Merchant) => merchantLabel[v]},
        {title: 'Сумма', dataIndex: 'amount', key: 'amount', render: (v: number) => v.toFixed(2)},
        {
            title: 'Статус',
            dataIndex: 'status',
            key: 'status',
            render: (v: ReplenishmentAdminItem['status']) => <Tag color={replenishmentStatusColor[v]}>{v}</Tag>,
        },
        {title: 'Дата', dataIndex: 'created_at', key: 'created_at', render: (v: string) => new Date(v).toLocaleString()},
    ]

    return (
        <div>
            <div style={{display: 'flex', justifyContent: 'space-between', marginBottom: 16}}>
                <Title level={3}>Пополнения</Title>
                <Space>
                    <Select
                        allowClear
                        placeholder="Мерчант"
                        style={{width: 220}}
                        options={merchantOptions}
                        value={merchant}
                        onChange={(v) => {
                            setMerchant(v)
                            setPage(1)
                        }}
                    />
                </Space>
            </div>
            <Table
                rowKey="id"
                loading={loading}
                columns={columns}
                dataSource={replenishments}
                pagination={{current: page, pageSize, total, onChange: setPage, showSizeChanger: false}}
            />
        </div>
    )
}
