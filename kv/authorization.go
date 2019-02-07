package kv

import (
	"context"
	"encoding/json"

	"github.com/influxdata/influxdb"
)

// UnexpectedAuthorizationBucketError is used when the error comes from an internal system.
func UnexpectedAuthorizationBucketError(err error) *influxdb.Error {
	return &influxdb.Error{
		Code: influxdb.EInternal,
		Msg:  "unexpected error retrieving authorization bucket",
		Err:  err,
		Op:   "kv/authorizationBucket",
	}
}

// UnexpectedAuthorizationIndexError is used when the error comes from an internal system.
func UnexpectedAuthorizationIndexError(err error) *influxdb.Error {
	return &influxdb.Error{
		Code: influxdb.EInternal,
		Msg:  "unexpected error retrieving authorization index",
		Err:  err,
		Op:   "kv/authorizationIndex",
	}
}

var (
	authorizationBucket = []byte("authorizationsv1")
	authorizationIndex  = []byte("authorizationindexv1")
)

var _ influxdb.AuthorizationService = (*Service)(nil)

func (s *Service) initializeAuthorizations(ctx context.Context, tx Tx) error {
	if _, err := s.authorizationBucket(tx); err != nil {
		return err
	}

	_, err := s.authorizationIndex(tx)
	return err
}

func (s *Service) authorizationBucket(tx Tx) (Bucket, error) {
	b, err := tx.Bucket([]byte(authorizationBucket))
	if err != nil {
		return nil, UnexpectedAuthorizationBucketError(err)
	}

	return b, nil
}

func (s *Service) authorizationIndex(tx Tx) (Bucket, error) {
	b, err := tx.Bucket([]byte(authorizationIndex))
	if err != nil {
		return nil, UnexpectedAuthorizationIndexError(err)
	}

	return b, nil
}

// FindAuthorizationByID retrieves a authorization by id.
func (s *Service) FindAuthorizationByID(ctx context.Context, id influxdb.ID) (*influxdb.Authorization, error) {
	var a *influxdb.Authorization
	var err error
	err = s.kv.View(func(tx Tx) error {
		a, err = s.findAuthorizationByID(ctx, tx, id)
		return err
	})

	return a, err
}

func (s *Service) findAuthorizationByID(ctx context.Context, tx Tx, id influxdb.ID) (*influxdb.Authorization, error) {
	encodedID, err := id.Encode()
	if err != nil {
		return nil, &influxdb.Error{
			Code: influxdb.EInvalid,
			Err:  err,
		}
	}

	bucket, err := s.authorizationBucket(tx)
	if err != nil {
		return nil, err
	}

	v, err := bucket.Get(encodedID)
	if err != nil {
		return nil, err
	}

	if len(v) == 0 {
		return nil, &influxdb.Error{
			Code: influxdb.ENotFound,
			Msg:  "authorization not found",
		}
	}

	var a influxdb.Authorization
	if err := decodeAuthorization(v, &a); err != nil {
		return nil, &influxdb.Error{
			Code: influxdb.EInvalid,
			Err:  err,
		}
	}

	return &a, nil
}

// FindAuthorizationByToken returns a authorization by token for a particular authorization.
func (s *Service) FindAuthorizationByToken(ctx context.Context, n string) (*influxdb.Authorization, error) {
	var a *influxdb.Authorization
	var err error
	err = s.kv.View(func(tx Tx) error {
		a, err = s.findAuthorizationByToken(ctx, tx, n)
		if err != nil {
			return err
		}
		return err
	})

	return a, err
}

func (s *Service) findAuthorizationByToken(ctx context.Context, tx Tx, n string) (*influxdb.Authorization, error) {
	bucket, err := s.authorizationIndex(tx)
	if err != nil {
		return nil, err
	}

	a, err := bucket.Get(authorizationIndexKey(n))
	if err != nil {
		return nil, err
	}

	if a == nil {
		return nil, &influxdb.Error{
			Code: influxdb.ENotFound,
			Msg:  "authorization not found",
		}
	}

	var id influxdb.ID
	if err := id.Decode(a); err != nil {
		return nil, &influxdb.Error{
			Code: influxdb.EInvalid,
			Err:  err,
		}
	}
	return s.findAuthorizationByID(ctx, tx, id)
}

func filterAuthorizationsFn(filter influxdb.AuthorizationFilter) func(a *influxdb.Authorization) bool {
	if filter.ID != nil {
		return func(a *influxdb.Authorization) bool {
			return a.ID == *filter.ID
		}
	}

	if filter.Token != nil {
		return func(a *influxdb.Authorization) bool {
			return a.Token == *filter.Token
		}
	}

	if filter.UserID != nil {
		return func(a *influxdb.Authorization) bool {
			return a.UserID == *filter.UserID
		}
	}

	return func(a *influxdb.Authorization) bool { return true }
}

