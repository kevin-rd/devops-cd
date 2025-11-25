# Dirty Fields 功能使用指南

## 📚 概述

Dirty Fields 功能帮助你追踪表单字段的变化，实现 **partial update**（只提交修改的字段）。

## 🎯 核心特性

✅ **自动追踪**：自动记录哪些字段被修改  
✅ **部分更新**：只提交变化的字段，减少网络传输  
✅ **智能缓存**：使用服务端返回数据直接更新，避免重新请求  
✅ **类型安全**：完整的 TypeScript 类型支持  
✅ **灵活配置**：支持排除字段、深度比较、空值处理等

---

## 📦 安装的依赖

```bash
npm install lodash-es
npm install -D @types/lodash-es
```

---

## 🛠️ 核心组件

### `useDirtyFields` Hook

**位置**：`frontend/src/hooks/useDirtyFields.ts`

**用途**：追踪表单字段变化的核心 Hook

**API：**

```typescript
const {
  setInitialValues,    // 设置初始值（编辑时调用）
  getDirtyValues,      // 获取变化的字段及其值
  getDirtyFields,      // 获取所有变化的字段名
  resetDirty,          // 重置 dirty 状态
  hasDirtyFields,      // 判断是否有变化
  isFieldDirty,        // 判断某个字段是否有变化
} = useDirtyFields(form, options)
```

**选项：**

```typescript
{
  excludeFields?: string[]     // 排除的字段（如 id, created_at）
  deepCompare?: boolean        // 是否深度比较（默认 true）
  treatEmptyAsSame?: boolean   // null/undefined/'' 是否视为相同（默认 true）
}
```

---

## 📖 使用示例

### 完整示例（Repository 页面）

```tsx
import { useDirtyFields } from '@/hooks/useDirtyFields'

const MyPage = () => {
  const [form] = Form.useForm()
  const [editingItem, setEditingItem] = useState(null)
  
  // 1. 初始化 useDirtyFields
  const {
    setInitialValues,
    getDirtyValues,
    getDirtyFields,
    resetDirty,
  } = useDirtyFields(form, {
    excludeFields: ['id', 'created_at', 'updated_at', 'status'],
    deepCompare: true,
    treatEmptyAsSame: true,
  })
  
  // 2. 编辑时设置初始值
  const handleEdit = (item) => {
    setEditingItem(item)
    form.setFieldsValue(item)
    setInitialValues(item as unknown as Record<string, unknown>)  // 设置初始值
    setModalVisible(true)
  }
  
  // 3. 提交时只发送 dirty fields
  const handleSubmit = () => {
    form.validateFields().then((values) => {
      let submitValues = values
      
      if (editingItem) {
        const dirtyValues = getDirtyValues()
        
        // 如果没有任何修改
        if (Object.keys(dirtyValues).length === 0) {
          message.info('没有任何修改')
          return
        }
        
        submitValues = dirtyValues
        console.log('📝 Dirty fields:', getDirtyFields())
        console.log('📦 Submitting:', submitValues)
      }
      
      mutation.mutate(submitValues)
    })
  }
  
  // 4. Mutation 成功后更新缓存
  const mutation = useMutation({
    mutationFn: async (values) => {
      if (editingItem) {
        return await api.update(editingItem.id, values)
      }
      return await api.create(values)
    },
    onSuccess: (response) => {
      // 使用返回的数据直接更新缓存
      if (response?.data) {
        queryClient.setQueryData(['items'], (oldData) => {
          // 更新逻辑...
          return updatedData
        })
      }
      
      resetDirty()  // 重置状态
      form.resetFields()
      setModalVisible(false)
    },
  })
  
  // 5. 使用标准的 Form.Item
  return (
    <Modal open={modalVisible} onOk={handleSubmit} onCancel={handleClose}>
      <Form form={form} layout="vertical">
        <Form.Item
          name="name"
          label="名称"
          rules={[{ required: true }]}
        >
          <Input />
        </Form.Item>
        
        <Form.Item
          name="description"
          label="描述"
        >
          <Input.TextArea />
        </Form.Item>
        
        {/* 对于复杂对象，也能正确追踪 */}
        <Form.Item
          name="env_clusters"
          label="环境集群"
        >
          <EnvClusterConfig />
        </Form.Item>
      </Form>
    </Modal>
  )
}
```

