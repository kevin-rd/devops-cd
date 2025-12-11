export const BatchStatusConfig: Record<number, { label: string, color: string, class_name: string }> = {
  0: {label: '草稿', color: 'yellow', class_name: 'status-draft'},// 草稿 - 淡黄色
  10: {label: '已封板', color: 'purple', class_name: 'status-sealed'},// 已封板 - 紫色
  20: {label: 'Pre已触发', color: 'blue', class_name: 'status-pre-deploying'},// 预发布中 - 蓝色流光
  21: {label: 'Pre进行中', color: 'processing', class_name: 'status-pre-deploying'},
  22: {label: 'Pre部署完成', color: 'success', class_name: 'status-pre-deployed'}, // 预发布完成 -固定蓝色
  30: {label: 'Prod已触发', color: 'blue', class_name: 'status-prod-deploying'},// 生产部署中 - 橙色流光
  31: {label: 'Prod进行中', color: 'warning', class_name: 'status-prod-deploying'},
  32: {label: 'Prod部署完成', color: 'success', class_name: 'status-prod-deployed'},// 生产部署完成 - 固定橙色
  40: {label: '已完成', color: 'success', class_name: 'status-completed'},// 已完成 - 绿色
  90: {label: '已取消', color: 'default', class_name: 'status-cancelled'},// 已取消 - 灰色
}