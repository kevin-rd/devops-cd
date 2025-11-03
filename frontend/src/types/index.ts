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

export interface UpdateRepositoryRequest extends Partial<CreateRepositoryRequest> {}

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

export interface UpdateApplicationRequest extends Partial<CreateApplicationRequest> {}

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

// Batch Types
export interface Batch {
  id: number
  batch_number: string
  initiator: string
  release_notes?: string
  approval_status: 'pending' | 'approved' | 'rejected' | 'skipped'
  approved_by?: string
  approved_at?: string
  reject_reason?: string
  status: number // 0:草稿 10:已封板 21:预发布中 22:预发布完成 31:生产部署中 32:生产部署完成 40:已完成 90:已取消
  status_name?: string // 状态名称（列表接口返回）
  app_count?: number // 应用数量（列表接口返回）
  tagged_at?: string
  pre_deploy_started_at?: string
  pre_deploy_finished_at?: string
  prod_deploy_started_at?: string
  prod_deploy_finished_at?: string
  final_accepted_at?: string
  final_accepted_by?: string
  cancelled_at?: string
  cancelled_by?: string
  cancel_reason?: string
  created_at: string
  updated_at: string
  apps?: ReleaseApp[] // 详情接口返回
}

export interface ReleaseApp {
  id: number
  batch_id: number
  app_id: number
  app_name?: string
  app_display_name?: string
  app_type?: string
  app_project?: string
  app_status?: number
  build_id?: number
  latest_build_id?: number
  tag?: string
  version?: string
  image_name?: string
  image_tag?: string
  deployed_tag?: string
  target_tag?: string
  previous_deployed_tag?: string
  repo_id?: number
  repo_name?: string
  repo_full_name?: string
  team_id?: number
  build_number?: number
  build_status?: string
  image_url?: string
  commit_id?: string
  commit_sha?: string
  commit_message?: string
  branch?: string
  commit_branch?: string
  release_notes?: string
  status: string
  is_locked: boolean
  pre_deploy_status?: string
  prod_deploy_status?: string
  deploy_retry_count?: number
  last_deploy_error?: string
  last_deploy_at?: string
  deploy_task_id?: string
  created_at: string
  updated_at: string
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

