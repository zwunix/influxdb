package kv

import (
	"context"
	"encoding/json"

	"github.com/influxdata/influxdb"
)

var (
	telegrafBucket = []byte("telegraf/v1")
)

var _ influxdb.TelegrafConfigStore = (*Service)(nil)

func (s *Service) initializeTelegrafs(ctx context.Context, tx Tx) error {
	if _, err := tx.Bucket(telegrafBucket); err != nil {
		return err
	}
	return nil
}

// FindTelegrafConfigByID returns a single telegraf config by ID.
func (s *Service) FindTelegrafConfigByID(ctx context.Context, id influxdb.ID) (tc *influxdb.TelegrafConfig, err error) {
	op := OpPrefix + influxdb.OpFindTelegrafConfigByID
	err = s.kv.View(func(tx Tx) error {
		tCfg, pe := s.findTelegrafConfigByID(ctx, tx, id)
		if pe != nil {
			return pe
		}
		tc = tCfg
		return nil
	})
	if err != nil {
		return nil, &influxdb.Error{
			Err: err,
			Op:  op,
		}
	}
	return tc, nil
}

func (s *Service) findTelegrafConfigByID(ctx context.Context, tx Tx, id influxdb.ID) (*influxdb.TelegrafConfig, error) {
	encodedID, err := id.Encode()
	if err != nil {
		return nil, &influxdb.Error{
			Code: influxdb.EEmptyValue,
			Err:  influxdb.ErrInvalidID,
		}
	}
	b, err := tx.Bucket(telegrafBucket)
	if err != nil {
		return nil, err
	}

	v, err := b.Get(encodedID)
	if IsNotFound(err) {
		return nil, &influxdb.Error{
			Code: influxdb.ENotFound,
			Msg:  influxdb.ErrTelegrafConfigNotFound,
		}
	}

	if err != nil {
		return nil, err
	}

	var tc influxdb.TelegrafConfig
	if err := json.Unmarshal(v, &tc); err != nil {
		return nil, &influxdb.Error{
			Err: err,
		}
	}
	return &tc, nil
}

// FindTelegrafConfig returns the first telegraf config that matches filter.
func (s *Service) FindTelegrafConfig(ctx context.Context, filter influxdb.TelegrafConfigFilter) (*influxdb.TelegrafConfig, error) {
	op := OpPrefix + influxdb.OpFindTelegrafConfig
	tcs, n, err := s.FindTelegrafConfigs(ctx, filter, influxdb.FindOptions{Limit: 1})
	if err != nil {
		return nil, err
	}
	if n > 0 {
		return tcs[0], nil
	}
	return nil, &influxdb.Error{
		Code: influxdb.ENotFound,
		Op:   op,
	}
}

func (s *Service) findTelegrafConfigs(ctx context.Context, tx Tx, filter influxdb.TelegrafConfigFilter, opt ...influxdb.FindOptions) ([]*influxdb.TelegrafConfig, int, error) {
	tcs := make([]*influxdb.TelegrafConfig, 0)
	m, _, err := s.FindUserResourceMappings(ctx, filter.UserResourceMappingFilter)
	if err != nil {
		return nil, 0, err
	}
	if len(m) == 0 {
		return tcs, 0, nil
	}
	for _, item := range m {
		tc, err := s.findTelegrafConfigByID(ctx, tx, item.ResourceID)
		if err != nil && influxdb.ErrorCode(err) != influxdb.ENotFound {
			return nil, 0, err
		}
		if tc != nil {
			// Restrict results by organization ID, if it has been provided
			if filter.OrganizationID != nil && filter.OrganizationID.Valid() && tc.OrganizationID != *filter.OrganizationID {
				continue
			}
			tcs = append(tcs, tc)
		}
	}

	return tcs, len(tcs), nil
}

// FindTelegrafConfigs returns a list of telegraf configs that match filter and the total count of matching telegraf configs.
// Additional options provide pagination & sorting.
func (s *Service) FindTelegrafConfigs(ctx context.Context, filter influxdb.TelegrafConfigFilter, opt ...influxdb.FindOptions) (tcs []*influxdb.TelegrafConfig, n int, err error) {
	var pe error
	_ = s.kv.View(func(tx Tx) error {
		tcs, n, pe = s.findTelegrafConfigs(ctx, tx, filter)
		if pe != nil {
			err = &influxdb.Error{
				Err: err,
				Op:  OpPrefix + influxdb.OpFindTelegrafConfigs,
			}
		}
		return nil
	})
	return tcs, n, err
}

