import {useEffect, useMemo, useState} from 'react'
import {Alert, Button, Checkbox, Empty, Input, Pagination, Select, Space, Spin, Table, Tag} from 'antd'
import {useTranslation} from 'react-i18next'
import {useQuery, useQueryClient} from '@tanstack/react-query'
import type {ColumnsType} from 'antd/es/table'
import type {ApplicationType, ApplicationWithBuild} from '@/types'
import {applicationService} from '@/services/application'
import {teamService, TeamSimple} from '@/services/team'
import './index.css'
import {ReloadOutlined} from "@ant-design/icons";

const {TextArea} = Input

export interface AppSelectionItem extends ApplicationWithBuild {
  selected: boolean
  inBatch: boolean
  releaseNotes: string
}

// 应用基本信息（用于 summary 显示）
interface AppBasicInfo {
  id: number
  name: string
  app_type?: string
  inBatch?: boolean
}

export interface AppSelectionTableProps {
  selection: {
    selectedIds: number[]
    existingIds?: number[]
    mode?: 'create' | 'edit'
  }
  projectId?: number  // 新增：当前批次的项目ID（用于筛选应用）
  onSelectionChange: (selectedIds: number[], meta?: { releaseNotes: Record<number, string> }) => void

  showReleaseNotes?: boolean
}

