/*
 * Teleport
 * Copyright (C) 2024  Gravitational, Inc.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package accesslistsv1

import (
	"context"

	emptypb "google.golang.org/protobuf/types/known/emptypb"

	accesslistv1 "github.com/gravitational/teleport/api/gen/proto/go/teleport/accesslist/v1"
	"github.com/gravitational/teleport/api/gen/proto/go/teleport/header/v1"
	traitv1 "github.com/gravitational/teleport/api/types/trait/convert/v1"
	"github.com/gravitational/teleport/api/types/accesslist"
	"github.com/gravitational/teleport/api/types/header"
	apievents "github.com/gravitational/teleport/api/types/events"
	"github.com/gravitational/teleport/lib/authz"
	"github.com/gravitational/teleport/lib/backend"
	"github.com/gravitational/teleport/lib/services/local"
	"github.com/sirupsen/logrus"
	"github.com/gravitational/trace"
	"google.golang.org/protobuf/types/known/timestamppb"

	"google.golang.org/protobuf/types/known/durationpb"
)

// ServiceConfig contains the configuration for creating a new AccessListService.
type ServiceConfig struct {
	// Authorizer is the authorizer for checking access.
	Authorizer authz.Authorizer
	// Backend is the backend store.
	Backend backend.Backend
	// Logger is the logger for the service.
	Logger logrus.FieldLogger
	// Emitter is the event emitter.
	Emitter apievents.Emitter
}

// VerifyServiceConfig validates the ServiceConfig.
func (c *ServiceConfig) CheckAndSetDefaults() error {
	if c.Authorizer == nil {
		return trace.BadParameter("Authorizer is required")
	}
	if c.Backend == nil {
		return trace.BadParameter("Backend is required")
	}
	if c.Logger == nil {
		c.Logger = logrus.WithFields(logrus.Fields{
			"component": "access-list-service",
		})
	}
	return nil
}

// Service implements the AccessList gRPC service.
type Service struct {
	accesslistv1.UnimplementedAccessListServiceServer

	authorizer authz.Authorizer
	backend    backend.Backend
	logger     logrus.FieldLogger
	emitter    apievents.Emitter

	localService *local.AccessListService
}

// NewService creates a new AccessListService.
func NewService(c *ServiceConfig) (*Service, error) {
	if err := c.CheckAndSetDefaults(); err != nil {
		return nil, trace.Wrap(err)
	}

	localService, err := local.NewAccessListService(c.Backend, nil)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return &Service{
		authorizer:   c.Authorizer,
		backend:      c.Backend,
		logger:       c.Logger,
		emitter:      c.Emitter,
		localService: localService,
	}, nil
}

// GetAccessLists returns all access lists.
func (s *Service) GetAccessLists(ctx context.Context, _ *accesslistv1.GetAccessListsRequest) (*accesslistv1.GetAccessListsResponse, error) {
	accessLists, err := s.localService.GetAccessLists(ctx)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return &accesslistv1.GetAccessListsResponse{AccessLists: toProtoAccessLists(accessLists)}, nil
}

// ListAccessLists returns a paginated list of access lists.
func (s *Service) ListAccessLists(ctx context.Context, req *accesslistv1.ListAccessListsRequest) (*accesslistv1.ListAccessListsResponse, error) {
	accessLists, nextToken, err := s.localService.ListAccessLists(ctx, int(req.GetPageSize()), req.GetNextToken())
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return &accesslistv1.ListAccessListsResponse{
		AccessLists: toProtoAccessLists(accessLists),
		NextToken:   nextToken,
	}, nil
}

// GetAccessList returns a specific access list by name.
func (s *Service) GetAccessList(ctx context.Context, req *accesslistv1.GetAccessListRequest) (*accesslistv1.AccessList, error) {
	accessList, err := s.localService.GetAccessList(ctx, req.GetName())
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return toProtoAccessList(accessList), nil
}

// UpsertAccessList creates or updates an access list.
func (s *Service) UpsertAccessList(ctx context.Context, req *accesslistv1.UpsertAccessListRequest) (*accesslistv1.AccessList, error) {
	internalList, err := fromProtoAccessList(req.GetAccessList())
	if err != nil {
		return nil, trace.Wrap(err)
	}
	accessList, err := s.localService.UpsertAccessList(ctx, internalList)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return toProtoAccessList(accessList), nil
}

// UpdateAccessList conditionally updates an access list.
func (s *Service) UpdateAccessList(ctx context.Context, req *accesslistv1.UpdateAccessListRequest) (*accesslistv1.AccessList, error) {
	internalList, err := fromProtoAccessList(req.GetAccessList())
	if err != nil {
		return nil, trace.Wrap(err)
	}
	accessList, err := s.localService.UpdateAccessList(ctx, internalList)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return toProtoAccessList(accessList), nil
}

// DeleteAccessList deletes an access list by name.
func (s *Service) DeleteAccessList(ctx context.Context, req *accesslistv1.DeleteAccessListRequest) (*emptypb.Empty, error) {
	if err := s.localService.DeleteAccessList(ctx, req.GetName()); err != nil {
		return nil, trace.Wrap(err)
	}
	return &emptypb.Empty{}, nil
}

// DeleteAllAccessLists deletes all access lists.
func (s *Service) DeleteAllAccessLists(ctx context.Context, _ *accesslistv1.DeleteAllAccessListsRequest) (*emptypb.Empty, error) {
	if err := s.localService.DeleteAllAccessLists(ctx); err != nil {
		return nil, trace.Wrap(err)
	}
	return &emptypb.Empty{}, nil
}

// GetAccessListsToReview returns access lists that need review.
func (s *Service) GetAccessListsToReview(ctx context.Context, _ *accesslistv1.GetAccessListsToReviewRequest) (*accesslistv1.GetAccessListsToReviewResponse, error) {
	// Note: GetAccessListsToReview is not implemented in the local service.
	// This is a placeholder that returns an empty response.
	return &accesslistv1.GetAccessListsToReviewResponse{}, nil
}

// CountAccessListMembers returns the number of members in an access list.
func (s *Service) CountAccessListMembers(ctx context.Context, req *accesslistv1.CountAccessListMembersRequest) (*accesslistv1.CountAccessListMembersResponse, error) {
	count, err := s.localService.CountAccessListMembers(ctx, req.AccessListName)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return &accesslistv1.CountAccessListMembersResponse{Count: count}, nil
}

// ListAccessListMembers returns a paginated list of access list members.
func (s *Service) ListAccessListMembers(ctx context.Context, req *accesslistv1.ListAccessListMembersRequest) (*accesslistv1.ListAccessListMembersResponse, error) {
	members, nextToken, err := s.localService.ListAccessListMembers(ctx, req.AccessList, int(req.PageSize), req.PageToken)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return &accesslistv1.ListAccessListMembersResponse{
		Members:       toProtoMembers(members),
		NextPageToken: nextToken,
	}, nil
}

// GetAccessListMember returns a specific access list member.
func (s *Service) GetAccessListMember(ctx context.Context, req *accesslistv1.GetAccessListMemberRequest) (*accesslistv1.Member, error) {
	member, err := s.localService.GetAccessListMember(ctx, req.AccessList, req.MemberName)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return toProtoMember(member), nil
}

// UpsertAccessListMember creates or updates an access list member.
func (s *Service) UpsertAccessListMember(ctx context.Context, req *accesslistv1.UpsertAccessListMemberRequest) (*accesslistv1.Member, error) {
	internalMember, err := fromProtoMember(req.GetMember())
	if err != nil {
		return nil, trace.Wrap(err)
	}
	member, err := s.localService.UpsertAccessListMember(ctx, internalMember)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return toProtoMember(member), nil
}

// UpdateAccessListMember conditionally updates an access list member.
func (s *Service) UpdateAccessListMember(ctx context.Context, req *accesslistv1.UpdateAccessListMemberRequest) (*accesslistv1.Member, error) {
	internalMember, err := fromProtoMember(req.GetMember())
	if err != nil {
		return nil, trace.Wrap(err)
	}
	member, err := s.localService.UpdateAccessListMember(ctx, internalMember)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return toProtoMember(member), nil
}

// DeleteAccessListMember deletes an access list member.
func (s *Service) DeleteAccessListMember(ctx context.Context, req *accesslistv1.DeleteAccessListMemberRequest) (*emptypb.Empty, error) {
	if err := s.localService.DeleteAccessListMember(ctx, req.AccessList, req.MemberName); err != nil {
		return nil, trace.Wrap(err)
	}
	return &emptypb.Empty{}, nil
}

// DeleteAllAccessListMembersForAccessList deletes all members for an access list.
func (s *Service) DeleteAllAccessListMembersForAccessList(ctx context.Context, req *accesslistv1.DeleteAllAccessListMembersForAccessListRequest) (*emptypb.Empty, error) {
	if err := s.localService.DeleteAllAccessListMembersForAccessList(ctx, req.AccessList); err != nil {
		return nil, trace.Wrap(err)
	}
	return &emptypb.Empty{}, nil
}

// DeleteAllAccessListMembers deletes all access list members.
func (s *Service) DeleteAllAccessListMembers(ctx context.Context, _ *accesslistv1.DeleteAllAccessListMembersRequest) (*emptypb.Empty, error) {
	if err := s.localService.DeleteAllAccessListMembers(ctx); err != nil {
		return nil, trace.Wrap(err)
	}
	return &emptypb.Empty{}, nil
}

// UpsertAccessListWithMembers creates or updates an access list and its members.
func (s *Service) UpsertAccessListWithMembers(ctx context.Context, req *accesslistv1.UpsertAccessListWithMembersRequest) (*accesslistv1.UpsertAccessListWithMembersResponse, error) {
	internalList, err := fromProtoAccessList(req.GetAccessList())
	if err != nil {
		return nil, trace.Wrap(err)
	}
	internalMembers, err := fromProtoMembers(req.GetMembers())
	if err != nil {
		return nil, trace.Wrap(err)
	}
	accessList, members, err := s.localService.UpsertAccessListWithMembers(ctx, internalList, internalMembers)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return &accesslistv1.UpsertAccessListWithMembersResponse{
		AccessList: toProtoAccessList(accessList),
		Members:    toProtoMembers(members),
	}, nil
}

// CreateAccessListWithMembers creates an access list and its members.
// Note: Not part of the current gRPC proto interface; kept as a local helper.
func (s *Service) CreateAccessListWithMembers(ctx context.Context, req *emptypb.Empty) (*emptypb.Empty, error) {
	return nil, trace.Errorf("CreateAccessListWithMembers not yet implemented")
}

// DeleteAccessListWithMembers deletes an access list and its members.
// Note: Not part of the current gRPC proto interface; kept as a local helper.
func (s *Service) DeleteAccessListWithMembers(ctx context.Context, req *emptypb.Empty) (*emptypb.Empty, error) {
	return nil, trace.Errorf("DeleteAccessListWithMembers not yet implemented")
}

// toProtoIneligibleStatus converts a domain IneligibleStatus string to the proto enum.
func toProtoIneligibleStatus(s string) accesslistv1.IneligibleStatus {
	if v, ok := accesslistv1.IneligibleStatus_value[s]; ok {
		return accesslistv1.IneligibleStatus(v)
	}
	return accesslistv1.IneligibleStatus_INELIGIBLE_STATUS_UNSPECIFIED
}

// toProtoOwners converts []accesslist.Owner to []*accesslistv1.AccessListOwner.
func toProtoOwners(owners []accesslist.Owner) []*accesslistv1.AccessListOwner {
	if owners == nil {
		return nil
	}
	result := make([]*accesslistv1.AccessListOwner, len(owners))
	for i, o := range owners {
		result[i] = &accesslistv1.AccessListOwner{
			Name:             o.Name,
			Description:      o.Description,
			IneligibleStatus: toProtoIneligibleStatus(o.IneligibleStatus),
		}
	}
	return result
}

// toProtoAudit converts accesslist.Audit to *accesslistv1.AccessListAudit.
func toProtoAudit(a accesslist.Audit) *accesslistv1.AccessListAudit {
	if (accesslist.Audit{}) == a {
		return nil
	}
	return &accesslistv1.AccessListAudit{
		NextAuditDate: timestamppb.New(a.NextAuditDate),
		Recurrence: &accesslistv1.Recurrence{
			Frequency:    accesslistv1.ReviewFrequency(a.Recurrence.Frequency),
			DayOfMonth:   accesslistv1.ReviewDayOfMonth(a.Recurrence.DayOfMonth),
		},
		Notifications: &accesslistv1.Notifications{
			Start: durationpb.New(a.Notifications.Start),
		},
	}
}

// toProtoRequires converts accesslist.Requires to *accesslistv1.AccessListRequires.
func toProtoRequires(r accesslist.Requires) *accesslistv1.AccessListRequires {
	if r.IsEmpty() {
		return nil
	}
	return &accesslistv1.AccessListRequires{
		Roles:  r.Roles,
		Traits: traitv1.ToProto(r.Traits),
	}
}

// toProtoGrants converts accesslist.Grants to *accesslistv1.AccessListGrants.
func toProtoGrants(g accesslist.Grants) *accesslistv1.AccessListGrants {
	if len(g.Roles) == 0 && len(g.Traits) == 0 {
		return nil
	}
	return &accesslistv1.AccessListGrants{
		Roles:  g.Roles,
		Traits: traitv1.ToProto(g.Traits),
	}
}

// toProtoStatus converts accesslist.Status to *accesslistv1.AccessListStatus.
func toProtoStatus(s accesslist.Status) *accesslistv1.AccessListStatus {
	if s.MemberCount == nil {
		return nil
	}
	return &accesslistv1.AccessListStatus{
		MemberCount: s.MemberCount,
	}
}

// toProtoAccessList converts a *accesslist.AccessList to its proto representation.
func toProtoAccessList(a *accesslist.AccessList) *accesslistv1.AccessList {
	if a == nil {
		return nil
	}
	return &accesslistv1.AccessList{
		Header: &headerv1.ResourceHeader{
			Kind:    a.GetKind(),
			Version: a.GetVersion(),
			Metadata: &headerv1.Metadata{
				Name:      a.GetName(),
				Namespace: "",
			},
		},
		Spec:   &accesslistv1.AccessListSpec{
			Title:                 a.Spec.Title,
			Description:           a.Spec.Description,
			Owners:                toProtoOwners(a.Spec.Owners),
			Audit:                 toProtoAudit(a.Spec.Audit),
			MembershipRequires:    toProtoRequires(a.Spec.MembershipRequires),
			OwnershipRequires:     toProtoRequires(a.Spec.OwnershipRequires),
			Grants:                toProtoGrants(a.Spec.Grants),
			OwnerGrants:           toProtoGrants(a.Spec.OwnerGrants),
		},
		Status: toProtoStatus(a.Status),
	}
}

// toProtoAccessLists converts a slice of *accesslist.AccessList to their proto representation.
func toProtoAccessLists(accessLists []*accesslist.AccessList) []*accesslistv1.AccessList {
	if accessLists == nil {
		return nil
	}
	result := make([]*accesslistv1.AccessList, len(accessLists))
	for i, a := range accessLists {
		result[i] = toProtoAccessList(a)
	}
	return result
}

// toProtoMember converts a *accesslist.AccessListMember to its proto representation.
func toProtoMember(m *accesslist.AccessListMember) *accesslistv1.Member {
	if m == nil {
		return nil
	}
	return &accesslistv1.Member{
		Header: &headerv1.ResourceHeader{
			Kind:    m.GetKind(),
			Version: m.GetVersion(),
			Metadata: &headerv1.Metadata{
				Name:      m.GetName(),
				Namespace: m.GetMetadata().Namespace,
			},
		},
		Spec: &accesslistv1.MemberSpec{
			AccessList:       m.Spec.AccessList,
			Name:             m.Spec.Name,
			Joined:           timestamppb.New(m.Spec.Joined),
			Expires:          timestamppb.New(m.Spec.Expires),
			Reason:           m.Spec.Reason,
			AddedBy:          m.Spec.AddedBy,
			IneligibleStatus: accesslistv1.IneligibleStatus(accesslistv1.IneligibleStatus_value[m.Spec.IneligibleStatus]),
		},
	}
}

// toProtoMembers converts a slice of *accesslist.AccessListMember to their proto representation.
func toProtoMembers(members []*accesslist.AccessListMember) []*accesslistv1.Member {
	if members == nil {
		return nil
	}
	result := make([]*accesslistv1.Member, len(members))
	for i, m := range members {
		result[i] = toProtoMember(m)
	}
	return result
}

// fromProtoAccessList converts a proto AccessList to *accesslist.AccessList.
func fromProtoAccessList(p *accesslistv1.AccessList) (*accesslist.AccessList, error) {
	if p == nil {
		return nil, trace.Errorf("access list is nil")
	}
	var owners []accesslist.Owner
	if p.Spec.Owners != nil {
		owners = make([]accesslist.Owner, len(p.Spec.Owners))
		for i, o := range p.Spec.Owners {
			owners[i] = accesslist.Owner{
				Name:             o.Name,
				Description:      o.Description,
				IneligibleStatus: accesslistv1.IneligibleStatus_name[int32(o.IneligibleStatus)],
			}
		}
	}
	var audit accesslist.Audit
	if p.Spec.Audit != nil {
		audit = accesslist.Audit{
			NextAuditDate: p.Spec.Audit.NextAuditDate.AsTime(),
			Recurrence: accesslist.Recurrence{
				Frequency:    accesslist.ReviewFrequency(p.Spec.Audit.Recurrence.Frequency),
				DayOfMonth:   accesslist.ReviewDayOfMonth(p.Spec.Audit.Recurrence.DayOfMonth),
			},
			Notifications: accesslist.Notifications{
				Start: p.Spec.Audit.Notifications.Start.AsDuration(),
			},
		}
	}
	var membershipRequires accesslist.Requires
	if p.Spec.MembershipRequires != nil {
		membershipRequires = accesslist.Requires{
			Roles:  p.Spec.MembershipRequires.Roles,
			Traits: traitv1.FromProto(p.Spec.MembershipRequires.Traits),
		}
	}
	var ownershipRequires accesslist.Requires
	if p.Spec.OwnershipRequires != nil {
		ownershipRequires = accesslist.Requires{
			Roles:  p.Spec.OwnershipRequires.Roles,
			Traits: traitv1.FromProto(p.Spec.OwnershipRequires.Traits),
		}
	}
	var grants accesslist.Grants
	if p.Spec.Grants != nil {
		grants = accesslist.Grants{
			Roles:  p.Spec.Grants.Roles,
			Traits: traitv1.FromProto(p.Spec.Grants.Traits),
		}
	}
	var ownerGrants accesslist.Grants
	if p.Spec.OwnerGrants != nil {
		ownerGrants = accesslist.Grants{
			Roles:  p.Spec.OwnerGrants.Roles,
			Traits: traitv1.FromProto(p.Spec.OwnerGrants.Traits),
		}
	}
	a := &accesslist.AccessList{
		ResourceHeader: header.ResourceHeader{
			Kind:      p.Header.GetKind(),
			SubKind:   p.Header.GetSubKind(),
			Version:   p.Header.GetVersion(),
			Metadata:  header.Metadata{Name: p.Header.Metadata.GetName(), Labels: p.Header.Metadata.GetLabels()},
		},
		Spec: accesslist.Spec{
			Title:             p.Spec.Title,
			Description:       p.Spec.Description,
			Owners:            owners,
			Audit:             audit,
			MembershipRequires: membershipRequires,
			OwnershipRequires: ownershipRequires,
			Grants:            grants,
			OwnerGrants:       ownerGrants,
		},
		Status: accesslist.Status{},
	}
	return a, nil
}

// fromProtoMembers converts a slice of proto Members to []*accesslist.AccessListMember.
func fromProtoMembers(protoMembers []*accesslistv1.Member) ([]*accesslist.AccessListMember, error) {
	if protoMembers == nil {
		return nil, nil
	}
	result := make([]*accesslist.AccessListMember, len(protoMembers))
	for i, m := range protoMembers {
		member, err := fromProtoMember(m)
		if err != nil {
			return nil, trace.Wrap(err)
		}
		result[i] = member
	}
	return result, nil
}

// fromProtoMember converts a proto Member to *accesslist.AccessListMember.
func fromProtoMember(p *accesslistv1.Member) (*accesslist.AccessListMember, error) {
	if p == nil {
		return nil, trace.Errorf("member is nil")
	}
	return &accesslist.AccessListMember{
		ResourceHeader: header.ResourceHeader{
			Kind:      p.Header.GetKind(),
			SubKind:   p.Header.GetSubKind(),
			Version:   p.Header.GetVersion(),
			Metadata:  header.Metadata{Name: p.Header.Metadata.GetName(), Labels: p.Header.Metadata.GetLabels()},
		},
		Spec: accesslist.AccessListMemberSpec{
			AccessList:       p.Spec.AccessList,
			Name:             p.Spec.Name,
			Joined:           p.Spec.Joined.AsTime(),
			Expires:          p.Spec.Expires.AsTime(),
			Reason:           p.Spec.Reason,
			AddedBy:          p.Spec.AddedBy,
			IneligibleStatus: p.Spec.IneligibleStatus.String(),
		},
	}, nil
}
