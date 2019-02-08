// Libraries
import React, {PureComponent} from 'react'
import {WithRouterProps, InjectedRouter} from 'react-router'
import {connect} from 'react-redux'
import _ from 'lodash'

// APIs
import {getDashboards} from 'src/organizations/apis'
import {client} from 'src/utils/api'

// Actions
import {updateOrg} from 'src/organizations/actions'
import * as notifyActions from 'src/shared/actions/notifications'

// Components
import {Page} from 'src/pageLayout'
import {SpinnerContainer, TechnoSpinner} from 'src/clockface'
import TabbedPage from 'src/shared/components/tabbed_page/TabbedPage'
import TabbedPageSection from 'src/shared/components/tabbed_page/TabbedPageSection'
import OrgTasksPage from 'src/organizations/components/OrgTasksPage'

// Types
import {AppState} from 'src/types/v2'
import {Organization} from '@influxdata/influx'
import * as NotificationsActions from 'src/types/actions/notifications'

// Decorators
import {ErrorHandling} from 'src/shared/decorators/errors'
import {Task} from 'src/tasks/containers/TasksPage'
import GetOrgResources from './GetOrgResources'

interface Props {
  router: InjectedRouter
  org: Organization
}

@ErrorHandling
export default class OrgTasksTab extends PureComponent<Props> {
  public render() {
    const {org, router} = this.props

    return (
      <TabbedPageSection id="org-view-tab--tasks" url="tasks_tab" title="Tasks">
        <GetOrgResources<Task[]> organization={org} fetcher={getTasks}>
          {(tasks, loading, fetch) => (
            <SpinnerContainer
              loading={loading}
              spinnerComponent={<TechnoSpinner />}
            >
              <OrgTasksPage
                tasks={tasks}
                orgName={org.name}
                orgID={org.id}
                onChange={fetch}
                router={router}
              />
            </SpinnerContainer>
          )}
        </GetOrgResources>
      </TabbedPageSection>
    )
  }
}
