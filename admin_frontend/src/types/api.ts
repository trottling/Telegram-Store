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
    referrer_id?: number | null
    referrals_enabled: boolean
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

export type Merchant = 'crystalpay' | 'yookassa' | 'tinkoff' | 'dummy' | 'referral'
export type ReplenishmentStatus = 'pending' | 'paid' | 'failed' | 'cancelled'

export interface ReplenishmentAdminItem {
    id: number
    user_id: number
    username: string
    merchant: Merchant
    invoice_id: string
    amount: number
    status: ReplenishmentStatus
    created_at: string
    completed_at?: string | null
}

export interface CrystalPaySettings {
    enabled: boolean
    login: string
    secret: string
    salt: string
    min_amount: number
    max_amount: number
}

export interface YooKassaSettings {
    enabled: boolean
    shop_id: string
    secret_key: string
    min_amount: number
    max_amount: number
}

export interface TinkoffSettings {
    enabled: boolean
    terminal_key: string
    password: string
    min_amount: number
    max_amount: number
}

export interface DummySettings {
    enabled: boolean
    min_amount: number
    max_amount: number
}

export interface ReferralSettings {
    enabled: boolean
    percent: number
}

export interface Settings {
    id: number
    support_username: string
    crystalpay: CrystalPaySettings
    yookassa: YooKassaSettings
    tinkoff: TinkoffSettings
    dummy: DummySettings
    referral: ReferralSettings
}

export interface ErrorResponse {
    code: string
    message: string
}