---

## 🔍 工作原理

### 1. 初始化阶段
```typescript
// 编辑时保存原始值
setInitialValues(originalData)
```

### 2. 编辑阶段
```typescript
// 表单字段被修改时，Hook 会自动追踪变化
// 可以通过 getDirtyFields() 查看哪些字段被修改了
```

### 3. 提交阶段
```typescript
// 只获取变化的字段
const dirtyValues = getDirtyValues()
// 例如：{ name: 'new-name', description: 'new-desc' }
// 而不是整个对象
```

### 4. 更新缓存
```typescript
// 使用服务端返回的数据直接更新缓存
queryClient.setQueryData(queryKey, (oldData) => {
  return updateWithNewData(oldData, response.data)
})
```

---

## 💡 最佳实践

### ✅ 推荐做法

1. **排除只读字段**
   ```typescript
   excludeFields: ['id', 'created_at', 'updated_at', 'status']
   ```

2. **使用深度比较处理对象/数组**
   ```typescript
   deepCompare: true  // 对于 env_clusters 等复杂字段
   ```

3. **空值统一处理**
   ```typescript
   treatEmptyAsSame: true  // null、undefined、'' 视为相同
   ```

4. **提交前验证**
   ```typescript
   if (Object.keys(dirtyValues).length === 0) {
     message.info('没有任何修改')
     return
   }
   ```

5. **直接更新缓存**
   ```typescript
   // ✅ 好：使用返回数据更新缓存
   onSuccess: (response) => {
     queryClient.setQueryData(key, updateFn)
   }
   
   // ❌ 坏：重新请求
   onSuccess: () => {
     queryClient.invalidateQueries(key)
   }
   ```

---

## 🐛 常见问题

### Q1: 为什么某些字段总是显示为 dirty？

**A**: 检查初始值和当前值的类型是否一致：
```typescript
// ❌ 问题：类型不一致
initialValue: null
currentValue: undefined

// ✅ 解决：启用 treatEmptyAsSame
useDirtyFields(form, {
  treatEmptyAsSame: true
})
```

### Q2: 复杂对象（如数组、嵌套对象）无法正确检测？

**A**: 确保启用深度比较：
```typescript
useDirtyFields(form, {
  deepCompare: true  // 对象/数组必须开启
})
```

### Q3: TypeScript 类型错误？

**A**: 使用类型转换：
```typescript
setInitialValues(data as unknown as Record<string, unknown>)
```

---

## 📊 性能考虑

| 操作 | 性能影响 | 建议 |
|------|---------|------|
| 深度比较 | 中等 | 仅在需要时启用 |
| 字段数量 | 低 | 支持大量字段 |
| 实时检测 | 低 | 使用 useCallback 优化 |
| 动画效果 | 低 | CSS 动画性能良好 |

---

## 🚀 进阶用法

### 1. 自定义比较逻辑

如果需要自定义某个字段的比较逻辑，可以在提交前手动处理：

```typescript
const dirtyValues = getDirtyValues()

// 自定义处理某些字段
if (dirtyValues.tags) {
  // 数组去重、排序后比较
  dirtyValues.tags = [...new Set(dirtyValues.tags)].sort()
}
```

### 2. 批量操作

```typescript
// 批量检查多个字段
const fields = ['name', 'description', 'app_type']
const allDirty = fields.every(field => isFieldDirty(field))
```

### 3. 条件提示

```typescript
// 关闭弹窗前提示
const handleClose = () => {
  if (hasDirtyFields()) {
    Modal.confirm({
      title: '有未保存的修改',
      content: '确定要关闭吗？',
      onOk: () => {
        resetDirty()
        setModalVisible(false)
      },
    })
  } else {
    setModalVisible(false)
  }
}
```

---

## 📝 总结

Dirty Fields 功能提供了：

1. ✨ **更好的用户体验**：高亮显示修改的字段
2. ⚡ **更高的性能**：只提交变化的字段
3. 🎯 **更少的网络请求**：直接更新缓存
4. 🔒 **更安全的操作**：避免意外覆盖未修改的字段

现在可以在任何需要编辑功能的页面中使用这个功能！

