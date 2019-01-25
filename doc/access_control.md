# Authorization

### Definitions

* `subject` - entity performing the action (`org`, `user`, or `token`)
* `resource` - object being accessed (`bucket`, `dashboard`, `org`, etc)
* `action` - the mode of access (`read` and `write`, although we could add more over time)
* `permission` - an action and resource (`read:buckets/1`, or `write:orgs/1`)

## Description

There are primarily three classes of actions that need to be authorized

1. Performing an action against a specific resource
2. Performing an action against a class of resource
2. Retrieving all resources subject is authorized to access

## Performing an action against a specific resource
An access control list is stored for each resource. The list contains the
subject type, subject id, and set of actions allowed.

As an example consider the table below

| ResourceType | ResourceID | SubjectType | SubjectID | Actions    |
|--------------|------------|-------------|-----------|------------|
| Dashboard    | 1          | User        | 1         | write      |
| Dashboard    | 1          | Token       | 1         | read       |

In this example, the following statements are true

* `User 1` is granted `write` access to `Dashboard 1`.
* `Token 1` is granted `read` access to `Dashboard 1`.

It is possible however for access to be granted transitively through an organization.

For example, consider the following table

| ResourceType | ResourceID | SubjectType | SubjectID | Actions    |
|--------------|------------|-------------|-----------|------------|
| Dashboard    | 1          | Org         | 1         | read,write |
| Org          | 1          | User        | 1         | read       |

In this example, the following statements are true
 * `Org 1` is granted `read,write` access to `Dashboard 1`
* `User 1` is granted `read` access to `Org 1`.
* `User 1` is granted `read` access to `Dashboard 1` (transitively through `Org 1`'s access grant).

Access can be transitively applied to both users and tokens (potentially to organizations as well if we that was a desirable feature).

This model can be extended by addings a `roles` or `groups` subject type and is commonly referred to as access control groups (ACG or ACLg).
It should be noted that this model is functionally equivalent to a minimal role based access control model [See the role based acces control document section on ACLg](https://en.wikipedia.org/wiki/Role-based_access_control).

## Performing an action against a class of resource

TODO: Who should be allowed to perform actions against top level resources. Specifically, who can create new users or orgs.

## Retrieving all resources subject is authorized to access

In addition to storing an ACL for each resource, we store an Inverse ACL (IACL) so that given a subject
and a resource type, we can retrieve the list of all scan the list for all resources of the resource type
provided.

For example consider the table below

| SubjectType | SubjectID | ResourceType | ResourceID |
|-------------|-----------|--------------|------------|
| User        | 1         | Dashboard    | 1          |
| User        | 1         | Dashboard    | 2          |
| User        | 2         | Dashboard    | 2          |

In this example

* `User 1` has access to the list of `Dashboard 1` and `Dashboard 2`.
* `User 2` has access to the list of `Dashboard 2`.

Just as in the ACL, a subject may have access transitively though an organization.

For example, consider

| SubjectType | SubjectID | ResourceType | ResourceID |
|-------------|-----------|--------------|------------|
| Org         | 1         | Dashboard    | 1          |
| User        | 1         | Dashboard    | 2          |
| User        | 1         | Org          | 1          |

In this example

* `Org 1` has access to the list of `Dashboard 1`
* `User 1` has access to the list of `Dashboard 1` and `Dashboard 2` (transitively though `Org 1`)


Implementers will have to take care to ensure that duplicate resources are not returned when constructing
the list of all resources a subject can access.


## Notes

### ACL
Conceptually, the ACL and IACL described above can be thought of as a table

| ResourceType | ResourceID | SubjectType | SubjectID | Actions    |
|--------------|------------|-------------|-----------|------------|

With two compound indexes.

* ACL - index of `(ResourceType, ResourceID, SubjectType, SubjectID)`
* IACL - index of `(SubjectType, SubjectID, ResourceType, ResourceID)`


### Subjects
#### Tokens
// TODO(desa)
Tokens have a set of permissions, those permissions are used to create entries into the ACL and IACL.

Upon revokation, each entry of a tokens entries in the ACL and IACL is removed. Provided that token revocation
should be relatively rare, the cost of this opperation should not be an issue. If it becomes an issue, we can simply
note at a higher level that the token is no longer valid and reject the request at a different level and clean up
revoked token log entries as a background task.

#### Orgs and Users
// TODO(desa)
Orgs and users will have to be added and removed to resources through an endpoint associated with the resource

possibly something along the lines of `/dashboards/:id/access` (this will only work for orgs or users)

### Other Notes
// TODO(desa)

When a resource is created, if an org is provided with that request, that org will be come the owner of that
resource. If no org is provided, then the resource will be owned by the user issuing the request


# Authorization Modalities

Authorization events occur three distinct modalities.

1. A user accessing resources through a Web UI.
2. An agent accessing a resource using a token.
3. An internal service accessing a resource on behalf of a subject.

In order for an authorization event to take place, the authorizer must be supplied the subject, resource, and action
of the event.


For cases (1) and (2), the subject, resource, and action should be apparent at the time of authorization.

For case (3), the service must store the subject (type and id) and refer back to it at the time of authorization.

