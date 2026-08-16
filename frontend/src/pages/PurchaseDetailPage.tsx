import {useEffect, useState} from 'react'
import {useNavigate, useParams} from 'react-router-dom'
import {Card, Descriptions, message, Tag, Typography} from 'antd'
import {getPurchase} from '../api/resources'
import {BackButton} from '../components/BackButton'
import type {PurchaseAdminItem, PurchaseStatus} from '../types/api'

const {Title} = Typography

const statusColor: Record<PurchaseStatus, string> = {
    pending: 'gold',
    completed: 'green',
    cancelled: 'red',
}

export function PurchaseDetailPage() {
    const {id} = useParams()
    const navigate = useNavigate()
    const [purchase, setPurchase] = useState<PurchaseAdminItem | null>(null)
    const [loading, setLoading] = useState(false)

    useEffect(() => {
        setLoading(true)
        getPurchase(Number(id))
            .then(setPurchase)
            .catch(() => {
                message.error('Покупка не найдена')
                navigate('/purchases')
            })
            .finally(() => setLoading(false))
    }, [id, navigate])

    if (!purchase) return null

    return (
        <div>
            <BackButton to="/purchases"/>
            <Title level={3}>Покупка #{purchase.id}</Title>
            <Card loading={loading}>
                <Descriptions column={1} bordered>
                    <Descriptions.Item label="Пользователь">
                        {purchase.username || purchase.user_id} ({purchase.user_id})
                    </Descriptions.Item>
                    <Descriptions.Item label="Товар">{purchase.product_name}</Descriptions.Item>
                    <Descriptions.Item label="Сумма">{purchase.amount.toFixed(2)}</Descriptions.Item>
                    <Descriptions.Item label="Статус">
                        <Tag color={statusColor[purchase.status]}>{purchase.status}</Tag>
                    </Descriptions.Item>
                    <Descriptions.Item label="Batch ID">{purchase.batch_id}</Descriptions.Item>
                    <Descriptions.Item label="Создана">{new Date(purchase.created_at).toLocaleString()}</Descriptions.Item>
                    <Descriptions.Item label="Завершена">
                        {purchase.completed_at ? new Date(purchase.completed_at).toLocaleString() : '—'}
                    </Descriptions.Item>
                </Descriptions>
            </Card>
        </div>
    )
}
