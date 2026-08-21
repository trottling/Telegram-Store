import type {Merchant, ReplenishmentStatus} from './api'

// Единое отображение мерчанта/статуса пополнения — используется в
// ReplenishmentsPage и UserDetailPage.
export const merchantLabel: Record<Merchant, string> = {
    crystalpay: 'CrystalPay',
    yookassa: 'ЮKassa',
    tinkoff: 'Тинькофф',
    dummy: 'Тест',
    referral: 'Реферальная программа',
}

export const replenishmentStatusColor: Record<ReplenishmentStatus, string> = {
    pending: 'gold',
    paid: 'green',
    failed: 'red',
    cancelled: 'default',
}
