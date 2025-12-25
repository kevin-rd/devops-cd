import request from '@/utils/request'
import type {ApiResponse} from '@/types'

export interface Credential {
  id: number
  scope: 'global' | 'project'
  project_id?: number
  name: string
  type: 'basic_auth' | 'token' | 'ssh_key' | 'tls_client_cert'
  meta_json?: unknown
  created_at: string
  updated_at: string
}

export const credentialService = {
  list: async (params?: {scope?: 'global' | 'project'; project_id?: number}): Promise<ApiResponse<Credential[]>> => {
    return request.get('/v1/credentials', {params})
  },
}


