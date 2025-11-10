import {Button, Descriptions, message, Modal, Popconfirm, Select, Space, Spin, Tag} from 'antd'
import {useEffect, useState} from 'react'
import {RocketOutlined} from '@ant-design/icons'
import type {BuildSummary} from '@/types'
import {AppStatus, AppStatusLabel, ReleaseApp} from '@/types/release_app.ts';
import {batchService} from '@/services/batch'
import {useAuthStore} from '@/stores/authStore'
import {Batch, BatchStatus} from "@/types/batch.ts";

interface AppNodeModalProps {
  visible: boolean
  releaseApp: ReleaseApp | null
  environment?: 'pre' | 'prod'
  batch: Batch
  onClose: () => void
  onRefresh?: () => void
}

const AppNodeModal: React.FC<AppNodeModalProps> = ({visible, releaseApp, environment, batch, onClose, onRefresh,}) => {
  const [triggering, setTriggering] = useState(false)
  const [loading, setLoading] = useState(false)
  const [detailData, setDetailData] = useState<ReleaseApp | null>(null)
  const [selectedRecentBuildId, setSelectedRecentBuildId] = useState<number | null>(null)
  const {user} = useAuthStore()

  // 当弹窗打开时，加载详细信息
  useEffect(() => {
    if (visible && releaseApp?.id) {
      loadReleaseAppDetail()
    }
  }, [visible, releaseApp?.id])

  const loadReleaseAppDetail = async () => {
    if (!releaseApp?.id) return

    try {
      setLoading(true)
      const response = await batchService.getReleaseApp(releaseApp.id)
      setDetailData(response.data)
      // 默认选择第一个 recent_build
      if (response.data.recent_builds && response.data.recent_builds.length > 0) {
        setSelectedRecentBuildId(response.data.recent_builds[0].id)
      }
    } catch (error: any) {
      message.error(error.response?.data?.message || '加载应用详情失败')
      // 如果加载失败，使用传入的数据
      setDetailData(releaseApp)
    } finally {
      setLoading(false)
    }
  }

  if (!releaseApp) return null

  // 使用详细数据或传入的数据
  const displayData = detailData || releaseApp

  const handleTriggerDeploy = async (action: string) => {
    if (!batch || !environment) {
      message.error('缺少批次ID或环境信息')
      return
    }

    try {
      setTriggering(true)
      const response = await batchService.manualDeploy({
        batch_id: batch.id,
        release_app_id: displayData.id,
        action: action,
        operator: user?.username || 'unknown',
        reason: '手动触发部署',
      })

      message.success(response.data.message || '触发部署成功')
      onClose()
      onRefresh?.()
    } catch (error: any) {
      message.error(error.response?.data?.message || '触发部署失败')
    } finally {
      setTriggering(false)
    }
  }


  // 渲染手动部署按钮
  const renderManualDeployButton = (releaseApp: ReleaseApp, batch: Batch) => {
    if (!batch || !releaseApp) return null

    // batch已封板状态, release可以提前Pre
    if (batch.status >= BatchStatus.Sealed && batch.status < BatchStatus.PreWaiting) {
      if (releaseApp.status === AppStatus.Tagged) {
        return (
          <Popconfirm title="确认提前发布" description={releaseApp.app_name + " => " + releaseApp.target_tag}
                      disabled={triggering} onConfirm={() => handleTriggerDeploy('manual_trigger_pre')}>
            <Button danger icon={<RocketOutlined/>} disabled={loading}>提前[Pre]发布</Button>
          </Popconfirm>
        )
      }
    }
    // batch已经开始预发布, release可以提前Prod
    if (batch.status >= BatchStatus.Sealed && batch.status < BatchStatus.ProdWaiting) {
      if (releaseApp.status == AppStatus.PreDeployed) {
        return (
          <Popconfirm title="确认提前发布" description={releaseApp.app_name + " => " + releaseApp.target_tag}
                      onConfirm={() => handleTriggerDeploy("manual_trigger_prod")}>
            <Button danger icon={<RocketOutlined/>} disabled={loading}>提前[Prod]发布</Button>
          </Popconfirm>
        )
      }
    }
    // 重新发布过, 可以继续发Prod
    if (batch.status >= BatchStatus.Sealed && batch.status < BatchStatus.Completed) {
      if (releaseApp.status == AppStatus.PreDeployed) {
        return (
          <Popconfirm title="确认发布" description={releaseApp.app_name + " => " + releaseApp.target_tag}
                      onConfirm={() => handleTriggerDeploy("manual_trigger_prod")}>
            <Button danger icon={<RocketOutlined/>} disabled={loading}>触发[Prod]发布</Button>
          </Popconfirm>
        )
      }
    }

  }


  // 重新发布（直接使用选中的 recent_build）
  const handleSwitchVersion = async (buildId: number) => {
    if (!batch || !environment) {
      message.error('缺少批次ID或环境信息')
      return
    }

    try {
      setTriggering(true)
      const response = await batchService.switchVersion({
        batch_id: batch?.id,
        release_app_id: displayData.id,
        environment,
        operator: user?.username || 'unknown',
        build_id: buildId,
        reason: '用户触发手动发布',
      })

      message.success(response.message || '触发部署成功')
      onClose()
      onRefresh?.()
    } catch (error: any) {
      message.error(error.response?.message || '触发部署失败')
      console.error(error)
    } finally {
      setTriggering(false)
    }
  }


  const hasNewTag = displayData.latest_build_id &&
    displayData.latest_build_id !== displayData.build_id

  // 获取状态显示文本
  const getStatusText = (status: string | number) => {
    const statusNum = typeof status === 'string' ? parseInt(status, 10) : status
    return AppStatusLabel[statusNum] || `状态${statusNum}`
  }

  return (
    <>
      <Modal
        title={
          <Space>
            <span style={{color: '#999', fontSize: 13}}>#{displayData.id}</span>
            <span>{displayData.app_name}</span>
            {displayData.app_type && <Tag color="blue">{displayData.app_type}</Tag>}
            {hasNewTag && <Tag color="orange">有新版本</Tag>}
          </Space>
        }
        open={visible}
        onCancel={onClose}
        width={650}
        footer={
          <div style={{display: 'flex', justifyContent: 'space-between', alignItems: 'center'}}>
            <span style={{fontSize: 12, color: '#999'}}>
              提示：底部"手动触发"按钮用于批次未开始时手动触发部署
            </span>
            <Space>
              <Button onClick={onClose}>关闭</Button>
              {renderManualDeployButton(displayData, batch)}
            </Space>
          </div>
        }
      >
        <Spin spinning={loading}>
          {/* 基本信息 */}
          <Descriptions column={2} size="small" bordered>
            <Descriptions.Item label="应用名称" span={2}>
              {displayData.app_name}
            </Descriptions.Item>
            <Descriptions.Item label="之前版本">
              <Tag>{displayData.previous_deployed_tag || '-'}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="当前状态">
              <Tag>{getStatusText(displayData.status)}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="目标版本" span={2}>
              <Space>
                {displayData.build_id && (
                  <span style={{fontSize: 12, color: '#999'}}>#{displayData.build_id}</span>
                )}
                {displayData.target_tag || '-'}
                <span style={{color: '#999'}}>{displayData.commit_message}</span>
              </Space>
            </Descriptions.Item>

            {/* 最新 Build 信息 - 根据 recent_builds 数量显示不同的UI */}
            <Descriptions.Item label="最新Build" span={2}>
              {displayData.recent_builds && displayData.recent_builds.length > 0 && (
                <div style={{display: 'flex', alignItems: 'center', gap: 8}}>
                  {displayData.recent_builds.length === 1 ? (
                    // 只有一个 recent_build，直接显示
                    <>
                      <span style={{fontSize: 12, color: '#999'}}>#{displayData.recent_builds[0].id}</span>
                      <Tag color="red">{displayData.recent_builds[0].image_tag}</Tag>
                      <span style={{color: '#ff4d4f'}}>有新版本可用</span>
                    </>
                  ) : (
                    // 多个 recent_builds，显示下拉列表
                    <>
                      <Select
                        value={selectedRecentBuildId}
                        onChange={setSelectedRecentBuildId}
                        style={{minWidth: 300}}
                        placeholder="选择构建版本"
                      >
                        {displayData.recent_builds.map((build: BuildSummary) => (
                          <Select.Option key={build.id} value={build.id}>
                            <Space>
                              <span style={{fontSize: 12, color: '#999'}}>#{build.id}</span>
                              <Tag color="blue">{build.image_tag}</Tag>
                              <span style={{fontSize: 12, color: '#666'}}>
                              {build.commit_message?.substring(0, 30)}
                                {(build.commit_message?.length || 0) > 30 ? '...' : ''}
                            </span>
                            </Space>
                          </Select.Option>
                        ))}
                      </Select>
                      {/*<span style={{ color: '#ff4d4f' }}>有新版本可用</span>*/}
                    </>
                  )}
                  <Button
                    type="primary"
                    size="small"
                    icon={<RocketOutlined/>}
                    onClick={() => {
                      if (selectedRecentBuildId) {
                        handleSwitchVersion(selectedRecentBuildId).then()
                      } else {
                        message.warning('请先选择要部署的构建版本')
                      }
                    }}
                    style={{marginLeft: 'auto'}}
                  >重新发布
                  </Button>
                </div>
              )}
            </Descriptions.Item>
          </Descriptions>

          {/* 提交信息 */}
          <Descriptions
            title="提交信息"
            column={1}
            size="small"
            bordered
            style={{marginTop: 16}}
          >
            <Descriptions.Item label="Commit SHA">
              <code>{displayData.commit_sha?.substring(0, 8) || '-'}</code>
            </Descriptions.Item>
            <Descriptions.Item label="Commit Message">
              {displayData.commit_message || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="分支">
              {displayData.commit_branch || '-'}
            </Descriptions.Item>
          </Descriptions>

          {/* 部署信息 */}
          {(displayData.last_deploy_at || displayData.last_deploy_error) && (
            <Descriptions
              title="部署信息"
              column={1}
              size="small"
              bordered
              style={{marginTop: 16}}
            >
              {displayData.last_deploy_at && (
                <Descriptions.Item label="上次部署时间">
                  {displayData.last_deploy_at}
                </Descriptions.Item>
              )}
              {displayData.last_deploy_error && (
                <Descriptions.Item label="上次部署错误">
                  <span style={{color: '#ff4d4f'}}>{displayData.last_deploy_error}</span>
                </Descriptions.Item>
              )}
            </Descriptions>
          )}
        </Spin>
      </Modal>

    </>
  )
}

export default AppNodeModal
