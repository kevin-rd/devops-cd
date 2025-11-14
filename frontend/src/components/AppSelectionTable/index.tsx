import {useEffect, useMemo, useState} from 'react'
import {Alert, Button, Checkbox, Input, Select, Space, Table, Tag} from 'antd'
import {LeftOutlined, RightOutlined} from '@ant-design/icons'
import {useTranslation} from 'react-i18next'
import {useQuery} from '@tanstack/react-query'
import type {ColumnsType} from 'antd/es/table'
import type {ApplicationWithBuild} from '@/types'
import {applicationService} from '@/services/application'
import './index.css'

const {TextArea} = Input

export interface AppSelectionItem extends ApplicationWithBuild {
  selected: boolean
  inBatch: boolean
  releaseNotes: string
}

export interface AppSelectionTableProps {
  selection: {
    selectedIds: number[]
    existingIds?: number[]
    mode?: 'create' | 'edit'
  }
  onSelectionChange: (selectedIds: number[], meta?: { releaseNotes: Record<number, string> }) => void

  showReleaseNotes?: boolean
}

export default function AppSelectionTable(
  {
    selection,
    onSelectionChange,
    showReleaseNotes = true,
  }: AppSelectionTableProps) {
  const {t} = useTranslation()
  const {selectedIds, existingIds = [], mode = 'create'} = selection

  const [releaseNotesMap, setReleaseNotesMap] = useState<Record<number, string>>({})
  const [expandedRowKeys, setExpandedRowKeys] = useState<number[]>([])

  // 内部查询状态
  const [queryData, setQueryData] = useState({
    page: 1,
    pageSize: 20,
    keyword: '',
  })

  // 搜索防抖
  const [debouncedSearchKeyword, setDebouncedSearchKeyword] = useState('')
  useEffect(() => {
    const timer = window.setTimeout(() => {
      setDebouncedSearchKeyword(queryData.keyword)
    }, 500)
    return () => {
      window.clearTimeout(timer)
    }
  }, [queryData.keyword])

  // 查询应用列表
  const {data: appsResponse, isLoading} = useQuery({
    queryKey: ['applicationsWithBuilds', debouncedSearchKeyword, queryData.page, queryData.pageSize],
    queryFn: async ({signal}) => {
      const res = await applicationService.searchWithBuilds({
        page: queryData.page,
        page_size: queryData.pageSize,
        keyword: debouncedSearchKeyword || undefined,
      }, signal)
      return res.data
    },
    staleTime: 30 * 1000,
    gcTime: 60 * 1000,
  })

  const list = useMemo(() => appsResponse?.items || [], [appsResponse?.items])
  const total = appsResponse?.total || 0

  // 合并属性
  const mergedList: AppSelectionItem[] = useMemo(
    () =>
      list.map((app) => ({
        ...app,
        selected: selectedIds.includes(app.id),
        inBatch: existingIds.includes(app.id),
        releaseNotes: releaseNotesMap[app.id] || '',
      })),
    [list, selectedIds, existingIds, releaseNotesMap],
  )

  const selectedApps = mergedList.filter((x) => x.selected)

  // 获取已移除应用的完整信息（从当前列表中查找）
  const removedApps = useMemo(() => {
    if (mode !== 'edit') return []
    const removedAppIds = existingIds.filter((id) => !selectedIds.includes(id))
    return removedAppIds.map((appId) => {
      const app = mergedList.find((x) => x.id === appId)
      return app || {id: appId, name: `应用 ${appId}`}
    })
  }, [mode, existingIds, selectedIds, mergedList])

  // 统计信息
  const selectedCount = selectedApps.length
  const addedCount = mode === 'edit' ? selectedApps.filter((app) => !app.inBatch).length : 0
  const removedCount = removedApps.length

  const toggleSelect = (id: number) => {
    const next = selectedIds.includes(id)
      ? selectedIds.filter((x) => x !== id)
      : [...selectedIds, id]
    onSelectionChange(next, {releaseNotes: releaseNotesMap})
  }


  // 取消选择某个应用（从Tag上）
  const handleDeselectApp = (app: AppSelectionItem) => {
    const newSelectedIds = selectedIds.filter((id) => id !== app.id)
    onSelectionChange(newSelectedIds, {releaseNotes: releaseNotesMap})
  }

  // 重新选择某个已移除的应用
  const handleReselectApp = (appId: number) => {
    const newSelectedIds = [...selectedIds, appId]
    onSelectionChange(newSelectedIds, {releaseNotes: releaseNotesMap})
  }

  // 更新应用发布说明
  const updateAppReleaseNotes = (appId: number, notes: string) => {
    const newNotesMap = {...releaseNotesMap, [appId]: notes}
    setReleaseNotesMap(newNotesMap)
    onSelectionChange(selectedIds, {releaseNotes: newNotesMap})
  }

  // Table columns
  const columns: ColumnsType<AppSelectionItem> = [
    {
      title: (
        <Checkbox
          checked={mergedList.length > 0 && mergedList.every((x) => x.selected)}
          indeterminate={
            mergedList.some((x) => x.selected) &&
            !mergedList.every((x) => x.selected)
          }
          onChange={(e) => {
            const ids = e.target.checked
              ? Array.from(new Set([...selectedIds, ...mergedList.map((x) => x.id)]))
              : selectedIds.filter((id) => !mergedList.some((x) => x.id === id))
            onSelectionChange(ids, {releaseNotes: releaseNotesMap})
          }}
        />
      ),
      width: 40,
      render: (_, record) => (
        <Checkbox checked={record.selected} onChange={() => toggleSelect(record.id)}/>
      ),
    },
    {
      title: t('batch.appName'),
      dataIndex: 'name',
      width: 200,
      render: (text, rec) => (
        <Space onClick={() => toggleSelect(rec.id)} style={{cursor: 'pointer'}}>
          <span style={{color: '#999', fontSize: 12}}>#{rec.id} </span>
          <span>{text}</span>
          {mode === 'edit' && rec.inBatch && <Tag color="blue">原有</Tag>}
        </Space>
      ),
    },
    {
      title: t('batch.appType'),
      dataIndex: 'app_type',
      key: 'app_type',
      width: 80,
      render: (type: string) => <Tag color="blue">{type}</Tag>,
    },
    {
      title: t('application.repository'),
      dataIndex: 'repo_full_name',
      key: 'repo_full_name',
      width: 200,
      ellipsis: true,
    },
    {
      title: t('batch.currentVersion'),
      dataIndex: 'deployed_tag',
      key: 'deployed_tag',
      width: 150,
      ellipsis: true,
      render: (tag: string) => tag || <Tag color="default">-</Tag>,
    },
    {
      title: t('build.latestBuild'),
      key: 'latest_version',
      width: 180,
      ellipsis: true,
      render: (_: unknown, record: AppSelectionItem) => {
        if (record.image_tag) {
          return (
            <div>
              <code style={{fontSize: 11}}>{record.image_tag}</code>
              {record.commit_message && (
                <div style={{fontSize: 11, color: '#8c8c8c', marginTop: 2}}>
                  {record.commit_message.substring(0, 30)}
                  {record.commit_message.length > 30 && '...'}
                </div>
              )}
            </div>
          )
        }
        return <Tag color="default">{mode === 'create' ? '无构建' : '-'}</Tag>
      },
    },
  ]

  return (
    <div className="app-selection-table">
      {/* 固定顶部区域：统计信息、搜索框和分页器 */}
      <div className="sticky-top-bar">
        {/* 已选/已移除应用列表 */}
        <Alert
          message={
            <div>
              <div style={{fontWeight: 500, marginBottom: 8}}>
                已选择 {selectedCount} 个应用
                {mode === 'edit' && addedCount > 0 && (
                  <Tag color="green" style={{marginLeft: 8}}>
                    新增 {addedCount}
                  </Tag>
                )}
                {mode === 'edit' && removedCount > 0 && (
                  <Tag color="red" style={{marginLeft: 8}}>
                    移除 {removedCount}
                  </Tag>
                )}
              </div>

              {/* 已选应用列表 */}
              {selectedCount > 0 && (
                <div style={{marginBottom: mode === 'edit' && removedCount > 0 ? 12 : 0}}>
                  {mode === 'edit' && (
                    <div style={{fontSize: 12, color: '#8c8c8c', marginBottom: 4}}>
                      已选应用：
                    </div>
                  )}
                  <div style={{display: 'flex', flexWrap: 'wrap', gap: 8}}>
                    {selectedApps.map((app) => (
                      <Tag
                        key={app.id}
                        color={mode === 'edit' && !app.inBatch ? 'green' : 'blue'}
                        closable
                        onClose={(e) => {
                          e.preventDefault()
                          handleDeselectApp(app)
                        }}
                      >
                        <span style={{color: "#999", fontSize: 11}}>#{app.id} </span>
                        <span>{app.name} {mode === 'edit' && !app.inBatch && ''}</span>
                      </Tag>
                    ))}
                  </div>
                </div>
              )}

              {/* 已移除应用列表（仅编辑模式） */}
              {mode === 'edit' && removedCount > 0 && (
                <div>
                  <div style={{fontSize: 12, color: '#8c8c8c', marginBottom: 4}}>
                    已移除应用：
                  </div>
                  <div style={{display: 'flex', flexWrap: 'wrap', gap: 8}}>
                    {removedApps.map((app) => (
                      <Tag
                        key={app.id}
                        color="red"
                        closable
                        onClose={(e) => {
                          e.preventDefault()
                          handleReselectApp(app.id)
                        }}
                      >
                        <span style={{color: "#999", fontSize: 11}}>#{app.id} </span>
                        <span>{app.name}</span>
                      </Tag>
                    ))}
                  </div>
                </div>
              )}
            </div>
          }
          type={selectedCount > 0 ? 'success' : 'info'}
          showIcon={false}
          style={{marginBottom: 12}}
        />

        {/* 搜索框和分页器 */}
        <div className="search-pagination-wrapper">
          <Input.Search
            placeholder="搜索应用名称、代码库、Commit、Tag..."
            allowClear
            style={{width: 400, minWidth: 280}}
            value={queryData.keyword}
            onChange={(e) => {
              setQueryData({...queryData, keyword: e.target.value})
            }}
          />
          <div style={{flex: 1}}/>
          <div style={{display: 'flex', alignItems: 'center', gap: 12, flexShrink: 0}}>
            <span style={{fontSize: 13, color: '#8c8c8c', whiteSpace: 'nowrap'}}>
              共 {total || 0} 个
            </span>
            <Space size={4}>
              <Button
                size="small"
                icon={<LeftOutlined/>}
                disabled={queryData.page === 1}
                onClick={() => {
                  setQueryData({...queryData, page: queryData.page - 1})
                }}
              />
              <span style={{fontSize: 13, whiteSpace: 'nowrap', padding: '0 4px'}}>
                {queryData.page} / {Math.ceil((total || 0) / queryData.pageSize) || 1}
              </span>
              <Button
                size="small"
                icon={<RightOutlined/>}
                disabled={queryData.page >= Math.ceil((total || 0) / queryData.pageSize)}
                onClick={() => {
                  setQueryData({...queryData, page: queryData.page + 1})
                }}
              />
              <Select
                size="small"
                value={queryData.pageSize}
                onChange={(value) => {
                  setQueryData({...queryData, pageSize: value, page: 1})  // 重置到第一页
                }}
                style={{width: 90}}
                options={[
                  {label: '10/页', value: 10},
                  {label: '20/页', value: 20},
                  {label: '50/页', value: 50},
                  {label: '100/页', value: 100},
                ]}
              />
            </Space>
          </div>
        </div>
      </div>

      {/* 应用表格 */}
      <Table
        rowKey="id"
        dataSource={mergedList}
        columns={columns}
        loading={isLoading}
        pagination={false}
        scroll={{y: 'calc(100vh - 400px)'}}
        expandable={
          showReleaseNotes
            ? {
              expandedRowRender: (record) => (
                <div style={{padding: '12px 24px'}}>
                  <div style={{marginBottom: 8}}>
                    <strong>{t('batch.appReleaseNotes')}:</strong>
                  </div>
                  <TextArea
                    rows={3}
                    placeholder={t('batch.appReleaseNotesPlaceholder')}
                    value={record.releaseNotes}
                    onChange={(e) => updateAppReleaseNotes(record.id, e.target.value)}
                  />
                </div>
              ),
              rowExpandable: (record) => record.selected,
              expandedRowKeys: expandedRowKeys,
              onExpand: (expanded, record) => {
                if (expanded) {
                  setExpandedRowKeys([...expandedRowKeys, record.id])
                } else {
                  setExpandedRowKeys(expandedRowKeys.filter((key) => key !== record.id))
                }
              },
            }
            : undefined
        }
      />
    </div>
  )
}

