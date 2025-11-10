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
  reason?: string
  pre_deploy_status?: string
  prod_deploy_status?: string
  deploy_retry_count?: number
  last_deploy_error?: string
  last_deploy_at?: string
  deploy_task_id?: string
  created_at: string
  updated_at: string
  recent_builds?: BuildSummary[] // 最近的构建记录（自上次部署以来）
  default_depends_on?: number[]
  temp_depends_on?: number[]
}

export enum AppStatus {
  Pending = 0, // 初始化
  Tagged = 10,  // 应用已发版确认
  PreWaiting = 20, // Pre 等待被触发
  PreCanTrigger = 21, // Pre 可以触发
  PreTriggered = 22, // Pre 均已触发
  PreDeployed = 23, // Pre 均部署完成
  PreFailed = 24,
  ProdWaiting = 30, // Prod 等待被触发
  ProdCanTrigger = 31, // Prod 可以触发
  ProdTriggered = 32, // Prod 均已触发
  ProdDeployed = 33, // Prod 均部署完成
  ProdFailed = 34,
}

export const AppStatusLabel: Record<number, string> = {
  [AppStatus.Pending]: '待处理',
  [AppStatus.Tagged]: '已确认版本',
  [AppStatus.PreWaiting]: '预发布等待',
  [AppStatus.PreCanTrigger]: '预发布可触发',
  [AppStatus.PreTriggered]: '预发布已触发',
  [AppStatus.PreDeployed]: '预发布完成',
  [AppStatus.PreFailed]: '预发布失败',
  [AppStatus.ProdWaiting]: '生产等待',
  [AppStatus.ProdCanTrigger]: '生产可触发',
  [AppStatus.ProdTriggered]: '生产已触发',
  [AppStatus.ProdDeployed]: '生产完成',
  [AppStatus.ProdFailed]: '生产失败',
}