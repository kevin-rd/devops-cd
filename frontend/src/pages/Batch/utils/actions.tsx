import React from 'react'
import {Batch, BatchAction, BatchStatus} from "@/types/batch.ts";
import {t} from "i18next";
import {CheckCircleOutlined, FastForwardOutlined, PlayCircleOutlined, StopOutlined,} from "@ant-design/icons";

export type BatchStatistics = {
  total: number
  preAppsCount: number
  skipPreCount: number
  allSkipPre: boolean
}

export type BatchActionMeta = {
  key: string
  label: string
  icon: React.ReactNode
  type?: 'primary' | 'default' | 'link'
  danger?: boolean
  action: BatchAction
}

function getAvailableActions(batch: Batch, statistics: BatchStatistics) {
  const actions: Array<BatchActionMeta> = []
  if (!batch) return actions

  // 审批通过：可以封板
  if (batch.status === BatchStatus.Draft && batch.approval_status === 'approved' && batch.app_count > 0) {
    actions.push({
      key: 'seal',
      label: t('batch.seal'),
      icon: <CheckCircleOutlined/>,
      type: 'primary',
      action: BatchAction.Seal,
    })
  }

  // 已封板：根据是否有pre应用决定显示哪个按钮
  if (batch.status === 10) {
    if (statistics.preAppsCount > 0) {
      // 有需要pre的应用，显示"开始预发布"按钮
      actions.push({
        key: 'start_pre_deploy',
        label: `${t('batch.startPreDeploy')} (${statistics.preAppsCount} 个应用)`,
        icon: <PlayCircleOutlined/>,
        type: 'primary',
        action: BatchAction.StartPreDeploy,
      })
    }

    if (statistics.allSkipPre) {
      // 所有应用都跳过pre，显示"直接开始生产部署"按钮
      actions.push({
        key: 'start_prod_deploy',
        label: `${t('batch.startProdDeploy')} (跳过预发布)`,
        icon: <FastForwardOutlined/>,
        type: 'primary',
        action: BatchAction.StartProdDeploy,
      })
    }
  }


  // Pre 部署完成：可以Pre验收
  if (batch.status === BatchStatus.PreDeployed) {
    actions.push({
      key: 'finish_pre_deploy',
      label: t('batch.finishPreDeploy'),
      icon: <CheckCircleOutlined/>,
      type: 'primary',
      action: BatchAction.ConfirmPre,
    })
  }

  // Pre 验收完成：可以开始Prod部署
  if (batch.status === BatchStatus.PreAccepted) {
    actions.push({
      key: 'start_prod_deploy',
      label: t('batch.startProdDeploy'),
      icon: <PlayCircleOutlined/>,
      type: 'primary',
      action: BatchAction.StartProdDeploy,
    })
  }


  // Prod 部署完成：可以prod验收
  if (batch.status === BatchStatus.ProdDeployed) {
    actions.push({
      key: 'finish_prod_deploy',
      label: t('batch.acceptProd'),
      icon: <CheckCircleOutlined/>,
      type: 'primary',
      action: BatchAction.ConfirmProd,
    })
  }

  // Prod 部署完成：可以最终验收(PM验收)
  if (batch.status === BatchStatus.ProdAccepted) {
    actions.push({
      key: 'complete',
      label: t('batch.acceptFinally'),
      icon: <CheckCircleOutlined/>,
      type: 'primary',
      action: BatchAction.Complete,
    })
  }


  // 未完成且未取消的批次可以取消
  if (batch.status < 40 && batch.status !== BatchStatus.Cancelled) {
    actions.push({
      key: 'cancel',
      label: t('batch.cancelBatch'),
      icon: <StopOutlined/>,
      danger: true,
      type: 'link',
      action: BatchAction.Cancel,
    })
  }

  return actions
}

export default getAvailableActions

