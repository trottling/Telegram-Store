import {useEffect, useState} from 'react'
import {Alert, Button, Card, Form, Input, InputNumber, message, Space, Switch, Typography} from 'antd'
import {SaveOutlined} from '@ant-design/icons'
import {getSettings, updateSettings} from '../api/resources'
import {ApiError} from '../api/client'
import type {Settings} from '../types/api'

const {Title} = Typography

export function SettingsPage() {
    const [form] = Form.useForm<Omit<Settings, 'id'>>()
    const [loading, setLoading] = useState(true)
    const [saving, setSaving] = useState(false)

    useEffect(() => {
        getSettings()
            .then((settings) => form.setFieldsValue(settings))
            .catch(() => message.error('Не удалось загрузить настройки'))
            .finally(() => setLoading(false))
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

    const handleSave = async () => {
        const values = await form.validateFields()
        setSaving(true)
        try {
            const settings = await updateSettings(values)
            form.setFieldsValue(settings)
            message.success('Настройки сохранены')
        } catch (err) {
            message.error(err instanceof ApiError ? err.message : 'Не удалось сохранить настройки')
        } finally {
            setSaving(false)
        }
    }

    return (
        <div>
            <div style={{display: 'flex', justifyContent: 'space-between', marginBottom: 16}}>
                <Title level={3}>Настройки</Title>
                <Button type="primary" icon={<SaveOutlined/>} loading={saving} onClick={handleSave}>
                    Сохранить
                </Button>
            </div>

            <Form form={form} layout="vertical" disabled={loading}>
                <Card title="Общее" style={{marginBottom: 16}}>
                    <Form.Item
                        name="support_username"
                        label="Username поддержки"
                        rules={[{required: true, message: 'Введите username поддержки'}]}
                    >
                        <Input placeholder="@username"/>
                    </Form.Item>
                    <Form.Item
                        name="catalog_refresh_interval_seconds"
                        label="Интервал обновления каталога, сек"
                        tooltip="Как часто фоновый воркер бота пересчитывает видимость категорий/товаров (наличие остатка). Применяется без перезапуска бота — подхватывается на следующем цикле воркера."
                        rules={[{required: true, type: 'number', min: 5, max: 3600, message: 'От 5 до 3600'}]}
                    >
                        <InputNumber min={5} max={3600} style={{width: 160}}/>
                    </Form.Item>
                </Card>

                <Card title="Реферальная программа" style={{marginBottom: 16}}>
                    <Space size="large" align="start">
                        <Form.Item name={['referral', 'enabled']} label="Включена" valuePropName="checked">
                            <Switch/>
                        </Form.Item>
                        <Form.Item
                            name={['referral', 'percent']}
                            label="Процент с покупки реферала"
                            rules={[{required: true, type: 'number', min: 0, max: 100, message: 'От 0 до 100'}]}
                        >
                            <InputNumber min={0} max={100} addonAfter="%" style={{width: 160}}/>
                        </Form.Item>
                    </Space>
                </Card>

                <Card title="CrystalPay" style={{marginBottom: 16}}>
                    <Form.Item name={['crystalpay', 'enabled']} label="Включён" valuePropName="checked">
                        <Switch/>
                    </Form.Item>
                    <Form.Item name={['crystalpay', 'login']} label="Login">
                        <Input/>
                    </Form.Item>
                    <Form.Item name={['crystalpay', 'secret']} label="Secret">
                        <Input.Password/>
                    </Form.Item>
                    <Form.Item name={['crystalpay', 'salt']} label="Salt">
                        <Input.Password/>
                    </Form.Item>
                    <Space size="large">
                        <Form.Item name={['crystalpay', 'min_amount']} label="Мин. сумма, ₽">
                            <InputNumber min={0} step={0.01} style={{width: 160}}/>
                        </Form.Item>
                        <Form.Item name={['crystalpay', 'max_amount']} label="Макс. сумма, ₽ (0 — без ограничения)">
                            <InputNumber min={0} step={0.01} style={{width: 160}}/>
                        </Form.Item>
                    </Space>
                </Card>

                <Card title="ЮKassa" style={{marginBottom: 16}}>
                    <Form.Item name={['yookassa', 'enabled']} label="Включена" valuePropName="checked">
                        <Switch/>
                    </Form.Item>
                    <Form.Item name={['yookassa', 'shop_id']} label="Shop ID">
                        <Input/>
                    </Form.Item>
                    <Form.Item name={['yookassa', 'secret_key']} label="Secret key">
                        <Input.Password/>
                    </Form.Item>
                    <Space size="large">
                        <Form.Item name={['yookassa', 'min_amount']} label="Мин. сумма, ₽">
                            <InputNumber min={0} step={0.01} style={{width: 160}}/>
                        </Form.Item>
                        <Form.Item name={['yookassa', 'max_amount']} label="Макс. сумма, ₽ (0 — без ограничения)">
                            <InputNumber min={0} step={0.01} style={{width: 160}}/>
                        </Form.Item>
                    </Space>
                </Card>

                <Card title="Тинькофф" style={{marginBottom: 16}}>
                    <Form.Item name={['tinkoff', 'enabled']} label="Включён" valuePropName="checked">
                        <Switch/>
                    </Form.Item>
                    <Form.Item name={['tinkoff', 'terminal_key']} label="Terminal key">
                        <Input/>
                    </Form.Item>
                    <Form.Item name={['tinkoff', 'password']} label="Password">
                        <Input.Password/>
                    </Form.Item>
                    <Space size="large">
                        <Form.Item name={['tinkoff', 'min_amount']} label="Мин. сумма, ₽">
                            <InputNumber min={0} step={0.01} style={{width: 160}}/>
                        </Form.Item>
                        <Form.Item name={['tinkoff', 'max_amount']} label="Макс. сумма, ₽ (0 — без ограничения)">
                            <InputNumber min={0} step={0.01} style={{width: 160}}/>
                        </Form.Item>
                    </Space>
                </Card>

                {/* Красная рамка/заголовок — не настоящий мерчант, деньги нигде не
                    двигаются, счёт подтверждается сам через ~10 секунд. Включать
                    только для разработки/демо, не в проде. */}
                <Card
                    title={<span style={{color: '#cf1322'}}>⚠️ Тестовый провайдер (dummy)</span>}
                    style={{borderColor: '#ffa39e'}}
                    styles={{header: {borderColor: '#ffa39e'}}}
                >
                    <Alert
                        type="error"
                        showIcon
                        style={{marginBottom: 16}}
                        message="Не настоящая оплата — счёт подтверждается сам через 10 секунд после создания. Включайте только для разработки или демо."
                    />
                    <Form.Item name={['dummy', 'enabled']} label="Включён" valuePropName="checked">
                        <Switch/>
                    </Form.Item>
                    <Space size="large">
                        <Form.Item name={['dummy', 'min_amount']} label="Мин. сумма, ₽">
                            <InputNumber min={0} step={0.01} style={{width: 160}}/>
                        </Form.Item>
                        <Form.Item name={['dummy', 'max_amount']} label="Макс. сумма, ₽ (0 — без ограничения)">
                            <InputNumber min={0} step={0.01} style={{width: 160}}/>
                        </Form.Item>
                    </Space>
                </Card>
            </Form>
        </div>
    )
}
