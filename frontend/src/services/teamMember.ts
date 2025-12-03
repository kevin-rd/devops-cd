import request from '@/utils/request'
import type {ApiResponse, PaginatedResponse, PaginationParams} from '@/types'

// TeamMember Types
export interface TeamMember {
  id: number
  team_id: number
  user_id: number
  roles: string[]
  username: string
  display_name?: string
  email?: string
  created_at: string
  updated_at: string
}

export interface CreateTeamMemberRequest {
  team_id: number
  user_id: number
  roles?: string[]
}

export interface UpdateTeamMemberRoleRequest {
  roles: string[]
}

export interface TeamMemberListQuery extends PaginationParams {
  team_id: number
  keyword?: string
}

// TeamMember Service
export const teamMemberService = {
  // 添加团队成员
  add: (data: CreateTeamMemberRequest) => {
    return request.post<ApiResponse<TeamMember>>('/v1/team_members', data)
  },

  // 获取团队成员列表
  getList: (params: TeamMemberListQuery) => {
    return request.get<ApiResponse<PaginatedResponse<TeamMember>>>('/v1/team_members', {
      params,
    })
  },

  // 更新成员角色
  updateRole: (id: number, data: UpdateTeamMemberRoleRequest) => {
    return request.put<ApiResponse<TeamMember>>(`/v1/team_members/${id}/role`, data)
  },

  // 删除成员
  delete: (id: number) => {
    return request.delete<ApiResponse<void>>(`/v1/team_members/${id}`)
  },
}

