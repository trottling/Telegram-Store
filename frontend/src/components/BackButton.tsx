import {Button} from 'antd'
import {ArrowLeftOutlined} from '@ant-design/icons'
import {useNavigate} from 'react-router-dom'

// Явный `to`, а не navigate(-1) — страницу могли открыть напрямую по ссылке, без истории.
export function BackButton({to, label = 'Назад'}: { to: string; label?: string }) {
    const navigate = useNavigate()
    return (
        <Button icon={<ArrowLeftOutlined/>} onClick={() => navigate(to)} style={{marginBottom: 16}}>
            {label}
        </Button>
    )
}
