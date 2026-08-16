import {useEffect, useState} from 'react'
import {Button, Form, Input, message, Modal, Popconfirm, Select, Space, Table, Typography} from 'antd'
import {DeleteOutlined, EditOutlined, PlusOutlined} from '@ant-design/icons'
import {createCategory, deleteCategory, listCategories, updateCategory} from '../api/resources'
import {ApiError} from '../api/client'
import type {Category} from '../types/api'

const {Title} = Typography

interface TreeRow extends Category {
    children?: TreeRow[]
    key: number
}

function buildTree(categories: Category[]): TreeRow[] {
    const byParent = new Map<number | 'root', TreeRow[]>()
    const rows: TreeRow[] = categories.map((c) => ({...c, key: c.id}))
    for (const row of rows) {
        const parentKey = row.parent_id ?? 'root'
        if (!byParent.has(parentKey)) byParent.set(parentKey, [])
        byParent.get(parentKey)!.push(row)
    }
    const attach = (nodes: TreeRow[]): TreeRow[] =>
        nodes.map((n) => {
            const children = byParent.get(n.id)
            return children && children.length ? {...n, children: attach(children)} : n
        })
    return attach(byParent.get('root') ?? [])
}

export function CategoriesPage() {
    const [categories, setCategories] = useState<Category[]>([])
    const [loading, setLoading] = useState(false)
    const [modalOpen, setModalOpen] = useState(false)
    const [editing, setEditing] = useState<Category | null>(null)
    const [form] = Form.useForm()

    const load = async () => {
        setLoading(true)
        try {
            setCategories(await listCategories())
        } catch {
            message.error('Не удалось загрузить категории')
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        load()
    }, [])

    const openCreate = () => {
        setEditing(null)
        form.resetFields()
        setModalOpen(true)
    }

    const openEdit = (category: Category) => {
        setEditing(category)
        form.setFieldsValue({parent_id: category.parent_id ?? null, name: category.name, description: category.description})
        setModalOpen(true)
    }

    const handleSubmit = async () => {
        const values = await form.validateFields()
        const body = {parent_id: values.parent_id ?? null, name: values.name, description: values.description ?? ''}
        try {
            if (editing) {
                await updateCategory(editing.id, body)
                message.success('Категория обновлена')
            } else {
                await createCategory(body)
                message.success('Категория создана')
            }
            setModalOpen(false)
            load()
        } catch (err) {
            message.error(err instanceof ApiError ? err.message : 'Не удалось сохранить категорию')
        }
    }

    const handleDelete = async (id: number) => {
        try {
            await deleteCategory(id)
            message.success('Категория удалена')
            load()
        } catch (err) {
            message.error(err instanceof ApiError ? err.message : 'Не удалось удалить категорию')
        }
    }

    const columns = [
        {title: 'Название', dataIndex: 'name', key: 'name'},
        {title: 'Описание', dataIndex: 'description', key: 'description'},
        {title: 'ID', dataIndex: 'id', key: 'id', width: 80},
        {
            title: '',
            key: 'actions',
            width: 120,
            render: (_: unknown, row: Category) => (
                <Space>
                    <Button size="small" icon={<EditOutlined/>} onClick={() => openEdit(row)}/>
                    <Popconfirm title="Удалить категорию?" onConfirm={() => handleDelete(row.id)}>
                        <Button size="small" danger icon={<DeleteOutlined/>}/>
                    </Popconfirm>
                </Space>
            ),
        },
    ]

    return (
        <div>
            <div style={{display: 'flex', justifyContent: 'space-between', marginBottom: 16}}>
                <Title level={3}>Категории</Title>
                <Button type="primary" icon={<PlusOutlined/>} onClick={openCreate}>
                    Добавить категорию
                </Button>
            </div>
            <Table
                rowKey="id"
                loading={loading}
                columns={columns}
                dataSource={buildTree(categories)}
                pagination={false}
            />

            <Modal
                title={editing ? 'Изменить категорию' : 'Новая категория'}
                open={modalOpen}
                onOk={handleSubmit}
                onCancel={() => setModalOpen(false)}
                destroyOnClose
            >
                <Form form={form} layout="vertical">
                    <Form.Item name="parent_id" label="Родительская категория">
                        <Select
                            allowClear
                            placeholder="Корневая категория"
                            options={categories
                                .filter((c) => c.id !== editing?.id)
                                .map((c) => ({value: c.id, label: c.name}))}
                        />
                    </Form.Item>
                    <Form.Item name="name" label="Название" rules={[{required: true, message: 'Введите название'}]}>
                        <Input/>
                    </Form.Item>
                    <Form.Item name="description" label="Описание">
                        <Input.TextArea rows={3}/>
                    </Form.Item>
                </Form>
            </Modal>
        </div>
    )
}
