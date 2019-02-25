// Libraries
import React, {PureComponent, createRef} from 'react'

// Components
import VariableTooltip from 'src/timeMachine/components/variableToolbar/VariableTooltip'

// Types
import {Variable} from '@influxdata/influx'

// Styles
import 'src/timeMachine/components/fluxFunctionsToolbar/FluxFunctionsToolbar.scss'

interface Props {
  variable: Variable
}

interface State {
  isActive: boolean
  hoverPosition: {top: number; right: number}
}

class VariableItem extends PureComponent<Props> {
  public state: State = {isActive: false, hoverPosition: undefined}
  private variableRef = createRef<HTMLDivElement>()

  public render() {
    const {variable} = this.props
    return (
      <div
        className="variables-toolbar--item"
        ref={this.variableRef}
        onMouseEnter={this.handleHover}
        onMouseLeave={this.handleStopHover}
      >
        <dd className="variables-toolbar--label">{variable.name}</dd>
        {this.tooltip}
      </div>
    )
  }

  private get tooltip(): JSX.Element | null {
    if (this.state.isActive) {
      return (
        <VariableTooltip
          variable={this.props.variable}
          onDismiss={this.handleStopHover}
          tipPosition={this.state.hoverPosition}
        />
      )
    }
  }

  private handleHover = () => {
    const {top, left} = this.variableRef.current.getBoundingClientRect()
    const right = window.innerWidth - left

    this.setState({isActive: true, hoverPosition: {top, right}})
  }

  private handleStopHover = () => {
    this.setState({isActive: false})
  }
}

export default VariableItem
