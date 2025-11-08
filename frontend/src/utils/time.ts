import dayjs from 'dayjs'
import isoWeek from 'dayjs/plugin/isoWeek'

dayjs.extend(isoWeek)

/**
 * 格式化创建时间，显示周几和相对时间
 * @param createdAt ISO 格式的时间字符串
 * @returns { time: 'YYYY-MM-DD HH:mm' 格式的时间, dayInfo: '相对时间 周几' }
 *
 * 示例：
 * - 今天 周三
 * - 昨天 周二
 * - 本周 周一
 * - 上周 周五
 * - 上上周 周四
 * - 2周前 周三
 * - 上个月 周一
 * - 3个月前 周五
 */
export const formatCreatedTime = (createdAt: string): { time: string; dayInfo: string } => {
  const created = dayjs(createdAt).startOf('day')
  const now = dayjs().startOf('day')

  const weekDays = ['日', '一', '二', '三', '四', '五', '六']
  const weekDay = weekDays[created.day()]

  // 周一为一周起点（国内习惯）
  const startOfThisWeek = now.startOf('isoWeek')  // 周一
  const startOfLastWeek = startOfThisWeek.subtract(1, 'week')
  const startOfThisMonth = now.startOf('month')

  const diffDays = now.diff(created, 'day')

  let dayInfo = ''

  // 今天
  if (diffDays === 0) {
    dayInfo = `今天 周${weekDay}`
  }
  // 昨天
  else if (diffDays === 1) {
    dayInfo = `昨天 周${weekDay}`
  }
  // 本周（周一到昨天）
  else if (created.isAfter(startOfThisWeek) && created.isBefore(now)) {
    dayInfo = `本周${weekDay}`
  }
  // 上周（上周一 ~ 上周日）
  else if (created.isAfter(startOfLastWeek) && created.isBefore(startOfThisWeek)) {
    dayInfo = `上周${weekDay}`
  }
  // 上上周及更早：按周数计算
  else if (diffDays < 30) { // 30天内用"周"显示
    const weeksAgo = Math.ceil(diffDays / 7)
    if (weeksAgo === 2) {
      dayInfo = `上上周${weekDay}`
    } else {
      dayInfo = `${weeksAgo}周前 周${weekDay}`
    }
  }
  // 上个月
  else if (created.isAfter(startOfThisMonth.subtract(1, 'month')) && created.isBefore(startOfThisMonth)) {
    dayInfo = `上个月 周${weekDay}`
  }
  // 更早：按月
  else {
    const monthsAgo = Math.floor(diffDays / 30)
    dayInfo = monthsAgo === 1 ? `上个月 周${weekDay}` : `${monthsAgo}个月前 周${weekDay}`
  }

  return {
    time: dayjs(createdAt).format('YYYY-MM-DD HH:mm'),
    dayInfo
  }
}