// FindAuthorizations retrives all authorizations that match an arbitrary authorization filter.
// Filters using ID, or Token should be efficient.
// Other filters will do a linear scan across all authorizations searching for a match.
func (s *Service) FindAuthorizations(ctx context.Context, filter influxdb.AuthorizationFilter, opt ...influxdb.FindOptions) ([]*influxdb.Authorization, int, error) {
	if filter.ID != nil {
		a, err := s.FindAuthorizationByID(ctx, *filter.ID)
		if err != nil {
			return nil, 0, &influxdb.Error{
				Err: err,
				Op:  "kv/" + influxdb.OpFindAuthorizations,
			}
		}

		return []*influxdb.Authorization{a}, 1, nil
	}

	if filter.Token != nil {
		a, err := s.FindAuthorizationByToken(ctx, *filter.Token)
		if err != nil {
			return nil, 0, &influxdb.Error{
				Err: err,
				Op:  "kv/" + influxdb.OpFindAuthorizations,
			}
		}

		return []*influxdb.Authorization{a}, 1, nil
	}

	as := []*influxdb.Authorization{}
	err := s.kv.View(func(tx Tx) error {
		auths, err := s.findAuthorizations(ctx, tx, filter)
		if err != nil {
			return err
		}
		as = auths
		return nil
	})

	if err != nil {
		return nil, 0, &influxdb.Error{
			Err: err,
			Op:  "kv/" + influxdb.OpFindAuthorizations,
		}
	}

	return as, len(as), nil
}

func (s *Service) findAuthorizations(ctx context.Context, tx Tx, f influxdb.AuthorizationFilter) ([]*influxdb.Authorization, error) {
	// If the users name was provided, look up user by ID first
	if f.User != nil {
		u, err := s.findUserByName(ctx, tx, *f.User)
		if err != nil {
			return nil, err
		}
		f.UserID = &u.ID
	}

	as := []*influxdb.Authorization{}
	filterFn := filterAuthorizationsFn(f)
	err := s.forEachAuthorization(ctx, tx, func(a *influxdb.Authorization) bool {
		if filterFn(a) {
			as = append(as, a)
		}
		return true
	})
	if err != nil {
		return nil, err
	}

	return as, nil
}

// CreateAuthorization creates a influxdb authorization and sets b.ID, and b.UserID if not provided.
func (s *Service) CreateAuthorization(ctx context.Context, a *influxdb.Authorization) error {
	op := "kv/" + influxdb.OpCreateAuthorization
	if err := a.Valid(); err != nil {
		return &influxdb.Error{
			Err: err,
			Op:  op,
		}
	}

	return s.kv.Update(func(tx Tx) error {
		_, pErr := s.findUserByID(ctx, tx, a.UserID)
		if pErr != nil {
			return influxdb.ErrUnableToCreateToken
		}

		/* TODO(goller): add this when creating token
		_, pErr = s.findOrganizationByID(ctx, tx, a.OrgID)
		if pErr != nil {
			return influxdb.ErrUnableToCreateToken
		}
		*/

		if unique := s.uniqueAuthorizationToken(ctx, tx, a); !unique {
			return influxdb.ErrUnableToCreateToken
		}

		if a.Token == "" {
			token, err := s.TokenGenerator.Token()
			if err != nil {
				return &influxdb.Error{
					Err: err,
					Op:  op,
				}
			}
			a.Token = token
		}

		a.ID = s.IDGenerator.ID()

		return s.putAuthorization(ctx, tx, a)
	})
}

// PutAuthorization will put a authorization without setting an ID.
func (s *Service) PutAuthorization(ctx context.Context, a *influxdb.Authorization) (err error) {
	return s.kv.Update(func(tx Tx) error {
		pe := s.putAuthorization(ctx, tx, a)
		if pe != nil {
			err = pe
		}
		return err
	})
}

func encodeAuthorization(a *influxdb.Authorization) ([]byte, error) {
	switch a.Status {
	case influxdb.Active, influxdb.Inactive:
	case "":
		a.Status = influxdb.Active
	default:
		return nil, &influxdb.Error{
			Code: influxdb.EInvalid,
			Msg:  "unknown authorization status",
		}
	}

	return json.Marshal(a)
}

