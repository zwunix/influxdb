// Libraries
import React, {PureComponent} from 'react'
import {connect} from 'react-redux'
import _ from 'lodash'

// APIs
import {client} from 'src/utils/api'

// Actions
import * as notifyActions from 'src/shared/actions/notifications'

// Components
import {SpinnerContainer, TechnoSpinner} from 'src/clockface'
import TabbedPageSection from 'src/shared/components/tabbed_page/TabbedPageSection'
import GetOrgResources from 'src/organizations/components/GetOrgResources'
import Buckets from 'src/organizations/components/Buckets'
import GetOrganization from 'src/organizations/components/GetOrganization'

// Types
import {Organization, Bucket} from '@influxdata/influx'
import * as NotificationsActions from 'src/types/actions/notifications'

// Decorators
import {ErrorHandling} from 'src/shared/decorators/errors'

interface OwnProps {
  params: {orgID: string}
}

interface DispatchProps {
   notify: NotificationsActions.PublishNotificationActionCreator
}

type Props = OwnProps & DispatchProps

@ErrorHandling
class BucketsPage extends PureComponent<Props> {
  public render() {
    const {params, notify} = this.props

    return (
      <TabbedPageSection
        id="org-view-tab--members"
        url="members_tab"
        title="Members"
      >
        <GetOrganization orgID={params.orgID}>
          {org => (
            <GetOrgResources<Bucket[]>
              organization={org}
              fetcher={this.getBuckets}
            >
              {(buckets, loading, fetch) => (
                <SpinnerContainer
                  loading={loading}
                  spinnerComponent={<TechnoSpinner />}
                >
                  <Buckets
                    buckets={buckets}
                    org={org}
                    onChange={fetch}
                    notify={notify}
                  />
                </SpinnerContainer>
              )}
            </GetOrgResources>
          )}
        </GetOrganization>
      </TabbedPageSection>
    )
  }
  private getBuckets = async (org: Organization) => {
    return client.buckets.getAllByOrg(org)
  }
}
const mdtp: DispatchProps = {
    notify: notifyActions.notify,
}
  
export default connect<{}, DispatchProps, {}>(
    mdtp
  )(BucketsPage)