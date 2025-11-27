// Batch Types
import {ReleaseApp} from "@/types/release_app.ts";
import {AppTypeConfigInfo} from "@/types/index.ts";

export interface Batch {
  id: number
  batch_number: string
  project_id: number  // 新增：关联的项目ID
  project_name?: string  // 新增：项目名称
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
  total_apps?: number // 应用总数（详情接口分页）
  app_page?: number // 当前页码（详情接口分页）
  app_page_size?: number // 每页数量（详情接口分页）
  app_type_configs?: Record<string, AppTypeConfigInfo>
}

export enum BatchStatus {
  Draft = 0,  // 草稿/准备中
  Sealed = 10, // 已封板
  PreTriggered = 20, // 预发布已触发
  PreDeploying = 21, // 预发布部署中
  PreDeployed = 22, // 预发布已部署完成, 验收中
  ProdTriggered = 30, // 生产已触发
  ProdDeploying = 31, // 生产部署中
  ProdDeployed = 32, // 生产已部署完成, 验收中
  Completed = 40, // 已完成
  FinalAccepted = 40,
  Cancelled = 90, // 已取消
}