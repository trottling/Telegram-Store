// По одной функции на эндпоинт backend'а, всё в одном файле.
import {apiRequest} from './client'
import type {AdminLog, AdminUser, Category, Paginated, Product, ProductAdminSummary, PurchaseAdminItem, ReplenishmentAdminItem, Settings, TokenResponse,} from '../types/api'

// аутентификация
export const exchangeLoginCode = (code: string) =>
    apiRequest<TokenResponse>('/api/auth/exchange', {method: 'POST', body: {code}})
export const getMe = () => apiRequest<AdminUser>('/api/auth/me')
export const logoutRequest = () => apiRequest<void>('/api/auth/logout', {method: 'POST'})

// категории
export const listCategories = () => apiRequest<Category[]>('/api/categories')
export const getCategory = (id: number) => apiRequest<Category>(`/api/categories/${id}`)
export const createCategory = (body: { parent_id: number | null; name: string; description: string }) =>
    apiRequest<Category>('/api/categories', {method: 'POST', body})
export const updateCategory = (id: number, body: { parent_id: number | null; name: string; description: string }) =>
    apiRequest<Category>(`/api/categories/${id}`, {method: 'PUT', body})
export const deleteCategory = (id: number) => apiRequest<void>(`/api/categories/${id}`, {method: 'DELETE'})

// товары
export const listProducts = (params: { offset?: number; limit?: number; category_id?: number }) =>
    apiRequest<Paginated<ProductAdminSummary>>('/api/products', {query: params})
export const getProduct = (id: number) => apiRequest<Product>(`/api/products/${id}`)
export const createProduct = (body: { category_id: number | null; name: string; description: string; price: number }) =>
    apiRequest<Product>('/api/products', {method: 'POST', body})
export const updateProduct = (
    id: number,
    body: { category_id: number | null; name: string; description: string; price: number; is_active: boolean },
) => apiRequest<Product>(`/api/products/${id}`, {method: 'PUT', body})
export const deleteProduct = (id: number) => apiRequest<void>(`/api/products/${id}`, {method: 'DELETE'})
export const addProductItems = (id: number, contents: string[]) =>
    apiRequest<void>(`/api/products/${id}/items`, {method: 'POST', body: {contents}})

// пользователи
export const listUsers = (params: { offset?: number; limit?: number }) =>
    apiRequest<Paginated<AdminUser>>('/api/users', {query: params})
export const getUser = (id: number) => apiRequest<AdminUser>(`/api/users/${id}`)
export const banUser = (id: number) => apiRequest<void>(`/api/users/${id}/ban`, {method: 'POST'})
export const unbanUser = (id: number) => apiRequest<void>(`/api/users/${id}/unban`, {method: 'POST'})
export const adjustBalance = (id: number, amount: number) =>
    apiRequest<void>(`/api/users/${id}/balance`, {method: 'POST', body: {amount}})
export const promoteUser = (id: number) => apiRequest<void>(`/api/users/${id}/promote`, {method: 'POST'})
export const demoteUser = (id: number) => apiRequest<void>(`/api/users/${id}/demote`, {method: 'POST'})
export const listUserReferrals = (id: number, params: { offset?: number; limit?: number }) =>
    apiRequest<Paginated<AdminUser>>(`/api/users/${id}/referrals`, {query: params})
export const enableUserReferrals = (id: number) => apiRequest<void>(`/api/users/${id}/referrals/enable`, {method: 'POST'})
export const disableUserReferrals = (id: number) => apiRequest<void>(`/api/users/${id}/referrals/disable`, {method: 'POST'})

// покупки
export const listPurchases = (params: {
    offset?: number
    limit?: number
    user_id?: number
    status?: string
    from?: string
    to?: string
}) => apiRequest<Paginated<PurchaseAdminItem>>('/api/purchases', {query: params})
export const getPurchase = (id: number) => apiRequest<PurchaseAdminItem>(`/api/purchases/${id}`)

// пополнения
export const listReplenishments = (params: { offset?: number; limit?: number; user_id?: number; merchant?: string }) =>
    apiRequest<Paginated<ReplenishmentAdminItem>>('/api/replenishments', {query: params})

// логи админов
export const listAdminLogs = (params: { offset?: number; limit?: number; admin_id?: number }) =>
    apiRequest<Paginated<AdminLog>>('/api/admin-logs', {query: params})

// настройки
export const getSettings = () => apiRequest<Settings>('/api/settings')
export const updateSettings = (body: Omit<Settings, 'id'>) => apiRequest<Settings>('/api/settings', {method: 'PUT', body})
