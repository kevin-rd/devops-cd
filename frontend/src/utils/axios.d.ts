import 'axios'

declare module 'axios' {
  export interface AxiosInstance {
    get<T = any>(url: string, config?: import('axios').AxiosRequestConfig): Promise<T>
    post<T = any>(url: string, data?: any, config?: import('axios').AxiosRequestConfig): Promise<T>
    put<T = any>(url: string, data?: any, config?: import('axios').AxiosRequestConfig): Promise<T>
    delete<T = any>(url: string, config?: import('axios').AxiosRequestConfig): Promise<T>
  }
}