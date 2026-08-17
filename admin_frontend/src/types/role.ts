import type {UserRole} from './api'

// Единое отображение роли в UI — используется в UsersPage и UserDetailPage.
export const roleColor: Record<UserRole, string> = {
    banned: 'red',
    user: 'default',
    admin: 'blue',
    root_admin: 'purple',
}

export const roleLabel: Record<UserRole, string> = {
    banned: 'забанен',
    user: 'обычный',
    admin: 'админ',
    root_admin: 'root admin',
}
