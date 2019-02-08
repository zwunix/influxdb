// Libraries
import {PureComponent} from 'react'
import _ from 'lodash'
import {connect} from 'react-redux'

// Components
import {ErrorHandling} from 'src/shared/decorators/errors'

// Types
import {Organization} from '@influxdata/influx'
import {AppState} from 'src/types/v2'

interface OwnProps {
  orgID: string
  children: (organization: Organization) => JSX.Element
}

interface StateProps {
  org: Organization
}

type Props = OwnProps & StateProps

@ErrorHandling
class GetOrganization extends PureComponent<Props> {
  public render() {
    return this.props.children(this.props.org)
  }
}

const mstp = (state: AppState, props: Props) => {
  const {orgs} = state
  const org = orgs.find(o => o.id === props.orgID)
  return {
    org,
  }
}

export default connect<StateProps, {}, OwnProps>(
  mstp,
  null
)(GetOrganization)
