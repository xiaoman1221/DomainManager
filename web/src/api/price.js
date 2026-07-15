import request from './request'

export const comparePrices = (data) => request.post('/price/compare', data)

export const getSupportedTLDs = () => request.get('/price/tlds')

export const refreshPrices = (domain) =>
  request.post('/price/refresh', null, { params: { domain } })
