import {useEffect, useState} from 'react'
import {useNavigate, useParams} from 'react-router-dom'
import {Button, Card, Descriptions, InputNumber, message, Modal, Popconfirm, Space, Table, Tag, Typography} from 'antd'
import {adjustBalance, banUser, demoteUser, getUser, listPurchases, promoteUser, unbanUser} from '../api/resources'
import {ApiError} from '../api/client'
import {useAuth} from '../auth/AuthContext'
import {BackButton} from '../components/BackButton'
import type {AdminUser, PurchaseAdminItem, PurchaseStatus} from '../types/api'
import {roleColor, roleLabel} from '../types/role'

const {Title} = Typography

const statusColor: Record<PurchaseStatus, string> = {
    pending: 'gold',
    completed: 'green',
    cancelled: 'red',
}

export function UserDetailPage() {
    const {telegramId} = useParams()
    const navigate = useNavigate()
    const {admin: currentAdmin} = useAuth()
    const [user, setUser] = useState<AdminUser | null>(null)
    const [loading, setLoading] = useState(false)

    const [balanceModalOpen, setBalanceModalOpen] = useState(false)
    const [balanceDelta, setBalanceDelta] = useState<number>(0)
    const [balanceSubmitting, setBalanceSubmitting] = useState(false)

    const [purchases, setPurchases] = useState<PurchaseAdminItem[]>([])
    const [purchasesTotal, setPurchasesTotal] = useState(0)
    const [purchasesPage, setPurchasesPage] = useState(1)
    const [purchasesLoading, setPurchasesLoading] = useState(false)
    const purchasesPageSize = 10

    const id = Number(telegramId)

    const load = async () => {
        setLoading(true)
        try {
            setUser(await getUser(id))
        } catch {
            message.error('Пользователь не найден')
            navigate('/users')
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        load()
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [id])

    useEffect(() => {
        setPurchasesLoading(true)
        listPurchases({user_id: id, offset: (purchasesPage - 1) * purchasesPageSize, limit: purchasesPageSize})
            .then((res) => {
                setPurchases(res.items)
                setPurchasesTotal(res.total)
            })
            .finally(() => setPurchasesLoading(false))
    }, [id, purchasesPage])

    const withErrorHandling = (fn: () => Promise<void>) => async () => {
        try {
            await fn()
            await load()
        } catch (err) {
            message.error(err instanceof ApiError ? err.message : 'Операция не удалась')
        }
    }

    const handleBan = withErrorHandling(async () => {
        await banUser(id)
        message.success('Пользователь забанен')
    })
    const handleUnban = withErrorHandling(async () => {
        await unbanUser(id)
        message.success('Пользователь разбанен')
    })
    const handlePromote = withErrorHandling(async () => {
        await promoteUser(id)
        message.success('Права администратора выданы. Чтобы войти в панель, новому админу нужно отправить боту /admin.')
    })
    const handleDemote = withErrorHandling(async () => {
        await demoteUser(id)
        message.success('Права администратора сняты')
    })

    const handleBalanceSubmit = async () => {
        if (!balanceDelta) {
            message.warning('Введите сумму')
            return
        }
        setBalanceSubmitting(true)
        try {
            await adjustBalance(id, balanceDelta)
            message.success('Баланс изменён')
            setBalanceModalOpen(false)
            setBalanceDelta(0)
            await load()
        } catch (err) {
            message.error(err instanceof ApiError ? err.message : 'Операция не удалась')
        } finally {
            setBalanceSubmitting(false)
        }
    }

    if (!user) return null

    const isSelf = currentAdmin?.telegram_id === user.telegram_id
    const isRootAdmin = user.role === 'root_admin'
    const isBanned = user.role === 'banned'
    const isAdmin = user.role === 'admin' || user.role === 'root_admin'
    // MakeAdmin на сервере только для root — скрываем кнопку для остальных.
    const canPromote = currentAdmin?.role === 'root_admin'

    const purchaseColumns = [
        {title: 'ID', dataIndex: 'id', key: 'id', width: 80},
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
            <BackButton to="/users"/>
            <Title level={3}>Пользователь {user.username || user.telegram_id}</Title>
            <Card loading={loading}>
                <Descriptions column={1} bordered>
                    <Descriptions.Item label="Telegram ID">{user.telegram_id}</Descriptions.Item>
                    <Descriptions.Item label="Username">{user.username || '—'}</Descriptions.Item>
                    <Descriptions.Item label="Баланс">{user.balance.toFixed(2)}</Descriptions.Item>
                    <Descriptions.Item label="Статус">
                        <Tag color={roleColor[user.role]}>{roleLabel[user.role]}</Tag>
                    </Descriptions.Item>
                    <Descriptions.Item label="Зарегистрирован">{new Date(user.created_at).toLocaleString()}</Descriptions.Item>
                </Descriptions>
            </Card>

            <Card title="Действия" style={{marginTop: 16}}>
                <Space wrap>
                    {isBanned ? (
                        <Popconfirm title="Разбанить пользователя?" okText="Разбанить" cancelText="Отмена" onConfirm={handleUnban}>
                            <Button>Разбанить</Button>
                        </Popconfirm>
                    ) : (
                        <Popconfirm
                            title="Забанить пользователя?"
                            okText="Забанить"
                            cancelText="Отмена"
                            disabled={isSelf || isRootAdmin}
                            onConfirm={handleBan}
                        >
                            <Button
                                danger
                                disabled={isSelf || isRootAdmin}
                                title={isSelf ? 'Нельзя забанить себя' : isRootAdmin ? 'Нельзя забанить root admin' : ''}
                            >
                                Забанить
                            </Button>
                        </Popconfirm>
                    )}

                    <Button onClick={() => setBalanceModalOpen(true)}>Изменить баланс</Button>

                    {isAdmin ? (
                        <Popconfirm
                            title="Снять права администратора?"
                            okText="Снять"
                            cancelText="Отмена"
                            disabled={isSelf || isRootAdmin}
                            onConfirm={handleDemote}
                        >
                            <Button
                                danger
                                disabled={isSelf || isRootAdmin}
                                title={isSelf ? 'Нельзя снять права с себя' : isRootAdmin ? 'Нельзя снять права с root admin' : ''}
                            >
                                Снять права администратора
                            </Button>
                        </Popconfirm>
                    ) : (
                        canPromote && (
                            <Popconfirm title="Выдать права администратора?" okText="Выдать" cancelText="Отмена" onConfirm={handlePromote}>
                                <Button type="primary">Выдать права администратора</Button>
                            </Popconfirm>
                        )
                    )}
                </Space>
            </Card>

            <Card title="История покупок" style={{marginTop: 16}}>
                <Table
                    rowKey="id"
                    loading={purchasesLoading}
                    columns={purchaseColumns}
                    dataSource={purchases}
                    onRow={(row) => ({onClick: () => navigate(`/purchases/${row.id}`), style: {cursor: 'pointer'}})}
                    pagination={{
                        current: purchasesPage,
                        pageSize: purchasesPageSize,
                        total: purchasesTotal,
                        onChange: setPurchasesPage,
                        showSizeChanger: false,
                    }}
                />
            </Card>

            <Modal
                title="Изменить баланс"
                open={balanceModalOpen}
                onOk={handleBalanceSubmit}
                onCancel={() => {
                    setBalanceModalOpen(false)
                    setBalanceDelta(0)
                }}
                confirmLoading={balanceSubmitting}
                okText="Изменить"
                cancelText="Отмена"
            >
                <InputNumber
                    autoFocus
                    placeholder="Сумма (можно отрицательную)"
                    value={balanceDelta || undefined}
                    onChange={(v) => setBalanceDelta(v ?? 0)}
                    style={{width: '100%'}}
                />
            </Modal>
        </div>
    )
}
