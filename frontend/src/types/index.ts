// API Response Types

export interface ApiResponse<T = any> {
  code: number
  message: string
  data: T
}

export interface PaginationParams {
  page?: number
  page_size?: number
}

export interface PaginatedResponse<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}

// Auth Types
export interface LoginRequest {
  username: string
  password: string
  auth_type: 'local' | 'ldap'
}

export interface LoginResponse {
  access_token: string
  refresh_token: string
  expires_in: number
  user: UserInfo
}

export interface UserInfo {
  username: string
  email: string
  display_name: string
  auth_type: 'local' | 'ldap'
}

// Repository Types
export interface Repository {
  id: number
  name: string
  project: string
  description: string
  git_url: string
  git_type: string
  default_branch: string
  language: string
  team_id: number
  team_name: string
  status: number
  created_at: string
  updated_at: string
  applications?: Application[]  // 新增：可选的应用列表
}

export interface CreateRepositoryRequest {
  name: string
  project: string
  description?: string
  git_url: string
  git_type: string
  git_token?: string
  default_branch?: string
  language?: string
  team_id?: number
}

export interface UpdateRepositoryRequest extends Partial<CreateRepositoryRequest> {
}

// Application Types
export interface Application {
  id: number
  name: string
  display_name: string
  description: string
  repo_id: number
  repo_name: string
  project: string
  app_type: string
  team_id: number
  team_name: string
  last_tag: string
  status: number
  created_at: string
  updated_at: string
}

export interface CreateApplicationRequest {
  name: string
  display_name?: string
  description?: string
  repo_id: number
  project: string
  app_type: string
  team_id?: number
}

export interface UpdateApplicationRequest extends Partial<CreateApplicationRequest> {
}

// Query Params
export interface RepositoryQueryParams extends PaginationParams {
  name?: string
  git_type?: string
  team_id?: number
  status?: number
  project?: string
  keyword?: string
  with_applications?: boolean  // 新增：是否包含应用列表
}

export interface ApplicationQueryParams extends PaginationParams {
  name?: string
  repo_id?: number
  project?: string
  app_type?: string
  team_id?: number
  status?: number
  keyword?: string
}

// Application with latest build info (flattened structure)
export interface ApplicationWithBuild extends Application {
  deployed_tag?: string | null
  repo_full_name?: string  // Repository的project/name
  build_id?: number
  build_number?: number
  image_tag?: string
  commit_sha?: string
  commit_message?: string | null
  commit_branch?: string
  build_status?: string
}

// Build Types
export interface Build {
  id: number
  repo_id: number
  repo_name?: string
  app_id: number
  app_name?: string
  build_number: number
  build_status: 'success' | 'failure' | 'error' | 'killed'
  build_event: 'push' | 'tag' | 'pull_request' | 'promote' | 'rollback'
  build_link: string
  commit_sha: string
  commit_ref: string
  commit_branch: string
  commit_message: string
  commit_link: string
  commit_author: string
  build_created: number
  build_started: number
  build_finished: number
  build_duration: number
  image_tag: string
  image_url: string
  app_build_success: boolean
  environment: string
  created_at: string
  updated_at: string
}

export interface BuildQueryParams extends PaginationParams {
  repo_id?: number
  app_id?: number
  build_status?: string
  build_event?: string
  image_tag?: string
  commit_sha?: string
  environment?: string
  keyword?: string
  start_time?: string
  end_time?: string
}

// Application Type
export interface ApplicationType {
  value: string
  label: string
  description: string
  icon: string
  color: string
}

export interface AppTypeConfigInfo {
  label: string
  description?: string
  icon?: string
  color?: string
  dependencies?: string[]
}

// 构建摘要（用于 recent_builds）
export interface BuildSummary {
  id: number
  build_number: number
  build_status: string
  image_tag: string
  commit_sha: string
  commit_message: string
  commit_author: string
  build_created: string
}


export interface CreateBatchRequest {
  batch_number: string
  initiator: string
  release_notes?: string
  apps: {
    app_id: number
    release_notes?: string
  }[]
}

export interface UpdateBatchRequest {
  batch_id: number
  operator: string
  batch_number?: string
  release_notes?: string
  add_apps?: {
    app_id: number
    release_notes?: string
  }[]
  remove_app_ids?: number[]
}

export interface BatchQueryParams extends PaginationParams {
  status?: number | number[]
  approval_status?: string
  initiator?: string
  keyword?: string
  start_time?: string
  end_time?: string
}

export interface BatchActionRequest {
  batch_id: number
  action: 'seal' | 'start_pre_deploy' | 'finish_pre_deploy' | 'start_prod_deploy' | 'finish_prod_deploy' | 'complete' | 'cancel'
  operator: string
  reason?: string
}

export interface UpdateReleaseDependenciesRequest {
  batch_id: number
  operator: string
  temp_depends_on: number[]
}

export interface ReleaseDependenciesResponse {
  batch_id: number
  release_app_id: number
  app_id: number
  default_depends_on: number[]
  temp_depends_on: number[]
  updated_at: string
}

export interface BatchApproveRequest {
  batch_id: number
  operator: string
  reason?: string
}

export interface BatchRejectRequest {
  batch_id: number
  operator: string
  reason: string
}

export interface SwitchVersionRequest {
  batch_id: number
  release_app_id: number
  environment: 'pre' | 'prod'
  operator: string
  build_id: number
  reason?: string
}

export interface TriggerDeployRequest {
  batch_id: number
  release_app_id: number
  action: string
  operator: string
  reason?: string
}

export interface TriggerDeployResponse {
  release_app_id: number
  app_id: number
  app_name: string
  environment: string
  old_status: number
  new_status: number
  old_build_id?: number
  new_build_id?: number
  old_tag: string
  new_tag: string
  message: string
}

