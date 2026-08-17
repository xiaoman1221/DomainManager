import request from './request'

export const getSystemSettings = () => request.get('/settings')
// Bulk-updates DB-backed runtime settings (WHOIS/ICP/OauthGo etc.):
// request.put('/settings', { WHOIS_API_URL: '...', ICP_API_URL: '...' })
export const updateSystemSettings = (data) => request.put('/settings', data)
export const getSystemInfo = () => request.get('/settings/info')
export const updateUserProfile = (data) => request.put('/settings/profile', data)
export const changePassword = (data) => request.put('/settings/password', data)
export const getUsers = () => request.get('/settings/users')
export const updateUser = (id, data) => request.put(`/settings/users/${id}`, data)
export const deleteUser = (id) => request.delete(`/settings/users/${id}`)
export const updateUserRoleGroup = (id, data) => request.put(`/settings/users/${id}/role-group`, data)
export const updateUserGroup = (id, data) => request.put(`/settings/users/${id}/user-group`, data)
export const adminUpdatePassword = (id, data) => request.put(`/settings/users/${id}/password`, data)

