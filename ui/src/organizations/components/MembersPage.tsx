// Libraries
import React, {PureComponent} from 'react'
import _ from 'lodash'

// APIs
import {client} from 'src/utils/api'

// Components
import {SpinnerContainer, TechnoSpinner} from 'src/clockface'
import TabbedPageSection from 'src/shared/components/tabbed_page/TabbedPageSection'
import GetOrgResources from 'src/organizations/components/GetOrgResources'
import MembersTab from 'src/organizations/components/MembersTab'
import GetOrganization from 'src/organizations/components/GetOrganization'

// Types
import {Organization, ResourceOwner} from '@influxdata/influx'

// Decorators
import {ErrorHandling} from 'src/shared/decorators/errors'

interface Props {
  params: {orgID: string}
}

@ErrorHandling
export default class MembersPage extends PureComponent<Props> {
  public render() {
    const {params} = this.props

    return (
      <TabbedPageSection
        id="org-view-tab--members"
        url="members_tab"
        title="Members"
      >
        <GetOrganization orgID={params.orgID}>
          {org => (
            <GetOrgResources<ResourceOwner[]>
              organization={org}
              fetcher={this.getOwnersAndMembers}
            >
              {(members, loading) => (
                <SpinnerContainer
                  loading={loading}
                  spinnerComponent={<TechnoSpinner />}
                >
                  <MembersTab members={members} orgName={org.name} />
                </SpinnerContainer>
              )}
            </GetOrgResources>
          )}
        </GetOrganization>
      </TabbedPageSection>
    )
  }
  private getOwnersAndMembers = async (org: Organization) => {
    const allMembers = await Promise.all([
      client.organizations.owners(org.id),
      client.organizations.members(org.id),
    ])

    return [].concat(...allMembers)
  }
}
