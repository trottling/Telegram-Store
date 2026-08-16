import {useEffect, useState} from 'react'
import {Card, Col, DatePicker, Row, Space, Spin, Statistic, Table, Typography} from 'antd'
import {Line} from '@ant-design/plots'
import type {Dayjs} from 'dayjs'
import {getDashboard} from '../api/resources'
import type {DashboardStats} from '../types/api'

const {Title} = Typography
const {RangePicker} = DatePicker

export function StatsPage() {
    const [stats, setStats] = useState<DashboardStats | null>(null)
    const [loading, setLoading] = useState(false)
    const [range, setRange] = useState<[Dayjs, Dayjs] | null>(null)

    useEffect(() => {
        setLoading(true)
        getDashboard({
            from: range?.[0]?.startOf('day').toISOString(),
            to: range?.[1]?.endOf('day').toISOString(),
        })
            .then(setStats)
            .finally(() => setLoading(false))
    }, [range])

    return (
        <div>
            <div style={{display: 'flex', justifyContent: 'space-between', marginBottom: 16}}>
                <Title level={3}>Статистика</Title>
                <Space>
                    <RangePicker value={range} onChange={(v) => setRange(v as [Dayjs, Dayjs] | null)}/>
                </Space>
            </div>

            {loading && !stats ? (
                <Spin size="large"/>
            ) : stats ? (
                <>
                    <Row gutter={16}>
                        <Col span={6}>
                            <Card>
                                <Statistic title="Выручка" value={stats.overview.total_revenue} precision={2}/>
                            </Card>
                        </Col>
                        <Col span={6}>
                            <Card>
                                <Statistic title="Покупок" value={stats.overview.total_purchases}/>
                            </Card>
                        </Col>
                        <Col span={6}>
                            <Card>
                                <Statistic title="Пользователей" value={stats.users.total_users}/>
                            </Card>
                        </Col>
                        <Col span={6}>
                            <Card>
                                <Statistic title="Суммарный баланс" value={stats.users.total_balance} precision={2}/>
                            </Card>
                        </Col>
                    </Row>

                    <Row gutter={16} style={{marginTop: 16}}>
                        <Col span={12}>
                            <Card>
                                <Statistic title="Забанено" value={stats.users.banned_users}/>
                            </Card>
                        </Col>
                        <Col span={12}>
                            <Card>
                                <Statistic title="Администраторов" value={stats.users.admin_users}/>
                            </Card>
                        </Col>
                    </Row>

                    <Card title="Выручка по дням" style={{marginTop: 16}}>
                        <Line
                            data={stats.revenue_series.map((p) => ({date: new Date(p.date).toLocaleDateString(), revenue: p.revenue}))}
                            xField="date"
                            yField="revenue"
                            height={280}
                        />
                    </Card>

                    <Row gutter={16} style={{marginTop: 16}}>
                        <Col span={12}>
                            <Card title="Топ товаров">
                                <Table
                                    size="small"
                                    rowKey="product_id"
                                    pagination={false}
                                    dataSource={stats.top_products}
                                    columns={[
                                        {title: 'Товар', dataIndex: 'product_name'},
                                        {title: 'Продано', dataIndex: 'units_sold', width: 90},
                                        {title: 'Выручка', dataIndex: 'revenue', width: 110, render: (v: number) => v.toFixed(2)},
                                    ]}
                                />
                            </Card>
                        </Col>
                        <Col span={12}>
                            <Card title="Топ категорий">
                                <Table
                                    size="small"
                                    rowKey={(r) => r.category_id ?? 'uncategorized'}
                                    pagination={false}
                                    dataSource={stats.top_categories}
                                    columns={[
                                        {title: 'Категория', dataIndex: 'category_name'},
                                        {title: 'Продано', dataIndex: 'units_sold', width: 90},
                                        {title: 'Выручка', dataIndex: 'revenue', width: 110, render: (v: number) => v.toFixed(2)},
                                    ]}
                                />
                            </Card>
                        </Col>
                    </Row>
                </>
            ) : null}
        </div>
    )
}
