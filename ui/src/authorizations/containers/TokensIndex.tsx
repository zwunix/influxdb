// Libraries
import React, {Component} from 'react'
import {connect} from 'react-redux'

// Components
import {ErrorHandling} from 'src/shared/decorators/errors'
import OrganizationNavigation from 'src/organizations/components/OrganizationNavigation'
import OrgHeader from 'src/organizations/containers/OrgHeader'
import {Tabs} from 'src/clockface'
import {Page} from '@influxdata/clockface'
import TabbedPageSection from 'src/shared/components/tabbed_page/TabbedPageSection'
import GetResources, {
  ResourceTypes,
} from 'src/configuration/components/GetResources'
import TokensTab from 'src/authorizations/components/TokensTab'

// Constants
import {PAGE_TITLE_SUFFIX} from 'src/shared/constants'

// Types
import {Organization} from '@influxdata/influx'
import {AppState} from 'src/types'

interface StateProps {
  org: Organization
}

@ErrorHandling
class TokensIndex extends Component<StateProps> {
  public render() {
    const {org} = this.props

    return (
      <Page
        loadingTitleTag={`Tokens | ${org.name}${PAGE_TITLE_SUFFIX}`}
        titleTag={`Tokens | ${org.name}${PAGE_TITLE_SUFFIX}`}
      >
        <OrgHeader />
        <Page.Contents fullWidth={false} scrollable={true}>
          <div className="col-xs-12">
            <Tabs>
              <OrganizationNavigation tab="tokens" orgID={org.id} />
              <Tabs.TabContents>
                <TabbedPageSection
                  id="org-view-tab--buckets"
                  url="buckets"
                  title="Buckets"
                >
                  <GetResources resource={ResourceTypes.Authorizations}>
                    <TokensTab />
                  </GetResources>
                </TabbedPageSection>
              </Tabs.TabContents>
            </Tabs>
          </div>
        </Page.Contents>
      </Page>
    )
  }
}

const mstp = ({orgs: {org}}: AppState) => ({org})

export default connect<StateProps, {}, {}>(
  mstp,
  null
)(TokensIndex)
