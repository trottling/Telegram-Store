import {useState} from 'react'
import {Alert, Button, Card, Form, Input, Typography} from 'antd'
import {useNavigate} from 'react-router-dom'
import {useAuth} from '../auth/AuthContext'
import {ApiError} from '../api/client'

const {Title, Text} = Typography

export function LoginPage() {
    const {login} = useAuth()
    const navigate = useNavigate()
    const [error, setError] = useState<string | null>(null)
    const [loading, setLoading] = useState(false)

    const onFinish = async (values: { code: string }) => {
        setError(null)
        setLoading(true)
        try {
            await login(values.code.trim())
            navigate('/categories', {replace: true})
        } catch (err) {
            if (err instanceof ApiError && err.status === 401) {
                setError('Код неверен или истёк — отправьте /admin боту ещё раз')
            } else {
                setError('Не удалось подключиться к серверу')
            }
        } finally {
            setLoading(false)
        }
    }

    return (
        <div style={{display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh', background: '#f0f2f5'}}>
            <Card style={{width: 380}}>
                <Title level={3} style={{textAlign: 'center'}}>
                    TG-Store Admin
                </Title>
                <Text type="secondary" style={{display: 'block', textAlign: 'center', marginBottom: 24}}>
                    Отправьте боту команду /admin и введите код — он действует 30 секунд
                </Text>
                {error && <Alert type="error" message={error} style={{marginBottom: 16}}/>}
                <Form layout="vertical" onFinish={onFinish}>
                    <Form.Item name="code" rules={[{required: true, message: 'Введите код'}]}>
                        <Input
                            placeholder="123456"
                            autoFocus
                            maxLength={6}
                            inputMode="numeric"
                            style={{textAlign: 'center', fontSize: 20, letterSpacing: 4}}
                        />
                    </Form.Item>
                    <Form.Item>
                        <Button type="primary" htmlType="submit" block loading={loading}>
                            Войти
                        </Button>
                    </Form.Item>
                </Form>
            </Card>
        </div>
    )
}
