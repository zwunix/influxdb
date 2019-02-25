import React, {SFC} from 'react'

interface Props {
  value: string
}

const TooltipValue: SFC<Props> = ({value}) => (
  <article className="flux-functions-toolbar--description">
    <div className="flux-functions-toolbar--heading">Value</div>
    <span>{value}</span>
  </article>
)

export default TooltipValue
