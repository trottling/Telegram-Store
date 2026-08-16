import {useEffect, useState} from 'react'
import {Button, Form, Input, InputNumber, message, Modal, Popconfirm, Select, Space, Switch, Table, Tag, Typography} from 'antd'
import {DeleteOutlined, EditOutlined, InboxOutlined, PlusOutlined} from '@ant-design/icons'
import {addProductItems, createProduct, deleteProduct, listCategories, listProducts, updateProduct,} from '../api/resources'
import {ApiError} from '../api/client'
import type {Category, ProductAdminSummary} from '../types/api'

const {Title} = Typography

export function ProductsPage() {
    const [products, setProducts] = useState<ProductAdminSummary[]>([])
    const [total, setTotal] = useState(0)
    const [categories, setCategories] = useState<Category[]>([])
    const [categoryFilter, setCategoryFilter] = useState<number | undefined>(undefined)
    const [page, setPage] = useState(1)
    const pageSize = 20
    const [loading, setLoading] = useState(false)

    const [modalOpen, setModalOpen] = useState(false)
    const [editing, setEditing] = useState<ProductAdminSummary | null>(null)
    const [form] = Form.useForm()

    const [stockModalOpen, setStockModalOpen] = useState(false)
    const [stockTarget, setStockTarget] = useState<ProductAdminSummary | null>(null)
    const [stockText, setStockText] = useState('')

    const load = async () => {
        setLoading(true)
        try {
            const res = await listProducts({offset: (page - 1) * pageSize, limit: pageSize, category_id: categoryFilter})
            setProducts(res.items)
            setTotal(res.total)
        } catch {
            message.error('Не удалось загрузить товары')
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        listCategories().then(setCategories).catch(() => undefined)
    }, [])

    useEffect(() => {
        load()
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [page, categoryFilter])

    const openCreate = () => {
        setEditing(null)
        form.resetFields()
        form.setFieldsValue({is_active: true})
        setModalOpen(true)
    }

    const openEdit = (product: ProductAdminSummary) => {
        setEditing(product)
        form.setFieldsValue({
            category_id: product.category_id ?? null,
            name: product.name,
            description: product.description,
            price: product.price,
            is_active: product.is_active,
        })
        setModalOpen(true)
    }

    const handleSubmit = async () => {
        const values = await form.validateFields()
        const body = {
            category_id: values.category_id ?? null,
            name: values.name,
            description: values.description ?? '',
            price: values.price,
            is_active: values.is_active ?? true,
        }
        try {
            if (editing) {
                await updateProduct(editing.id, body)
                message.success('Товар обновлён')
            } else {
                await createProduct(body)
                message.success('Товар создан')
            }
            setModalOpen(false)
            load()
        } catch (err) {
            message.error(err instanceof ApiError ? err.message : 'Не удалось сохранить товар')
        }
    }

    const handleDelete = async (id: number) => {
        try {
            await deleteProduct(id)
            message.success('Товар удалён')
            load()
        } catch (err) {
            message.error(err instanceof ApiError ? err.message : 'Не удалось удалить товар')
        }
    }

    const openStock = (product: ProductAdminSummary) => {
        setStockTarget(product)
        setStockText('')
        setStockModalOpen(true)
    }

    const submitStock = async () => {
        if (!stockTarget) return
        const contents = stockText.split('\n').map((s) => s.trim()).filter(Boolean)
        if (contents.length === 0) {
            message.warning('Добавьте хотя бы одну позицию')
            return
        }
        try {
            await addProductItems(stockTarget.id, contents)
            message.success(`Добавлено ${contents.length} шт.`)
            setStockModalOpen(false)
            load()
        } catch (err) {
            message.error(err instanceof ApiError ? err.message : 'Не удалось добавить сток')
        }
    }

    const columns = [
        {title: 'Название', dataIndex: 'name', key: 'name'},
        {title: 'Категория', dataIndex: 'category_name', key: 'category_name', render: (v: string) => v || '—'},
        {title: 'Цена', dataIndex: 'price', key: 'price', render: (v: number) => v.toFixed(2)},
        {title: 'В наличии', dataIndex: 'available_count', key: 'available_count'},
        {
            title: 'Статус',
            dataIndex: 'is_active',
            key: 'is_active',
            render: (active: boolean) => (active ? <Tag color="green">активен</Tag> : <Tag>неактивен</Tag>),
        },
        {
            title: '',
            key: 'actions',
            width: 160,
            render: (_: unknown, row: ProductAdminSummary) => (
                <Space>
                    <Button size="small" icon={<InboxOutlined/>} onClick={() => openStock(row)} title="Добавить сток"/>
                    <Button size="small" icon={<EditOutlined/>} onClick={() => openEdit(row)}/>
                    <Popconfirm title="Удалить товар?" onConfirm={() => handleDelete(row.id)}>
                        <Button size="small" danger icon={<DeleteOutlined/>}/>
                    </Popconfirm>
                </Space>
            ),
        },
    ]

    return (
        <div>
            <div style={{display: 'flex', justifyContent: 'space-between', marginBottom: 16}}>
                <Title level={3}>Товары</Title>
                <Space>
                    <Select
                        allowClear
                        placeholder="Фильтр по категории"
                        style={{width: 220}}
                        options={categories.map((c) => ({value: c.id, label: c.name}))}
                        value={categoryFilter}
                        onChange={(v) => {
                            setCategoryFilter(v)
                            setPage(1)
                        }}
                    />
                    <Button type="primary" icon={<PlusOutlined/>} onClick={openCreate}>
                        Добавить товар
                    </Button>
                </Space>
            </div>
            <Table
                rowKey="id"
                loading={loading}
                columns={columns}
                dataSource={products}
                pagination={{current: page, pageSize, total, onChange: setPage, showSizeChanger: false}}
            />

            <Modal
                title={editing ? 'Изменить товар' : 'Новый товар'}
                open={modalOpen}
                onOk={handleSubmit}
                onCancel={() => setModalOpen(false)}
                destroyOnClose
            >
                <Form form={form} layout="vertical">
                    <Form.Item name="category_id" label="Категория">
                        <Select allowClear placeholder="Без категории" options={categories.map((c) => ({value: c.id, label: c.name}))}/>
                    </Form.Item>
                    <Form.Item name="name" label="Название" rules={[{required: true, message: 'Введите название'}]}>
                        <Input/>
                    </Form.Item>
                    <Form.Item name="description" label="Описание">
                        <Input.TextArea rows={3}/>
                    </Form.Item>
                    <Form.Item name="price" label="Цена" rules={[{required: true, message: 'Введите цену'}]}>
                        <InputNumber min={0.01} step={0.01} style={{width: '100%'}}/>
                    </Form.Item>
                    <Form.Item name="is_active" label="Активен" valuePropName="checked">
                        <Switch/>
                    </Form.Item>
                </Form>
            </Modal>

            <Modal
                title={`Добавить сток: ${stockTarget?.name ?? ''}`}
                open={stockModalOpen}
                onOk={submitStock}
                onCancel={() => setStockModalOpen(false)}
                destroyOnClose
            >
                <Typography.Paragraph type="secondary">Одна позиция на строку</Typography.Paragraph>
                <Input.TextArea rows={8} value={stockText} onChange={(e) => setStockText(e.target.value)} placeholder={'key1\nkey2\nkey3'}/>
            </Modal>
        </div>
    )
}
