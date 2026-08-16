import request from './request'

export const getRegistrars = () => request.get('/registrars')
export const getRegistrar = (id) => request.get(`/registrars/${id}`)
export const createRegistrar = (data) => request.post('/registrars', data)
export const updateRegistrar = (id, data) => request.put(`/registrars/${id}`, data)
export const deleteRegistrar = (id) => request.delete(`/registrars/${id}`)
export const getRegistrarTypes = () => request.get('/registrars/types')
export const importDomains = (data) => request.post('/registrars/import', data)
export const exportRegistrars = () => request.get('/registrars/export', { responseType: 'blob' })
export const importRegistrarsCSV = (formData) => request.post('/registrars/import-csv', formData, { headers: { 'Content-Type': 'multipart/form-data' } })
