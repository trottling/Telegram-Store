// Зеркалит формы ответов/запросов backend'а (internal/domain/models + backend/dto).

export interface Paginated<T> {
    items: T[]
    total: number
    offset: number
    limit: number
}

export type UserRole = 'banned' | 'user' | 'admin' | 'root_admin'

export interface AdminUser {
    telegram_id: number
    username: string
    balance: number
    role: UserRole
    created_at: string
    updated_at: string
}

export interface Category {
    id: number
    parent_id?: number | null
    name: string
    description?: string
    created_at: string
}

export interface ProductAdminSummary {
    id: number
    category_id?: number | null
    category_name?: string
    name: string
    description: string
    price: number
    is_active: boolean
    available_count: number
    created_at: string
}

export interface ProductItem {
    id: number
    product_id: number
    is_sold: boolean
    sold_at?: string | null
    created_at: string
}

export interface Product {
    id: number
    category_id?: number | null
    name: string
    description: string
    price: number
    is_active: boolean
    created_at: string
    items?: ProductItem[]
}

export type PurchaseStatus = 'pending' | 'completed' | 'cancelled'

export interface PurchaseAdminItem {
    id: number
    user_id: number
    username: string
    product_id: number
    product_name: string
    item_id?: number | null
    batch_id: string
    amount: number
    status: PurchaseStatus
    created_at: string
    completed_at?: string | null
}

export interface SalesOverview {
    total_revenue: number
    total_purchases: number
}

export interface RevenuePoint {
    date: string
    revenue: number
    count: number
}

export interface ProductStat {
    product_id: number
    product_name: string
    units_sold: number
    revenue: number
}

export interface CategoryStat {
    category_id?: number | null
    category_name: string
    units_sold: number
    revenue: number
}

export interface UserStats {
    total_users: number
    banned_users: number
    admin_users: number
    total_balance: number
}

export interface DashboardStats {
    overview: SalesOverview
    revenue_series: RevenuePoint[]
    top_products: ProductStat[]
    top_categories: CategoryStat[]
    users: UserStats
}

export interface TokenResponse {
    token: string
}

export interface AdminLog {
    id: number
    admin_id: number
    action: string
    target_id?: number | null
    details?: unknown
    created_at: string
}

export interface ErrorResponse {
    code: string
    message: string
}
