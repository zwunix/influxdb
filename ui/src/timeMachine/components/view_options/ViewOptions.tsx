// Libraries
import React, {PureComponent} from 'react'
import {connect} from 'react-redux'

// Actions
import {setType, setVegaOptions} from 'src/timeMachine/actions'

// Components
import OptionsSwitcher from 'src/timeMachine/components/view_options/OptionsSwitcher'
import FancyScrollbar from 'src/shared/components/fancy_scrollbar/FancyScrollbar'
import {Grid} from 'src/clockface'

// Utils
import {getActiveTimeMachine} from 'src/timeMachine/selectors'

// Types
import {View, NewView, AppState} from 'src/types/v2'

// Styles
import './ViewOptions.scss'
import TextArea from 'src/clockface/components/inputs/TextArea'

interface DispatchProps {
  onUpdateType: typeof setType
  onUpdateVegaOptions: typeof setVegaOptions
}

interface StateProps {
  view: View | NewView
  vegaOptions: string
}

type Props = DispatchProps & StateProps

class ViewOptions extends PureComponent<Props> {
  public render() {
    const {vegaOptions} = this.props
    return (
      <div className="view-options">
        <FancyScrollbar autoHide={false}>
          <div className="view-options--container">
            <Grid>
              <Grid.Row>
                <TextArea
                  value={vegaOptions}
                  placeholder="Write text here"
                  onChange={this.handleTextChange}
                />
                {/* <OptionsSwitcher view={this.props.view} /> */}
              </Grid.Row>
            </Grid>
          </div>
        </FancyScrollbar>
      </div>
    )
  }

  private handleTextChange = (newVegaOptions: string) => {
    const {onUpdateVegaOptions} = this.props
    console.log(newVegaOptions)
    onUpdateVegaOptions(newVegaOptions)
  }
}

const mstp = (state: AppState): StateProps => {
  const {view} = getActiveTimeMachine(state)
  const {vegaOptions} = state.timeMachines

  return {view, vegaOptions}
}

const mdtp: DispatchProps = {
  onUpdateType: setType,
  onUpdateVegaOptions: setVegaOptions,
}

export default connect<StateProps, DispatchProps, {}>(
  mstp,
  mdtp
)(ViewOptions)
