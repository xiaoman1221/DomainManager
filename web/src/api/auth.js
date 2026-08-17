import request from './request'

export const login = (data) => request.post('/auth/login', data)
export const register = (data) => request.post('/auth/register', data)
export const getProfile = () => request.get('/auth/profile')

// OauthGo third-party login (https://o.1v.fit/docs)
export const getOauthProviders = () => request.get('/auth/oauth/providers')
export const oauthLogin = (type) => request.post('/auth/oauth/login', { type })
export const redeemOauthTicket = (ticket) => request.post('/auth/oauth/ticket', { ticket })

// OauthGo account binding (profile page)
export const getOauthBindings = () => request.get('/auth/oauth/bindings')
export const oauthBind = (type) => request.post('/auth/oauth/bind', { type })
export const oauthUnbind = (provider) => request.delete(`/auth/oauth/bind/${provider}`)

// Public site config (footer / SNS) for the login page.
export const getSiteConfig = () => request.get('/site/config')
