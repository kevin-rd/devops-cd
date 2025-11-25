import type { FormInstance } from 'antd'
import { useRef, useCallback } from 'react'
import { isEqual } from 'lodash-es'

interface UseDirtyFieldsOptions<T> {
  // 需要排除的字段（如 id、created_at 等不应该提交的字段）
  excludeFields?: (keyof T)[]
  // 深度比较还是浅比较
  deepCompare?: boolean
  // 空值是否算作变化（null、undefined、'' 之间的转换）
  treatEmptyAsSame?: boolean
}

interface DirtyFieldsReturn<T> {
  // 设置初始值（在编辑模式时调用）
  setInitialValues: (values: T) => void
  // 获取变化的字段
  getDirtyValues: () => Partial<T>
  // 获取所有变化的字段名
  getDirtyFields: () => (keyof T)[]
  // 重置 dirty 状态
  resetDirty: () => void
  // 判断是否有变化
  hasDirtyFields: () => boolean
  // 判断某个字段是否有变化
  isFieldDirty: (fieldName: keyof T) => boolean
}

/**
 * 跟踪表单字段变化的 Hook
 * 
 * @example
 * ```tsx
 * const { setInitialValues, getDirtyValues, isFieldDirty } = useDirtyFields<Application>(appForm, {
 *   excludeFields: ['id', 'created_at', 'updated_at'],
 * })
 * 
 * // 编辑时设置初始值
 * handleEdit(app) {
 *   appForm.setFieldsValue(app)
 *   setInitialValues(app)
 * }
 * 
 * // 提交时只获取变化的字段
 * handleSubmit() {
 *   const dirtyValues = getDirtyValues()
 *   if (Object.keys(dirtyValues).length === 0) {
 *     message.info('没有任何修改')
 *     return
 *   }
 *   api.update(id, dirtyValues)
 * }
 * ```
 */
export function useDirtyFields<T extends Record<string, unknown> = Record<string, unknown>>(
  form: FormInstance,
  options?: UseDirtyFieldsOptions<T>
): DirtyFieldsReturn<T> {
  // 存储初始值
  const initialValuesRef = useRef<T | null>(null)
  
  // 默认配置
  const {
    excludeFields = [],
    deepCompare = true,
    treatEmptyAsSame = true,
  } = options || {}
  
  /**
   * 判断两个值是否为"空"且相等
   */
  const isEmptyValue = (value: unknown): boolean => {
    return value === null || value === undefined || value === ''
  }
  
  /**
   * 比较两个值是否相等
   */
  const areValuesEqual = useCallback(
    (value1: unknown, value2: unknown): boolean => {
      // 处理空值情况
      if (treatEmptyAsSame && isEmptyValue(value1) && isEmptyValue(value2)) {
        return true
      }
      
      // 深度比较或浅比较
      return deepCompare ? isEqual(value1, value2) : value1 === value2
    },
    [deepCompare, treatEmptyAsSame]
  )
  
  /**
   * 设置初始值（在编辑模式时调用）
   */
  const setInitialValues = useCallback((values: T) => {
    initialValuesRef.current = { ...values }
  }, [])
  
  /**
   * 判断某个字段是否有变化
   */
  const isFieldDirty = useCallback(
    (fieldName: keyof T): boolean => {
      if (!initialValuesRef.current) {
        // 如果没有初始值，认为是创建模式，所有字段都不算 dirty
        return false
      }
      
      // 排除字段不算 dirty
      if (excludeFields.includes(fieldName)) {
        return false
      }
      
      const currentValue = form.getFieldValue(fieldName as string)
      const initialValue = initialValuesRef.current[fieldName]
      
      return !areValuesEqual(currentValue, initialValue)
    },
    [form, excludeFields, areValuesEqual]
  )
  
  /**
   * 获取所有变化的字段名
   */
  const getDirtyFields = useCallback((): (keyof T)[] => {
    if (!initialValuesRef.current) {
      return []
    }
    
    const currentValues = form.getFieldsValue()
    const dirtyFieldNames: (keyof T)[] = []
    
    Object.keys(currentValues).forEach((key) => {
      if (isFieldDirty(key as keyof T)) {
        dirtyFieldNames.push(key as keyof T)
      }
    })
    
    return dirtyFieldNames
  }, [form, isFieldDirty])
  
  /**
   * 获取变化的字段及其值
   */
  const getDirtyValues = useCallback((): Partial<T> => {
    if (!initialValuesRef.current) {
      // 如果没有初始值，返回所有表单值（创建模式）
      return form.getFieldsValue()
    }
    
    const currentValues = form.getFieldsValue()
    const dirtyFields: Partial<T> = {}
    
    Object.keys(currentValues).forEach((key) => {
      if (isFieldDirty(key as keyof T)) {
        dirtyFields[key as keyof T] = currentValues[key]
      }
    })
    
    return dirtyFields
  }, [form, isFieldDirty])
  
  /**
   * 重置 dirty 状态
   */
  const resetDirty = useCallback(() => {
    initialValuesRef.current = null
  }, [])
  
  /**
   * 判断是否有改动
   */
  const hasDirtyFields = useCallback((): boolean => {
    return getDirtyFields().length > 0
  }, [getDirtyFields])
  
  return {
    setInitialValues,
    getDirtyValues,
    getDirtyFields,
    resetDirty,
    hasDirtyFields,
    isFieldDirty,
  }
}

