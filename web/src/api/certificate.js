import request from './request'

export const getCertificates = (params) => request.get('/certificates', { params })
export const getCertificate = (id) => request.get(`/certificates/${id}`)
export const createCertificate = (data) => request.post('/certificates', data)
export const updateCertificate = (id, data) => request.put(`/certificates/${id}`, data)
export const deleteCertificate = (id) => request.delete(`/certificates/${id}`)
export const getCertificateStats = () => request.get('/certificates/stats')
export const getCertimateConfig = () => request.get('/certificates/certimate/config')
export const saveCertimateConfig = (data) => request.post('/certificates/certimate/config', data)
export const syncCertimateCertificates = () => request.post('/certificates/certimate/sync')
