import {Table} from 'src/minard'
import {assert} from 'src/minard/utils/assert'
import {isNumeric} from 'src/minard/utils/isNumeric'

export const getNumericColumnData = (
  table: Table,
  colName: string
): number[] => {
  const col = table.columns[colName]

  assert(`could not find column "${colName}"`, !!col)
  assert(`unsupported column type "${col.type}"`, isNumeric(col.type))

  return col.data as number[]
}
