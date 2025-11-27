import {Steps, Tag, Tooltip} from 'antd'
import {CheckCircleOutlined, FastForwardOutlined, LoadingOutlined, PlayCircleOutlined} from '@ant-design/icons'
import dayjs from 'dayjs'
import {useTranslation} from 'react-i18next'
import {useEffect, useState} from 'react'
import {Batch, BatchStatus} from '@/types/batch.ts'
import {formatCreatedTime} from '@/utils/time'
import './index.css'

interface BatchTimelineProps {
  batch: Batch
  onAction?: (action: string) => void
  hasPreApps?: boolean  // 是否有需要预发布的应用
}

export const BatchTimeline: React.FC<BatchTimelineProps> = ({batch, onAction, hasPreApps = true}) => {
  const {t} = useTranslation()
  const [isVertical, setIsVertical] = useState(false)

  // 检测屏幕宽度，在小屏幕时切换为纵向布局
  useEffect(() => {
    const handleResize = () => {
      setIsVertical(window.innerWidth <= 576)
    }

    handleResize() // 初始化
    window.addEventListener('resize', handleResize)
    return () => window.removeEventListener('resize', handleResize)
  }, [])

  // 根据批次状态计算当前步骤
  const getCurrentStep = () => {
    if (batch.status === 90) {
      // 已取消 - 返回取消前的最后步骤
      if (batch.prod_deploy_started_at) return hasPreApps ? 3 : 2
      if (batch.pre_deploy_started_at) return 2
      if (batch.tagged_at) return 1
      return 0
    }
    if (batch.status >= 40) return hasPreApps ? 4 : 3 // 已完成
    if (batch.status >= 32) return hasPreApps ? 3 : 2 // 生产部署完成
    if (batch.status >= 30) return hasPreApps ? 3 : 2 // 生产待触发/部署中
    if (batch.status >= 22) return 2 // 预发布完成
    if (batch.status >= 20) return 2 // 预发布待触发/部署中
    if (batch.status >= 10) return 1 // 已封板
    return 0 // 待封板或草稿
  }

  // 根据批次状态确定步骤状态
  const getStepStatus = (stepIndex: number): 'wait' | 'process' | 'finish' | 'error' => {
    const currentStep = getCurrentStep()
    if (stepIndex < currentStep) return 'finish'
    if (stepIndex === currentStep) {
      // 如果是部署中或待触发的步骤，显示 process
      if ((batch.status === 20 || batch.status === 21) && stepIndex === 2) {
        return 'process'
      }
      if ((batch.status === 30 || batch.status === 31) && stepIndex === (hasPreApps ? 3 : 2)) {
        return 'process'
      }
      return 'finish'
    }
    return 'wait'
  }

  const formatTime = (time?: string) => {
    return time ? dayjs(time).format('MM-DD HH:mm') : '-'
  }

  // 获取预发布步骤的时间描述（显示开始和完成时间，分两行）
  const getPreDeployTimeDescription = () => {
    const lines: string[] = []
    if (batch.pre_deploy_started_at) {
      lines.push(`${t('common.start')}: ${formatTime(batch.pre_deploy_started_at)}`)
    }
    if (batch.pre_deploy_finished_at) {
      lines.push(`${t('common.finish')}: ${formatTime(batch.pre_deploy_finished_at)}`)
    }
    return lines.length > 0 ? lines : ['-']
  }

  // 获取预发布步骤的状态描述（用于 subTitle，带样式类）
  const getPreDeployStatusText = () => {
    if (batch.pre_deploy_finished_at) {
      return {text: t('batch.statusCompleted'), className: 'status-tag status-finished'}
    }
    if (batch.status === 21) {
      return {text: t('batch.statusPreDeploying'), className: 'status-tag status-pending'}
    }
    if (batch.status === 20) {
      return {text: t('batch.statusPreTriggered'), className: 'status-tag status-pending'}
    }
    return {text: '-', className: ''}
  }


  const preDeployStatus = getPreDeployStatusText()
  const preDeployTimes = getPreDeployTimeDescription()

  // 获取自定义图标（用于可点击和进行中状态）
  const getCustomIcon = (stepIndex: number) => {
    // 封板步骤 (index 1)
    if (stepIndex === 1) {
      // 如果还没封板，显示可点击的封板图标
      if (!batch.tagged_at && onAction) {
        return (
          <Tooltip title={t('batch.seal')}>
            <CheckCircleOutlined
              className="timeline-icon-clickable"
              onClick={(e) => {
                e.stopPropagation()
                onAction('seal')
              }}
            />
          </Tooltip>
        )
      }
    }

    // 预发布步骤 (index 2) - 只在有预发布应用时显示
    if (stepIndex === 2 && hasPreApps) {
      // 如果正在预发布中，显示转圈图标
      if (batch.status === BatchStatus.PreTriggered || batch.status === BatchStatus.PreDeploying) {
        return <LoadingOutlined className="timeline-icon-loading"/>
      }
      // 如果已封板且未开始预发布，显示可点击的开始图标
      if (batch.status === 10 && onAction) {
        return (
          <Tooltip title={t('batch.startPreDeploy')}>
            <PlayCircleOutlined
              className="timeline-icon-clickable"
              onClick={(e) => {
                e.stopPropagation()
                onAction('start_pre_deploy')
              }}
            />
          </Tooltip>
        )
      }
    }

    // 生产部署步骤 (index 3 for hasPreApps, index 2 for !hasPreApps)
    const prodStepIndex = hasPreApps ? 3 : 2
    if (stepIndex === prodStepIndex) {
      // 如果正在生产部署中，显示转圈图标
      if (batch.status === BatchStatus.ProdTriggered || batch.status === BatchStatus.ProdDeploying) {
        return <LoadingOutlined className="timeline-icon-loading"/>
      }
      // 如果预发布完成且未开始生产部署，或者跳过pre已封板，显示可点击的开始图标
      if (((hasPreApps && batch.status === BatchStatus.PreDeployed) || (!hasPreApps && batch.status === BatchStatus.Sealed)) && onAction) {
        return (
          <Tooltip title={t('batch.startProdDeploy')}>
            <PlayCircleOutlined
              className="timeline-icon-clickable"
              onClick={(e) => {
                e.stopPropagation()
                onAction('start_prod_deploy')
              }}
            />
          </Tooltip>
        )
      }
    }

    return undefined
  }

  const createdTimeInfo = formatCreatedTime(batch.created_at)

  // 构建时间线步骤
  const steps = [
    {
      title: t('batch.timelineCreate'),
      description: (
        <div>
          <div>{formatTime(batch.created_at)}</div>
          <div style={{fontSize: '12px', color: '#999', marginTop: '4px'}}>{createdTimeInfo.dayInfo}</div>
        </div>
      ),
      subTitle: batch.initiator,
    },
    {
      title: t('batch.timelineSeal'),
      description: formatTime(batch.tagged_at),
      subTitle: batch.tagged_at ? t('batch.statusSealed') : '-',
    },
  ]

  // 根据是否有预发布应用，动态添加预发布步骤
  if (hasPreApps) {
    steps.push({
      title: t('batch.timelinePreDeploy'),
      description: (
        <div className="timeline-step-description">
          {preDeployTimes.map((line, index) => (
            <div key={index} className="timeline-times">{line}</div>
          ))}
        </div>
      ),
      // @ts-ignore
      subTitle: preDeployStatus.text !== '-' ? (
        <Tag className={preDeployStatus.className}>
          {preDeployStatus.text}
        </Tag>
      ) : '-',
    })
  } else {
    // 没有预发布应用，显示跳过标记
    steps.push({
      title: '跳过预发布',
      description: '所有应用直接部署到生产',
      // @ts-ignore
      subTitle: (
        <Tag color="orange" icon={<FastForwardOutlined/>}>
          已跳过
        </Tag>
      ),
    })
  }

  // 添加生产部署步骤
  // @ts-ignore
  steps.push(batch.status >= BatchStatus.ProdTriggered ? (
      {
        title: t('batch.timelineProdDeploy'),
        subTitle:
          <Tag className={batch.status === BatchStatus.ProdDeployed ? 'status-tag status-finished' :
            batch.status === BatchStatus.ProdDeploying ? 'status-tag status-processing' :
              batch.status === BatchStatus.ProdTriggered ? 'status-tag status-processing' : ''}>
            {batch.status === BatchStatus.ProdDeployed ? t('batch.statusCompleted') :
              batch.status === BatchStatus.ProdDeploying ? t('batch.statusProdDeploying') :
                batch.status === BatchStatus.ProdTriggered ? t('batch.statusProdTriggered') : '-'}
          </Tag>,
        description: (
          <div className="timeline-step-description">
            <div className="timeline-times">{t('common.start')}: {formatTime(batch.prod_deploy_started_at) || '-'}</div>
            <div className="timeline-times">{t('common.finish')}: {formatTime(batch.prod_deploy_finished_at) || '-'}</div>
          </div>
        ),
      }) : ({
      title: t('batch.timelineProdDeploy'),
      subTitle: '-',
      description: formatTime(batch.prod_deploy_started_at),
    }),
    {
      title: t('batch.timelineAccept'),
      description: formatTime(batch.final_accepted_at),
      subTitle: batch.final_accepted_by || '-',
    }
  )

  return (
    <div className="batch-timeline">
      <Steps
        current={getCurrentStep()}
        direction={isVertical ? 'vertical' : 'horizontal'}
        items={steps.map((step, index) => {
          const customIcon = getCustomIcon(index)
          return {
            ...step,
            status: getStepStatus(index),
            icon: customIcon,
          }
        })}
      />
    </div>
  )
}