func (s *Service) putAuthorization(ctx context.Context, tx Tx, a *influxdb.Authorization) error {
	v, err := encodeAuthorization(a)
	if err != nil {
		return &influxdb.Error{
			Code: influxdb.EInvalid,
			Err:  err,
		}
	}

	encodedID, err := a.ID.Encode()
	if err != nil {
		return &influxdb.Error{
			Code: influxdb.ENotFound,
			Err:  err,
		}
	}

	bucketIdx, err := s.authorizationIndex(tx)
	if err != nil {
		return err
	}

	if err := bucketIdx.Put(authorizationIndexKey(a.Token), encodedID); err != nil {
		return &influxdb.Error{
			Code: influxdb.EInternal,
			Err:  err,
		}
	}

	bucket, err := s.authorizationBucket(tx)
	if err != nil {
		return err
	}

	if err := bucket.Put(encodedID, v); err != nil {
		return &influxdb.Error{
			Err: err,
		}
	}

	return nil
}

func authorizationIndexKey(n string) []byte {
	return []byte(n)
}

func decodeAuthorization(b []byte, a *influxdb.Authorization) error {
	if err := json.Unmarshal(b, a); err != nil {
		return err
	}
	if a.Status == "" {
		a.Status = influxdb.Active
	}
	return nil
}

// forEachAuthorization will iterate through all authorizations while fn returns true.
func (s *Service) forEachAuthorization(ctx context.Context, tx Tx, fn func(*influxdb.Authorization) bool) error {
	bucket, err := s.authorizationBucket(tx)
	if err != nil {
		return err
	}

	cur, err := bucket.Cursor()
	if err != nil {
		return err
	}

	for k, v := cur.First(); k != nil; k, v = cur.Next() {
		a := &influxdb.Authorization{}

		if err := decodeAuthorization(v, a); err != nil {
			return err
		}
		if !fn(a) {
			break
		}
	}

	return nil
}

func (s *Service) uniqueAuthorizationToken(ctx context.Context, tx Tx, a *influxdb.Authorization) bool {
	bucket, err := s.authorizationIndex(tx)
	if err != nil {
		return true // TODO(goller): this can hide some sort of storage problem
	}

	_, err = bucket.Get(authorizationIndexKey(a.Token))
	return err != nil
}

// DeleteAuthorization deletes a authorization and prunes it from the index.
func (s *Service) DeleteAuthorization(ctx context.Context, id influxdb.ID) error {
	err := s.kv.Update(func(tx Tx) (err error) {
		return s.deleteAuthorization(ctx, tx, id)
	})
	return err
}

func (s *Service) deleteAuthorization(ctx context.Context, tx Tx, id influxdb.ID) error {
	a, pe := s.findAuthorizationByID(ctx, tx, id)
	if pe != nil {
		return pe
	}

	bucketIdx, err := s.authorizationIndex(tx)
	if err != nil {
		return err
	}

	if err := bucketIdx.Delete(authorizationIndexKey(a.Token)); err != nil {
		return &influxdb.Error{
			Err: err,
		}
	}
	encodedID, err := id.Encode()
	if err != nil {
		return &influxdb.Error{
			Err: err,
		}
	}

	bucket, err := s.authorizationBucket(tx)
	if err != nil {
		return err
	}

	if err := bucket.Delete(encodedID); err != nil {
		return &influxdb.Error{
			Err: err,
		}
	}

	return nil
}

// SetAuthorizationStatus updates the status of the authorization. Useful
// for setting an authorization to inactive or active.
func (s *Service) SetAuthorizationStatus(ctx context.Context, id influxdb.ID, status influxdb.Status) error {
	return s.kv.Update(func(tx Tx) error {
		if pe := s.updateAuthorization(ctx, tx, id, status); pe != nil {
			return &influxdb.Error{
				Err: pe,
				Op:  influxdb.OpSetAuthorizationStatus,
			}
		}
		return nil
	})
}

func (s *Service) updateAuthorization(ctx context.Context, tx Tx, id influxdb.ID, status influxdb.Status) error {
	a, pe := s.findAuthorizationByID(ctx, tx, id)
	if pe != nil {
		return pe
	}

	a.Status = status
	b, err := encodeAuthorization(a)
	if err != nil {
		return &influxdb.Error{
			Err: err,
		}
	}

	encodedID, err := id.Encode()
	if err != nil {
		return &influxdb.Error{
			Err: err,
		}
	}

	bucket, err := s.authorizationBucket(tx)
	if err != nil {
		return err
	}

	if err = bucket.Put(encodedID, b); err != nil {
		return &influxdb.Error{
			Err: err,
		}
	}
	return nil
}
