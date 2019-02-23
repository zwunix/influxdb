import {Organization} from '@influxdata/influx'

describe('TelegrafConfig', () => {
  beforeEach(() => {
    cy.flush()

    cy.setupUser().then(({body}) => {
      cy.signin(body.org.id)
      cy.wrap(body.org).as('org')
    })

    cy.get<Organization>('@org').then(({id}) => {
      cy.visit(`/organizations/${id}/telegrafs_tab`)
    })

  })

  it('can create a telegraf config', () => {
    cy.getByDataTest('empty-state').within(() => {
      cy.getByDataTest('create-button').click()
    })

    // select bundle
    cy.getByDataTest('bundle-system').click()
    cy.getByDataTest('continue-button').click()

    // configure
    const telegrafName = "telegraf yo"
    const telegrafDescription = "i am a telegraf config"
    cy.getByInputName('telegraf-name').clear().type(telegrafName)
    cy.getByInputName('telegraf-description').type(telegrafDescription)
    
    cy.getByDataTest('continue-button').click()

    // verify 
    cy.getByDataTest('continue-button').click()

    const row = cy.getByDataTest('collector-row')
    row.should('have.length', 1)
    row.within(() => {
      cy.getByDataTest('editable-name').contains(telegrafName)
      cy.getByDataTest('editable-description').contains(telegrafDescription)
    })
  })
})