export default function AppSelectionTable(
  {
    projectId,
    selection,
    onSelectionChange,
    showReleaseNotes = true,
  }: AppSelectionTableProps) {
  const {t} = useTranslation()
  const queryClient = useQueryClient()
  const {selectedIds, existingIds = [], mode = 'create'} = selection

  const [releaseNotesMap, setReleaseNotesMap] = useState<Record<number, string>>({})
  const [expandedRowKeys, setExpandedRowKeys] = useState<number[]>([])

  // 存储所有已选/已存在应用的基本信息（不依赖当前分页）
  const [appsInfoMap, setAppsInfoMap] = useState<Record<number, AppBasicInfo>>({})

  // 控制已选应用列表的展开/折叠
  const [selectedAppsExpanded, setSelectedAppsExpanded] = useState(false)
  // 控制已移除应用列表的展开/折叠
  const [removedAppsExpanded, setRemovedAppsExpanded] = useState(false)

  // 限制显示的应用数量
  const MAX_DISPLAY_COUNT = 6

  // 内部查询状态
  const [queryData, setQueryData] = useState({
    page: 1,
    pageSize: 20,
    keyword: '',
    project_id: projectId,
    team_ids: [] as number[],  // 新增：多选团队
    app_types: [] as string[],  // 新增：多选应用类型
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
    queryKey: ['applicationsWithBuilds', debouncedSearchKeyword, queryData.page, queryData.pageSize, projectId, queryData.team_ids, queryData.app_types],
    queryFn: async ({signal}) => {
      const res = await applicationService.searchWithBuilds({
        page: queryData.page,
        page_size: queryData.pageSize,
        keyword: debouncedSearchKeyword || undefined,
        project_id: projectId,
        team_ids: queryData.team_ids.length > 0 ? queryData.team_ids : undefined,
        app_types: queryData.app_types.length > 0 ? queryData.app_types : undefined,
      }, signal)
      return res.data
    },
    staleTime: 10 * 1000,
    gcTime: 30 * 1000,
    enabled: !!projectId,
  })

  const list = useMemo(() => appsResponse?.items || [], [appsResponse?.items])
  const total = appsResponse?.total || 0

  // 加载团队列表（用于筛选）
  const {data: teamsResponse} = useQuery({
    queryKey: ['teams', projectId],
    queryFn: async (): Promise<TeamSimple[]> => {
      const res = await teamService.getList(projectId)
      return res.data
    },
    staleTime: 5 * 60 * 1000, // 5分钟
    enabled: !!projectId,
  })

  // 加载应用类型列表（用于筛选）
  const {data: appTypeMap} = useQuery({
    queryKey: ['appTypes'],
    queryFn: async () => {
      const res = await applicationService.getTypes()
      return new Map<string, ApplicationType>(
        res.data.types.map(type => [type.value, type])
      )
    },
    staleTime: 5 * 60 * 1000, // 5分钟
  })

  const teamOptions = useMemo(() => {
    return teamsResponse?.map(team => ({
      label: team.name,
      value: team.id,
    }))
  }, [teamsResponse])

  const appTypeOptions = useMemo(() => {
    return Array.from(appTypeMap?.values() || [], item => ({
      label: item.label,
      value: item.value,
    }))
  }, [appTypeMap])

  // 当列表数据变化时，更新 appsInfoMap（补充新出现的应用信息）
  useEffect(() => {
    setAppsInfoMap((prevInfoMap) => {
      const newInfoMap = {...prevInfoMap}
      let hasUpdate = false

      list.forEach((app) => {
        // 如果应用被选中或在批次中，且信息尚未存储，则存储其基本信息
        if ((selectedIds.includes(app.id) || existingIds.includes(app.id)) && !newInfoMap[app.id]) {
          newInfoMap[app.id] = {
            id: app.id,
            name: app.name,
            app_type: app.app_type,
            inBatch: existingIds.includes(app.id),
          }
          hasUpdate = true
        }
      })

      return hasUpdate ? newInfoMap : prevInfoMap
    })
  }, [list, selectedIds, existingIds])

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

  // 从 appsInfoMap 获取已选应用（不依赖当前分页）
  const selectedApps = useMemo(() => {
    return selectedIds.map((id) => {
      const info = appsInfoMap[id]
      return info || {id, name: `应用 ${id}`, inBatch: existingIds.includes(id)}
    })
  }, [selectedIds, appsInfoMap, existingIds])

  // 获取已移除应用的完整信息（从 appsInfoMap 获取）
  const removedApps = useMemo(() => {
    if (mode !== 'edit') return []
    const removedAppIds = existingIds.filter((id) => !selectedIds.includes(id))
    return removedAppIds.map((appId) => {
      const info = appsInfoMap[appId]
      return info || {id: appId, name: `应用 ${appId}`}
    })
  }, [mode, existingIds, selectedIds, appsInfoMap])

  // 统计信息
  const selectedCount = selectedApps.length
  const addedCount = mode === 'edit' ? selectedApps.filter((app) => !app.inBatch).length : 0
  const removedCount = removedApps.length

  const toggleSelect = (id: number) => {
    const isCurrentlySelected = selectedIds.includes(id)
    const next = isCurrentlySelected
      ? selectedIds.filter((x) => x !== id)
      : [...selectedIds, id]

    // 如果是新选中，需要存储应用信息
    if (!isCurrentlySelected) {
      const app = mergedList.find((x) => x.id === id)
      if (app) {
        setAppsInfoMap((prevInfoMap) => {
          if (!prevInfoMap[id]) {
            return {
              ...prevInfoMap,
              [id]: {
                id: app.id,
                name: app.name,
                app_type: app.app_type,
                inBatch: existingIds.includes(app.id),
              },
            }
          }
          return prevInfoMap
        })
      }
    }

    onSelectionChange(next, {releaseNotes: releaseNotesMap})
  }


  // 取消选择某个应用（从Tag上）
  const handleDeselectApp = (app: AppBasicInfo) => {
    const newSelectedIds = selectedIds.filter((id) => id !== app.id)
    onSelectionChange(newSelectedIds, {releaseNotes: releaseNotesMap})
  }

  // 重新选择某个已移除的应用
  const handleReselectApp = (appId: number) => {
    const newSelectedIds = [...selectedIds, appId]

    // 尝试从当前列表中获取应用信息并存储
    const app = mergedList.find((x) => x.id === appId)
    if (app) {
      setAppsInfoMap((prevInfoMap) => {
        if (!prevInfoMap[appId]) {
          return {
            ...prevInfoMap,
            [appId]: {
              id: app.id,
              name: app.name,
              app_type: app.app_type,
              inBatch: existingIds.includes(app.id),
            },
          }
        }
        return prevInfoMap
      })
    }

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
            const isChecked = e.target.checked
            let ids: number[]

            if (isChecked) {
              // 全选当前页
              ids = Array.from(new Set([...selectedIds, ...mergedList.map((x) => x.id)]))

              // 更新 appsInfoMap：添加当前页新选中的应用信息
              setAppsInfoMap((prevInfoMap) => {
                const newInfoMap = {...prevInfoMap}
                let hasUpdate = false

                mergedList.forEach((app) => {
                  if (!newInfoMap[app.id]) {
                    newInfoMap[app.id] = {
                      id: app.id,
                      name: app.name,
                      app_type: app.app_type,
                      inBatch: existingIds.includes(app.id),
                    }
                    hasUpdate = true
                  }
                })

                return hasUpdate ? newInfoMap : prevInfoMap
              })
            } else {
              // 取消当前页的所有选择
              ids = selectedIds.filter((id) => !mergedList.some((x) => x.id === id))
            }

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
        <Space>
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
      render: (type: string) => <Tag color={appTypeMap?.get(type)?.color || 'blue'}>{type}</Tag>,
    },
    {
      title: t('application.projectAndTeam'),
      key: 'project_name-team_name',
      width: 120,
      ellipsis: true,
      render: (_, record) =>
        record.project_name || record.team_name ? (
          <Tag>
            <span>{record.project_name ? record.project_name : '-'}</span>
            <span> / </span>
            <span>{record.team_name ? record.team_name : '-'}</span>
          </Tag>
        ) : (
          <Tag style={{color: '#999'}}>-</Tag>
        ),
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
              <code style={{fontSize: 13}}>{record.image_tag}</code>
              {record.commit_message && (
                <div style={{
                  fontSize: 11,
                  color: '#8c8c8c',
                  whiteSpace: 'nowrap',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis'
                }}>{record.commit_message}
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

              {/* 已选应用列表（保留占位） */}
              <div style={{marginBottom: mode === 'edit' ? 12 : 0, minHeight: 45}}>
                {selectedCount > 0 && (
                  <>
                    {mode === 'edit' && (
                      <div style={{fontSize: 12, color: '#8c8c8c', marginBottom: 4}}>
                        已选应用：
                      </div>
                    )}
                    <div style={{display: 'flex', flexWrap: 'wrap', gap: 8, alignItems: 'center'}}>
                      {(selectedAppsExpanded ? selectedApps : selectedApps.slice(0, MAX_DISPLAY_COUNT)).map((app) => (
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
                      {selectedCount > MAX_DISPLAY_COUNT && (
                        <Button
                          type="link"
                          size="small"
                          style={{padding: 0, height: 'auto', fontSize: 12}}
                          onClick={() => setSelectedAppsExpanded(!selectedAppsExpanded)}
                        >
                          {selectedAppsExpanded ? '收起' : `+${selectedCount - MAX_DISPLAY_COUNT} 更多`}
                        </Button>
                      )}
                    </div>
                  </>
                )}
              </div>

              {/* 已移除应用列表（仅编辑模式，保留占位） */}
              {mode === 'edit' && (
                <div style={{minHeight: 45}}>
                  {removedCount > 0 && (
                    <>
                      <div style={{fontSize: 12, color: '#8c8c8c', marginBottom: 4}}>
                        已移除应用：
                      </div>
                      <div style={{display: 'flex', flexWrap: 'wrap', gap: 8, alignItems: 'center'}}>
                        {(removedAppsExpanded ? removedApps : removedApps.slice(0, MAX_DISPLAY_COUNT)).map((app) => (
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
                        {removedCount > MAX_DISPLAY_COUNT && (
                          <Button
                            type="link"
                            size="small"
                            style={{padding: 0, height: 'auto', fontSize: 12}}
                            onClick={() => setRemovedAppsExpanded(!removedAppsExpanded)}
                          >
                            {removedAppsExpanded ? '收起' : `+${removedCount - MAX_DISPLAY_COUNT} 更多`}
                          </Button>
                        )}
                      </div>
                    </>
                  )}
                </div>
              )}
            </div>
          }
          type={selectedCount > 0 ? 'success' : 'info'}
          showIcon={false}
          style={{marginBottom: 12}}
        />

        {/* 筛选器, 搜索框和分页器 */}
        <div
          style={{display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: 12}}>
          <Space>
            {/* 筛选器：团队和应用类型 */}
            <Select
              mode="multiple"
              placeholder="筛选团队"
              style={{width: 160}}
              maxTagCount="responsive"
              value={queryData.team_ids}
              onChange={(values) => setQueryData({...queryData, team_ids: values, page: 1})}
              options={teamOptions}
              allowClear
            />

            <Select
              mode="multiple"
              placeholder="筛选应用类型"
              style={{width: 160}}
              maxTagCount="responsive"
              value={queryData.app_types}
              onChange={(values) => setQueryData({...queryData, app_types: values, page: 1})}
              options={appTypeOptions}
              allowClear
            />

            <Input.Search
              placeholder="搜索应用名称、代码库、Commit、Tag..."
              allowClear
              style={{width: 200, minWidth: 200}}
              value={queryData.keyword}
              onChange={(e) => {
                setQueryData({...queryData, keyword: e.target.value})
              }}
            />

            <Button icon={<ReloadOutlined/>}
                    onClick={() => queryClient.invalidateQueries({queryKey: ['applicationsWithBuilds', debouncedSearchKeyword]})}/>
          </Space>

          <Pagination
            size="small"
            current={queryData.page}
            pageSize={queryData.pageSize}
            total={total}
            showTotal={(total) => `${t('common.total')} ${total} ${t('common.unit')}`}
            showSizeChanger
            pageSizeOptions={['10', '20', '50', '100']}
            onChange={(page, pageSize) => {
              setQueryData({...queryData, page, pageSize})
            }}
          />
        </div>

      </div>

      {/* 应用表格 */}
      <Table
        rowKey="id"
        dataSource={mergedList}
        columns={columns}
        loading={{
          spinning: isLoading,
          indicator: <Spin size="large"/>,
        }}
        pagination={false}
        scroll={{y: 'calc(100vh - 400px)'}}
        locale={{
          emptyText: (
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description={
                <span style={{fontSize: 14, color: '#8c8c8c'}}>
                  {debouncedSearchKeyword ? '未找到匹配的应用' : '暂无可用应用'}
                </span>
              }
            />
          ),
        }}
        onRow={(record) => ({
          onClick: (e) => {
            // 避免点击 checkbox、Tag 的关闭按钮等交互元素时触发行选择
            const target = e.target as HTMLElement
            if (
              target.closest('.ant-checkbox') ||
              target.closest('.ant-tag-close-icon') ||
              target.closest('.ant-btn')
            ) {
              return
            }
            toggleSelect(record.id)
          },
          style: {cursor: 'pointer'},
        })}
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

