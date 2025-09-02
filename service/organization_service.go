package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	db "github.com/rahulmishra/goslack/db/sqlc"
)

// OrganizationService handles organization-related business logic
type OrganizationService struct {
	store db.Store
}

// NewOrganizationService creates a new organization service
func NewOrganizationService(store db.Store) *OrganizationService {
	return &OrganizationService{
		store: store,
	}
}

// CreateOrganization creates a new organization
func (s *OrganizationService) CreateOrganization(ctx context.Context, req CreateOrganizationRequest) (db.Organization, error) {
	organization, err := s.store.CreateOrganization(ctx, req.Name)
	if err != nil {
		return db.Organization{}, fmt.Errorf("failed to create organization: %w", err)
	}

	return organization, nil
}

// GetOrganization retrieves an organization by ID
func (s *OrganizationService) GetOrganization(ctx context.Context, organizationID int64) (db.Organization, error) {
	organization, err := s.store.GetOrganization(ctx, organizationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return db.Organization{}, errors.New("organization not found")
		}
		return db.Organization{}, fmt.Errorf("failed to get organization: %w", err)
	}

	return organization, nil
}

// UpdateOrganization updates an organization's information
func (s *OrganizationService) UpdateOrganization(ctx context.Context, organizationID int64, name string) (db.Organization, error) {
	arg := db.UpdateOrganizationParams{
		ID:   organizationID,
		Name: name,
	}

	organization, err := s.store.UpdateOrganization(ctx, arg)
	if err != nil {
		if err == sql.ErrNoRows {
			return db.Organization{}, errors.New("organization not found")
		}
		return db.Organization{}, fmt.Errorf("failed to update organization: %w", err)
	}

	return organization, nil
}

// ListOrganizations lists organizations with pagination
func (s *OrganizationService) ListOrganizations(ctx context.Context, limit, offset int32) ([]db.Organization, error) {
	arg := db.ListOrganizationsParams{
		Limit:  limit,
		Offset: offset,
	}

	organizations, err := s.store.ListOrganizations(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("failed to list organizations: %w", err)
	}

	return organizations, nil
}

// DeleteOrganization deletes an organization
func (s *OrganizationService) DeleteOrganization(ctx context.Context, organizationID int64) error {
	err := s.store.DeleteOrganization(ctx, organizationID)
	if err != nil {
		return fmt.Errorf("failed to delete organization: %w", err)
	}

	return nil
}
