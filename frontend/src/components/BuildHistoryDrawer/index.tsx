import React, { useState } from 'react'
import {
  Drawer,
  Table,
  Tag,
  Space,
  Button,
  Input,
  Select,
  DatePicker,
  Tooltip,
  Empty,
} from 'antd'
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  WarningOutlined,
  StopOutlined,
  LinkOutlined,
  SearchOutlined,
  ReloadOutlined,
  CloseOutlined,
} from '@ant-design/icons'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import 'dayjs/locale/zh-cn'
import buildService from '@/services/build'
import type { Build, BuildQueryParams } from '@/types'
import type { ColumnsType } from 'antd/es/table'
import './index.css'

dayjs.extend(relativeTime)

const { RangePicker } = DatePicker

interface BuildHistoryDrawerProps {
  open: boolean
  appId: number | null
  appName: string
  onClose: () => void
}

const BuildHistoryDrawer: React.FC<BuildHistoryDrawerProps> = ({
  open,
  appId,
  appName,
  onClose,
}) => {
  const { t, i18n } = useTranslation()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  
  // 筛选条件
  const [filters, setFilters] = useState<Partial<BuildQueryParams>>({})

  // 查询构建列表
  const { data: buildResponse, isLoading, refetch } = useQuery({
    queryKey: ['builds', appId, page, pageSize, filters],
    queryFn: async () => {
      if (!appId) return null
      const res = await buildService.getList({
        app_id: appId,
        page,
        page_size: pageSize,
        ...filters,
      })
      return res.data
    },
    enabled: open && !!appId,
  })

  const buildData = buildResponse?.items || []
  const total = buildResponse?.total || 0

  // 重置筛选
  const handleResetFilters = () => {
    setFilters({})
    setPage(1)
  }

  // 构建状态渲染
  const renderStatus = (status: string) => {
    const statusConfig = {
      success: {
        color: 'success',
        icon: <CheckCircleOutlined />,
        text: t('build.statusSuccess'),
      },
      failure: {
        color: 'error',
        icon: <CloseCircleOutlined />,
        text: t('build.statusFailure'),
      },
      error: {
        color: 'warning',
        icon: <WarningOutlined />,
        text: t('build.statusError'),
      },
      killed: {
        color: 'default',
        icon: <StopOutlined />,
        text: t('build.statusKilled'),
      },
    }

    const config = statusConfig[status as keyof typeof statusConfig] || statusConfig.error

    return (
      <Tag color={config.color} icon={config.icon}>
        {config.text}
      </Tag>
    )
  }

  // 构建事件渲染
  const renderEvent = (event: string) => {
    const eventConfig = {
      push: { color: 'blue', text: t('build.eventPush') },
      tag: { color: 'green', text: t('build.eventTag') },
      pull_request: { color: 'purple', text: t('build.eventPullRequest') },
      promote: { color: 'orange', text: t('build.eventPromote') },
      rollback: { color: 'red', text: t('build.eventRollback') },
    }

    const config = eventConfig[event as keyof typeof eventConfig] || { color: 'default', text: event }

    return <Tag color={config.color}>{config.text}</Tag>
  }

  // 格式化耗时
  const formatDuration = (seconds: number) => {
    if (seconds < 60) return `${seconds}${t('build.second')}`
    if (seconds < 3600) return `${Math.floor(seconds / 60)}${t('build.minute')}`
    return `${Math.floor(seconds / 3600)}${t('build.hour')}`
  }

  // 格式化相对时间
  const formatRelativeTime = (timestamp: number) => {
    dayjs.locale(i18n.language === 'zh' ? 'zh-cn' : 'en')
    return dayjs.unix(timestamp).fromNow()
  }

  // 表格列定义
  const columns: ColumnsType<Build> = [
    {
      title: t('build.number'),
      dataIndex: 'build_number',
      key: 'build_number',
      width: 60,
      render: (num: number, record: Build) => (
        <Space size={4}>
          <strong>#{num}</strong>
          {record.build_link && (
            <a
              href={record.build_link}
              target="_blank"
              rel="noopener noreferrer"
              onClick={(e) => e.stopPropagation()}
            >
              <LinkOutlined style={{ fontSize: 11 }} />
            </a>
          )}
        </Space>
      ),
    },
    {
      title: t('build.status'),
      dataIndex: 'build_status',
      key: 'build_status',
      width: 60,
      render: renderStatus,
    },
    {
      title: t('build.event'),
      dataIndex: 'build_event',
      key: 'build_event',
      width: 60,
      render: renderEvent,
    },
    {
      title: t('build.tag'),
      dataIndex: 'image_tag',
      key: 'image_tag',
      width: 100,
      ellipsis: true,
      render: (tag: string) => (
        <Tooltip title={tag}>
          <Tag color="cyan">{tag || '-'}</Tag>
        </Tooltip>
      ),
    },
    {
      title: t('build.environment'),
      dataIndex: 'environment',
      key: 'environment',
      width: 60,
      render: (env: string) => (
        <Tag color={env === 'production' ? 'red' : env === 'staging' ? 'orange' : 'blue'}>
          {env || '-'}
        </Tag>
      ),
    },
    {
      title: t('build.commitMessage'),
      dataIndex: 'commit_message',
      key: 'commit_message',
      ellipsis: true,
      width: 160,
      render: (msg: string, record: Build) => (
        <Tooltip title={msg}>
          <div className="commit-info">
            <span className="commit-message">{msg || '-'}</span>
            {record.commit_link && (
              <a
                href={record.commit_link}
                target="_blank"
                rel="noopener noreferrer"
                onClick={(e) => e.stopPropagation()}
                className="commit-link"
              >
                <LinkOutlined />
              </a>
            )}
          </div>
        </Tooltip>
      ),
    },
    {
      title: t('build.author'),
      dataIndex: 'commit_author',
      key: 'commit_author',
      width: 60,
      ellipsis: true,
    },
    {
      title: t('build.duration'),
      dataIndex: 'build_duration',
      key: 'build_duration',
      width: 60,
      render: formatDuration,
    },
    {
      title: t('build.startTime'),
      dataIndex: 'build_started',
      key: 'build_started',
      width: 60,
      sorter: (a, b) => b.build_started - a.build_started,
      defaultSortOrder: 'ascend',
      render: (timestamp: number) => (
        <Tooltip title={dayjs.unix(timestamp).format('YYYY-MM-DD HH:mm:ss')}>
          {formatRelativeTime(timestamp)}
        </Tooltip>
      ),
    },
  ]

  return (
    <Drawer
      title={
        <div className="drawer-title">
          <span>{appName} - {t('build.history')}</span>
          <Space>
            <Button
              type="text"
              size="small"
              icon={<ReloadOutlined />}
              onClick={() => refetch()}
              loading={isLoading}
            />
            <Button
              type="text"
              size="small"
              icon={<CloseOutlined />}
              onClick={onClose}
              className="drawer-close-btn"
            />
          </Space>
        </div>
      }
      width="65%"
      open={open}
      onClose={onClose}
      closeIcon={null}
      maskClosable={true}
      className="build-history-drawer"
    >
      {/* 筛选器 */}
      <div className="filter-section">
        <Space wrap size="small">
          <Select
            placeholder={t('build.filterByStatus')}
            style={{ width: 110 }}
            allowClear
            size="small"
            value={filters.build_status}
            onChange={(value) => {
              setFilters({ ...filters, build_status: value })
              setPage(1)
            }}
            options={[
              { label: t('build.statusSuccess'), value: 'success' },
              { label: t('build.statusFailure'), value: 'failure' },
              { label: t('build.statusError'), value: 'error' },
              { label: t('build.statusKilled'), value: 'killed' },
            ]}
          />

          <Select
            placeholder={t('build.filterByEvent')}
            style={{ width: 110 }}
            allowClear
            size="small"
            value={filters.build_event}
            onChange={(value) => {
              setFilters({ ...filters, build_event: value })
              setPage(1)
            }}
            options={[
              { label: 'push', value: 'push' },
              { label: 'tag', value: 'tag' },
              { label: 'pull_request', value: 'pull_request' },
              { label: 'promote', value: 'promote' },
              { label: 'rollback', value: 'rollback' },
            ]}
          />

          <Input
            placeholder={t('build.filterByEnvironment')}
            style={{ width: 110 }}
            allowClear
            size="small"
            value={filters.environment}
            onChange={(e) => {
              setFilters({ ...filters, environment: e.target.value })
              setPage(1)
            }}
          />

          <Input
            placeholder={t('build.filterByKeyword')}
            style={{ width: 150 }}
            allowClear
            size="small"
            prefix={<SearchOutlined />}
            value={filters.keyword}
            onChange={(e) => {
              setFilters({ ...filters, keyword: e.target.value })
              setPage(1)
            }}
          />

          <RangePicker
            placeholder={[t('build.startTime'), t('build.endTime')]}
            size="small"
            style={{ width: 280 }}
            format="YYYY-MM-DD"
            presets={[
              { label: '最近7天', value: [dayjs().subtract(7, 'day'), dayjs()] },
              { label: '最近14天', value: [dayjs().subtract(14, 'day'), dayjs()] },
              { label: '最近30天', value: [dayjs().subtract(30, 'day'), dayjs()] },
              { label: '最近90天', value: [dayjs().subtract(90, 'day'), dayjs()] },
            ]}
            onChange={(dates) => {
              if (dates && dates[0] && dates[1]) {
                setFilters({
                  ...filters,
                  start_time: dates[0].startOf('day').toISOString(),
                  end_time: dates[1].endOf('day').toISOString(),
                })
              } else {
                const { start_time, end_time, ...rest } = filters
                setFilters(rest)
              }
              setPage(1)
            }}
          />

          {Object.keys(filters).length > 0 && (
            <Button size="small" onClick={handleResetFilters}>{t('build.clearFilters')}</Button>
          )}
        </Space>
      </div>

      {/* 构建列表 */}
      <Table
        columns={columns}
        dataSource={buildData}
        rowKey="id"
        loading={isLoading}
        size="small"
        scroll={{ x: 880 }}
        pagination={{
          current: page,
          pageSize: pageSize,
          total: total,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (total) => `${t('common.total')} ${total} ${t('build.title')}`,
          onChange: (page, pageSize) => {
            setPage(page)
            setPageSize(pageSize)
          },
          size: 'small',
        }}
        locale={{
          emptyText: (
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description={t('build.noData')}
            />
          ),
        }}
      />
    </Drawer>
  )
}

export default BuildHistoryDrawer

