import {useCallback, useEffect, useState} from 'react'
import {Button, Drawer, Form, Input, message, Space, Steps,} from 'antd'
import {AppstoreOutlined, FormOutlined, SaveOutlined} from '@ant-design/icons'
import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query'
import {useTranslation} from 'react-i18next'
import {batchService} from '@/services/batch'
import {useAuthStore} from '@/stores/authStore'
import type {UpdateBatchRequest} from '@/types'
import AppSelectionTable from '@/components/AppSelectionTable'
import './index.css'
import {Batch} from "@/types/batch.ts";
import {ReleaseApp} from "@/types/release_app.ts";

const {TextArea} = Input

interface BatchEditDrawerProps {
  open: boolean
  batch: Batch | null
  onClose: () => void
  onSuccess: () => void
}

export default function BatchEditDrawer({open, batch, onClose, onSuccess}: BatchEditDrawerProps) {
  const {t} = useTranslation()
  const [form] = Form.useForm()
  const {user} = useAuthStore()
  const queryClient = useQueryClient()

  const [currentStep, setCurrentStep] = useState(1) // 默认显示应用管理页面

  // 选中的应用ID和发布说明
  const [selectedAppIds, setSelectedAppIds] = useState<number[]>([])
  const [releaseNotesMap, setReleaseNotesMap] = useState<Record<number, string>>({})

  // 批次原有的应用IDs
  const [existingAppIds, setExistingAppIds] = useState<number[]>([])

  // 查询批次详情（使用 placeholderData 先展示传入的数据）
  const {data: batchDetailResponse, isLoading: loadingBatchDetail, isFetching: fetchingBatchDetail} = useQuery({
    queryKey: ['batchDetail', batch?.id],
    queryFn: async () => {
      if (!batch?.id) return null
      const res = await batchService.get(batch.id)
      return res.data
    },
    enabled: open && !!batch,
    placeholderData: batch || undefined,
    staleTime: 0, // 始终获取最新数据
  })

  // 当获取到最新数据时，同步更新列表页缓存
  useEffect(() => {
    if (batchDetailResponse && batch?.id && !fetchingBatchDetail) {
      // 更新批次列表缓存
      queryClient.setQueriesData(
        {queryKey: ['batchList'], exact: false},
        (oldData: any) => {
          if (!oldData?.items) return oldData
          return {
            ...oldData,
            items: oldData.items.map((item: Batch) =>
              item.id === batch.id ? {...item, ...batchDetailResponse} : item
            ),
          }
        }
      )

      // 更新展开详情缓存
      queryClient.setQueriesData(
        {queryKey: ['batchDetails'], exact: false},
        (oldData: any) => {
          if (!oldData) return oldData
          return {
            ...oldData,
            [batch.id]: batchDetailResponse,
          }
        }
      )
    }
  }, [batchDetailResponse, batch?.id, fetchingBatchDetail, queryClient])

  const batchDetail = batchDetailResponse || batch

  // 初始化批次应用IDs
  useEffect(() => {
    if (!batchDetail?.apps) {
      return
    }
    const appIds = batchDetail.apps
      .map((app: ReleaseApp) => app.app_id)
      .filter((id): id is number => id !== undefined)

    setExistingAppIds(appIds)
    setSelectedAppIds(appIds)
  }, [batchDetail])

  // 更新批次 Mutation
  const updateMutation = useMutation({
    mutationFn: (data: UpdateBatchRequest) => batchService.update(data),
    onSuccess: (response) => {
      message.success(t('batch.updateSuccess'))

      // 如果返回了更新后的批次数据，更新缓存
      if (response?.data) {
        const updatedBatch = response.data

        // 更新批次详情缓存
        queryClient.setQueryData(['batchDetail', batch?.id], updatedBatch)

        // 更新批次列表缓存中的对应项
        queryClient.setQueriesData(
          {queryKey: ['batchList'], exact: false},
          (oldData: any) => {
            if (!oldData?.items) return oldData
            return {
              ...oldData,
              items: oldData.items.map((item: Batch) =>
                item.id === updatedBatch.id ? {...item, ...updatedBatch} : item
              ),
            }
          }
        )

        // 更新展开详情缓存
        queryClient.setQueriesData(
          {queryKey: ['batchDetails'], exact: false},
          (oldData: any) => {
            if (!oldData) return oldData
            return {
              ...oldData,
              [updatedBatch.id]: updatedBatch,
            }
          }
        )
      } else {
        // 如果没有返回数据，则失效缓存让其重新获取
        queryClient.invalidateQueries({queryKey: ['batchList']})
        queryClient.invalidateQueries({queryKey: ['batchDetail', batch?.id]})
        queryClient.invalidateQueries({queryKey: ['batchDetails']})
      }

      handleClose()
      onSuccess()
    },
    onError: (error: any) => {
      // 显示后端返回的详细错误信息
      const errorMsg = error.response?.data?.message || error.message || t('common.error')
      const errorDetail = error.response?.data?.detail

      if (errorDetail) {
        message.error(`${errorMsg}: ${errorDetail}`, 5) // 显示5秒
      } else {
        message.error(errorMsg, 5)
      }
    },
  })

  // 初始化表单
  useEffect(() => {
    if (open && batchDetail) {
      form.setFieldsValue({
        batch_number: batchDetail.batch_number,
        release_notes: batchDetail.release_notes || '',
      })
    }
  }, [open, batchDetail, form])

  // 关闭并重置
  const handleClose = () => {
    form.resetFields()
    setCurrentStep(1) // 重置为默认步骤
    setSelectedAppIds([])
    setReleaseNotesMap({})
    setExistingAppIds([])
    onClose()
  }

  // 步骤切换处理
  const handleStepChange = (step: number) => {
    // 如果要切换到步骤2（应用管理），先验证基本信息
    if (step === 1) {
      form.validateFields(['batch_number'])
        .then(() => {
          setCurrentStep(step)
        })
        .catch((error) => {
          console.error('Validation failed:', error)
          message.warning('请先填写批次编号')
        })
    } else {
      setCurrentStep(step)
    }
  }

  // 处理应用选择变化
  const handleSelectionChange = useCallback(
    (newSelectedIds: number[], newReleaseNotes?: Record<number, string>) => {
      setSelectedAppIds(newSelectedIds)
      if (newReleaseNotes) {
        setReleaseNotesMap(newReleaseNotes)
      }
    },
    []
  )

  // 保存批次修改
  const handleUpdate = async () => {
    if (!batch || !batchDetail) return

    try {
      const formValues = await form.validateFields()

      // 计算应用变更
      const addedAppIds = selectedAppIds.filter(id => !existingAppIds.includes(id))
      const removedAppIds = existingAppIds.filter(id => !selectedAppIds.includes(id))

      // 构建请求数据
      const requestData: UpdateBatchRequest = {
        batch_id: batch.id,
        operator: user?.username || 'unknown',
      }

      // 添加基本信息变更
      if (formValues.batch_number !== batch.batch_number) {
        requestData.batch_number = formValues.batch_number
      }
      if (formValues.release_notes !== (batch.release_notes || '')) {
        requestData.release_notes = formValues.release_notes?.trim() || ''
      }

      // 添加应用变更
      if (addedAppIds.length > 0) {
        requestData.add_apps = addedAppIds.map(appId => {
          const app: any = {app_id: appId}
          const releaseNotes = releaseNotesMap[appId]?.trim()
          if (releaseNotes) {
            app.release_notes = releaseNotes
          }
          return app
        })
      }
      if (removedAppIds.length > 0) {
        requestData.remove_app_ids = removedAppIds
      }

      // 检查是否有变化
      if (
        !requestData.batch_number &&
        requestData.release_notes === undefined &&
        !requestData.add_apps &&
        !requestData.remove_app_ids
      ) {
        message.info('没有需要更新的内容')
        return
      }

      console.log('Updating batch with data:', requestData)
      updateMutation.mutate(requestData)
    } catch (error) {
      console.error('Validation failed:', error)
    }
  }


  // 底部按钮
  const renderFooter = () => {
    return (
      <Space>
        <Button onClick={handleClose}>
          {t('common.cancel')}
        </Button>
        <Button
          type="primary"
          icon={<SaveOutlined/>}
          loading={updateMutation.isPending}
          onClick={handleUpdate}
        >
          {t('common.save')}
        </Button>
      </Space>
    )
  }

  if (!batch) return null

  return (
    <Drawer
      title={t('batch.edit')}
      placement="right"
      width="70%"
      open={open}
      onClose={handleClose}
      destroyOnClose={false}
      maskClosable={false}
      footer={renderFooter()}
      footerStyle={{textAlign: 'right'}}
      className="batch-edit-drawer"
    >
      {/* 步骤指示器 */}
      <div style={{
        display: 'flex',
        justifyContent: 'center',
        marginBottom: 0,
        paddingTop: 24,
        paddingLeft: 24,
        paddingRight: 24
      }}>
        <Steps
          current={currentStep}
          onChange={handleStepChange}
          items={[
            {
              title: t('batch.step1'),
              icon: <FormOutlined/>,
            },
            {
              title: t('batch.step2'),
              icon: <AppstoreOutlined/>,
            },
          ]}
          style={{maxWidth: 500}}
        />
      </div>

      <Form
        form={form}
        layout="vertical"
      >
        {/* 步骤1: 基本信息 */}
        <div style={{display: currentStep === 0 ? 'block' : 'none'}}>
          <Form.Item
            name="batch_number"
            label={t('batch.batchNumber')}
            rules={[{required: true, message: '请输入批次编号'}]}
          >
            <Input
              placeholder={t('batch.batchNumberPlaceholder')}
              size="large"
            />
          </Form.Item>

          <Form.Item
            name="release_notes"
            label={t('batch.releaseNotes')}
          >
            <TextArea
              rows={8}
              placeholder={t('batch.releaseNotesPlaceholder')}
            />
          </Form.Item>
        </div>

        {/* 步骤2: 应用管理 */}
        <div style={{display: currentStep === 1 ? 'block' : 'none'}}>
          <AppSelectionTable
            selection={{selectedIds: selectedAppIds, existingIds: existingAppIds, mode: 'edit'}}
            projectId={batch.project_id}
            onSelectionChange={handleSelectionChange}
            showReleaseNotes={true}
          />
        </div>
      </Form>
    </Drawer>
  )
}