// PutTelegrafConfig will put a telegraf config without setting an ID.
func (s *Service) PutTelegrafConfig(ctx context.Context, tc *influxdb.TelegrafConfig) error {
	return s.kv.Update(func(tx Tx) error {
		return s.putTelegrafConfig(ctx, tx, tc)
	})
}

func (s *Service) putTelegrafConfig(ctx context.Context, tx Tx, tc *influxdb.TelegrafConfig) error {
	if !tc.ID.Valid() {
		return &influxdb.Error{
			Code: influxdb.EEmptyValue,
			Err:  influxdb.ErrInvalidID,
		}
	}
	if !tc.OrganizationID.Valid() {
		return &influxdb.Error{
			Code: influxdb.EEmptyValue,
			Msg:  influxdb.ErrTelegrafConfigInvalidOrganizationID,
		}
	}
	v, err := json.Marshal(tc)
	if err != nil {
		return &influxdb.Error{
			Err: err,
		}
	}
	encodedID, err := tc.ID.Encode()
	if err != nil {
		return &influxdb.Error{
			Code: influxdb.EInvalid,
			Err:  err,
		}
	}

	b, err := tx.Bucket(telegrafBucket)
	if err != nil {
		return err
	}

	if err = b.Put(encodedID, v); err != nil {
		return &influxdb.Error{
			Err: err,
		}
	}
	return nil
}

// CreateTelegrafConfig creates a new telegraf config and sets b.ID with the new identifier.
func (s *Service) CreateTelegrafConfig(ctx context.Context, tc *influxdb.TelegrafConfig, userID influxdb.ID) error {
	err := s.kv.Update(func(tx Tx) error {
		tc.ID = s.IDGenerator.ID()
		return s.putTelegrafConfig(ctx, tx, tc)
	})
	if err != nil {
		return &influxdb.Error{
			Err: err,
			Op:  OpPrefix + influxdb.OpCreateTelegrafConfig,
		}
	}

	urm := &influxdb.UserResourceMapping{
		ResourceID:   tc.ID,
		UserID:       userID,
		UserType:     influxdb.Owner,
		ResourceType: influxdb.TelegrafsResourceType,
	}
	if err := s.CreateUserResourceMapping(ctx, urm); err != nil {
		return err
	}

	return nil
}

// UpdateTelegrafConfig updates a single telegraf config.
// Returns the new telegraf config after update.
func (s *Service) UpdateTelegrafConfig(ctx context.Context, id influxdb.ID, tc *influxdb.TelegrafConfig, userID influxdb.ID) (*influxdb.TelegrafConfig, error) {
	err := s.kv.Update(func(tx Tx) error {
		current, pe := s.findTelegrafConfigByID(ctx, tx, id)
		if pe != nil {
			return pe
		}
		tc.ID = id
		// OrganizationID can not be updated
		tc.OrganizationID = current.OrganizationID

		return s.putTelegrafConfig(ctx, tx, tc)
	})
	if err != nil {
		return tc, &influxdb.Error{
			Err: err,
			Op:  OpPrefix + influxdb.OpUpdateTelegrafConfig,
		}
	}

	return tc, nil
}

// DeleteTelegrafConfig removes a telegraf config by ID.
func (s *Service) DeleteTelegrafConfig(ctx context.Context, id influxdb.ID) error {
	op := OpPrefix + influxdb.OpDeleteTelegrafConfig
	if !id.Valid() {
		return &influxdb.Error{
			Op:   op,
			Code: influxdb.EEmptyValue,
			Err:  influxdb.ErrInvalidID,
		}
	}
	err := s.kv.Update(func(tx Tx) error {
		if _, pe := s.findTelegrafConfigByID(ctx, tx, id); pe != nil {
			return pe
		}
		encodedID, pe := id.Encode()
		if pe != nil {
			return &influxdb.Error{
				Code: influxdb.EInvalid,
				Err:  pe,
			}
		}

		b, pe := tx.Bucket(telegrafBucket)
		if pe != nil {
			return pe
		}

		if pe = b.Delete(encodedID); pe != nil {
			return pe
		}
		return s.deleteUserResourceMapping(ctx, tx, influxdb.UserResourceMappingFilter{
			ResourceID:   id,
			ResourceType: influxdb.TelegrafsResourceType,
		})
	})

	if err != nil {
		return &influxdb.Error{
			Code: influxdb.ErrorCode(err),
			Op:   op,
			Err:  err,
		}
	}
	return nil
}
