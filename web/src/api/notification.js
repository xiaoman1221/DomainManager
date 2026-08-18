import request from './request'

export const getNotificationTypes = () => request.get('/notifications/types')
export const getNotificationChannels = () => request.get('/notifications/channels')
export const createNotificationChannel = (data) => request.post('/notifications/channels', data)
export const updateNotificationChannel = (id, data) => request.put(`/notifications/channels/${id}`, data)
export const deleteNotificationChannel = (id) => request.delete(`/notifications/channels/${id}`)
export const toggleNotificationChannel = (id) => request.post(`/notifications/channels/${id}/toggle`)
export const testNotificationChannel = (id, data) => request.post(`/notifications/channels/${id}/test`, data)
export const getNotificationLogs = (params) => request.get('/notifications/logs', { params })
export const sendExpiryNotifications = () => request.post('/notifications/send-expiry')

// Scheduled tasks (定时推送系统信息)
export const getScheduledTasks = () => request.get('/notifications/schedules')
export const createScheduledTask = (data) => request.post('/notifications/schedules', data)
export const updateScheduledTask = (id, data) => request.put(`/notifications/schedules/${id}`, data)
export const deleteScheduledTask = (id) => request.delete(`/notifications/schedules/${id}`)
export const runScheduledTask = (id) => request.post(`/notifications/schedules/${id}/run`)
