// Libraries
import React, {PureComponent} from 'react'
import _ from 'lodash'
import {connect} from 'react-redux'
import {withRouter, WithRouterProps} from 'react-router'

// Components
import ImportOverlay from 'src/shared/components/ImportOverlay'

// Constants
import {dashboardImportFailed} from 'src/shared/copy/notifications'

// Actions
import {notify as notifyAction} from 'src/shared/actions/notifications'
import {getDashboardsAsync} from 'src/dashboards/actions/v2'
import {createDashboardFromTemplate as createDashboardFromTemplateAction} from 'src/dashboards/actions/v2'

// Types
import {Organization, AppState} from 'src/types/v2'

interface OwnProps extends WithRouterProps {
  params: {orgID: string}
}

interface StateProps {
  orgs: Organization[]
}
interface DispatchProps {
  notify: typeof notifyAction
  createDashboardFromTemplate: typeof createDashboardFromTemplateAction
  populateDashboards: typeof getDashboardsAsync
}

type Props = OwnProps & DispatchProps & StateProps

class ImportDashboardOverlay extends PureComponent<Props> {
  constructor(props: Props) {
    super(props)
  }

  public render() {
    return (
      <ImportOverlay
        onDismissOverlay={this.handleDismissOverlay}
        resourceName="Dashboard"
        onSubmit={this.handleUploadDashboard}
      />
    )
  }

  private handleUploadDashboard = async (
    uploadContent: string
  ): Promise<void> => {
    const {
      notify,
      createDashboardFromTemplate,
      populateDashboards,
      params: {orgID},
      orgs,
    } = this.props

    try {
      const template = JSON.parse(uploadContent)

      if (_.isEmpty(template)) {
        this.handleDismissOverlay()
        return
      }

      await createDashboardFromTemplate(
        template,
        orgID || _.get(orgs, '0.id', '')
      )
      await populateDashboards()

      this.handleDismissOverlay()
    } catch (error) {
      notify(dashboardImportFailed(error))
    }
  }

  private handleDismissOverlay = () => {
    this.props.router.goBack()
  }
}

const mstp = ({orgs}: AppState): StateProps => ({
  orgs,
})

const mdtp: DispatchProps = {
  notify: notifyAction,
  createDashboardFromTemplate: createDashboardFromTemplateAction,
  populateDashboards: getDashboardsAsync,
}

export default connect<StateProps, DispatchProps, OwnProps>(
  mstp,
  mdtp
)(withRouter(ImportDashboardOverlay))
