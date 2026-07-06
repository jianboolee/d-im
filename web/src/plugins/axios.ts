import axios from 'axios'
import {config} from '../config'

const instance = axios.create({
  baseURL: config.baseURL,
  timeout: 15000,
  withCredentials: true,
  headers: { 'X-Custom-Header': 'x-rv' }
})

// 可以在这里添加请求拦截器
instance.interceptors.request.use(config => {
  // 在发送请求之前做些什么
  return config
}, error => {
  // 对请求错误做些什么
  return Promise.reject(error)
})

// 可以在这里添加响应拦截器
instance.interceptors.response.use(response => {
  // 对响应数据做点什么
  return response
}, error => {
  // 对响应错误做点什么
  return Promise.reject(error)
})

export default instance
