import {BuildSummary} from "@/types/index.ts";


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
  status: number
  is_locked: boolean
  skip_pre_env: boolean  // 是否跳过预发布环境
  reasons?: string[]
  pre_deploy_status?: string
  prod_deploy_status?: string
  deploy_retry_count?: number
  last_deploy_error?: string
  last_deploy_at?: string
  created_at: string
  updated_at: string
  recent_builds?: BuildSummary[] // 最近的构建记录（自上次部署以来）
  default_depends_on?: number[]
  temp_depends_on?: number[]

  // deployment 级部署详情（由 /v1/release_app 返回）
  deployments?: DeploymentRecord[]
}

export interface DeploymentRecord {
  id: number
  batch_id: number
  release_id: number
  app_id: number

  env: 'pre' | 'prod' | string
  cluster_name: string
  namespace: string
  deployment_name: string
  driver_type?: string | null
  status: 'pending' | 'running' | 'success' | 'failed' | string
  retry_count: number
  max_retry_count: number
  error_message?: string | null

  started_at?: string | null
  finished_at?: string | null
  created_at: string
  updated_at: string
}

export enum AppStatus {
  Pending = 0, // 初始化
  Tagged = 10,  // 应用已发版确认
  PreWaiting = 20, // Pre 等待被触发
  PreCanTrigger = 21, // Pre 可以触发
  PreTriggered = 22, // Pre 均已触发
  PreDeployed = 23, // Pre 均部署完成
  PreFailed = 24,
  PreAccepted = 25, // Pre 已验收
  ProdWaiting = 30, // Prod 等待被触发
  ProdCanTrigger = 31, // Prod 可以触发
  ProdTriggered = 32, // Prod 均已触发
  ProdDeployed = 33, // Prod 均部署完成
  ProdFailed = 34,
  ProdAccepted = 35, // Prod 已验收
}

export interface LabelValue {
  label: string
  color?: string
}

export const AppStatusLabel: Record<number, LabelValue> = {
  [AppStatus.Pending]: {label: '初始化', color: 'gray'},
  [AppStatus.Tagged]: {label: '已确认版本', color: 'blue'},
  [AppStatus.PreWaiting]: {label: 'Pre等待', color: 'blue'},
  [AppStatus.PreCanTrigger]: {label: 'Pre可触发部署', color: 'green'},
  [AppStatus.PreTriggered]: {label: 'Pre部署中', color: 'yellow'},
  [AppStatus.PreDeployed]: {label: 'Pre完成', color: 'green'},
  [AppStatus.PreFailed]: {label: 'Pre失败', color: 'red'},
  [AppStatus.PreAccepted]: {label: 'Pre已验收', color: 'cyan'},
  [AppStatus.ProdWaiting]: {label: 'Prod等待', color: 'blue'},
  [AppStatus.ProdCanTrigger]: {label: 'Prod可触发部署', color: 'green'},
  [AppStatus.ProdTriggered]: {label: 'Prod已触发', color: 'yellow'},
  [AppStatus.ProdDeployed]: {label: 'Prod完成', color: 'green'},
  [AppStatus.ProdFailed]: {label: 'Prod失败', color: 'red'},
  [AppStatus.ProdAccepted]: {label: 'Prod已验收', color: 'cyan'},
}