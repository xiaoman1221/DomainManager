import request from './request'

export const getSystemSettings = () => request.get('/settings')
export const updateSystemSetting = (data) => request.put('/settings', data)
export const getSystemInfo = () => request.get('/settings/info')
export const updateUserProfile = (data) => request.put('/settings/profile', data)
export const changePassword = (data) => request.put('/settings/password', data)
export const getUsers = () => request.get('/settings/users')
export const updateUserRole = (id, data) => request.put(`/settings/users/${id}/role`, data)
export const adminUpdatePassword = (id, data) => request.put(`/settings/users/${id}/password`, data)
