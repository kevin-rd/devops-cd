import {memo} from 'react'
import type {NodeProps} from 'reactflow'
import styles from './GroupNode.module.css'

interface GroupNodeData {
  label: string
  width: number
  height: number
}

const GroupNode = memo(({}: NodeProps<GroupNodeData>) => {
  return (
    <div className={styles.group}>
    </div>
  )
})

GroupNode.displayName = 'GroupNode'

export default GroupNode
