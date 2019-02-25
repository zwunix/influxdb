// Libraries
import React, {PureComponent, MouseEvent, CSSProperties, createRef} from 'react'

// Components
import {ErrorHandling} from 'src/shared/decorators/errors'
import TooltipValue from 'src/timeMachine/components/variableToolbar/TooltipValue'

// Types
import {Variable} from '@influxdata/influx'

// Styles
import 'src/timeMachine/components/variableToolbar/VariableToolbar.scss'

interface Props {
  variable: Variable
  onDismiss: () => void
  tipPosition?: {top: number; right: number}
}

interface State {
  bottomPosition: number | null
}

class VariableTooltip extends PureComponent<Props, State> {
  public state: State = {bottomPosition: null}

  private tooltipRef = createRef<HTMLDivElement>()

  public componentDidMount() {
    const {bottom, height} = this.tooltipRef.current.getBoundingClientRect()

    if (bottom > window.innerHeight) {
      this.setState({bottomPosition: height / 2})
    }
  }

  public render() {
    return (
      <>
        <div
          style={this.stylePosition}
          className="variables-toolbar--tooltip"
          ref={this.tooltipRef}
        >
          <button
            className="variables-toolbar--tooltip-dismiss"
            onClick={this.handleDismiss}
          />
          <div className="variables-toolbar--tooltip-contents">
            <TooltipValue value={'value....'} />
          </div>
        </div>
        <span
          className="variables-toolbar--tooltip-caret"
          style={this.styleCaretPosition}
        />
      </>
    )
  }

  private get styleCaretPosition(): CSSProperties {
    const {top, right} = this.props.tipPosition

    return {
      top: `${Math.min(top, window.innerHeight)}px`,
      right: `${right + 4}px`,
    }
  }

  private get stylePosition(): CSSProperties {
    const {top, right} = this.props.tipPosition
    const {bottomPosition} = this.state

    return {
      bottom: `${bottomPosition || window.innerHeight - top - 15}px`,
      right: `${right + 2}px`,
    }
  }

  private handleDismiss = (e: MouseEvent<HTMLElement>) => {
    const {onDismiss} = this.props

    e.preventDefault()
    e.stopPropagation()
    onDismiss()
  }
}

export default ErrorHandling(VariableTooltip)
