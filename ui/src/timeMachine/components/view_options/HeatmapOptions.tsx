// Libraries
import React, {SFC} from 'react'
import {connect} from 'react-redux'

// Components
import {Form, Grid} from 'src/clockface'
import NuColorSchemeDropdown from 'src/shared/components/NuColorSchemeDropdown'

// Actions
import {setColorScheme} from 'src/timeMachine/actions'

interface StateProps {}

interface DispatchProps {
  onSetColorScheme: typeof setColorScheme
}

interface OwnProps {
  colors: string[]
}

type Props = StateProps & DispatchProps & OwnProps

const HeatmapOptions: SFC<Props> = ({colors, onSetColorScheme}) => {
  return (
    <Grid.Column>
      <h4 className="view-options--header">Customize Heatmap</h4>
      <Form.Element label="Color Scheme">
        <NuColorSchemeDropdown
          colorScheme={colors}
          onSetColorScheme={onSetColorScheme}
        />
      </Form.Element>
    </Grid.Column>
  )
}

const mstp = () => {
  return {}
}

const mdtp = {
  onSetColorScheme: setColorScheme,
}

export default connect<StateProps, DispatchProps, OwnProps>(
  mstp,
  mdtp
)(HeatmapOptions)
