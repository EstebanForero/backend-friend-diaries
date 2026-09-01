package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	groupsvc "github.com/friend-diaries/backend/internal/application/group"
	"github.com/friend-diaries/backend/internal/domain/group"
	"github.com/friend-diaries/backend/internal/interfaces/dto"
	"github.com/friend-diaries/backend/internal/interfaces/http/middleware"
	"github.com/friend-diaries/backend/pkg/apperror"
	"github.com/friend-diaries/backend/pkg/response"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type GroupHandler struct {
	groupService *groupsvc.Service
	groupRepo    group.Repository
}

func NewGroupHandler(groupService *groupsvc.Service, groupRepo group.Repository) *GroupHandler {
	return &GroupHandler{groupService: groupService, groupRepo: groupRepo}
}

func (h *GroupHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)

	var req dto.CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.BadRequest("invalid request body"))
		return
	}

	if err := req.Validate(); err != nil {
		response.Error(w, err)
		return
	}

	g, err := h.groupService.Create(r.Context(), groupsvc.CreateInput{
		Name:     req.Name,
		OwnerID:  userID,
		Timezone: req.Timezone,
		Kind:     req.Kind,
		Emoji:    req.Emoji,
		Color:    req.Color,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, dto.ToGroupResponse(g))
}

func (h *GroupHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)

	groups, err := h.groupService.GetUserGroups(r.Context(), userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	result := make([]dto.GroupResponse, 0, len(groups))
	for _, g := range groups {
		result = append(result, dto.ToGroupResponse(g))
	}

	response.OK(w, result)
}

func (h *GroupHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	groupID := mustUUID(r, "groupId")

	g, err := h.groupService.GetGroup(r.Context(), groupID, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, dto.ToGroupResponse(g))
}

func (h *GroupHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	groupID := mustUUID(r, "groupId")
	var req dto.CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.BadRequest("invalid request body"))
		return
	}
	if err := req.Validate(); err != nil {
		response.Error(w, err)
		return
	}
	g, err := h.groupService.Update(r.Context(), groupsvc.UpdateInput{
		GroupID:  groupID,
		UserID:   userID,
		Name:     req.Name,
		Timezone: req.Timezone,
		Kind:     req.Kind,
		Emoji:    req.Emoji,
		Color:    req.Color,
	})
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, dto.ToGroupResponse(g))
}

func (h *GroupHandler) GetMembers(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	groupID := mustUUID(r, "groupId")

	// Verify requester is a member.
	member, err := h.groupRepo.FindMember(r.Context(), groupID, userID)
	if err != nil || !member.IsActive() {
		response.Error(w, apperror.Forbidden("access denied"))
		return
	}

	result, err := h.groupRepo.FindActiveMembersWithDetails(r.Context(), groupID)
	if err != nil {
		response.Error(w, apperror.Internal("fetching member details", err))
		return
	}

	response.OK(w, result)
}

func (h *GroupHandler) Invite(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	groupID := mustUUID(r, "groupId")

	var req dto.InviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.BadRequest("invalid request body"))
		return
	}

	if err := h.groupService.Invite(r.Context(), groupsvc.InviteInput{
		GroupID:   groupID,
		InviterID: userID,
		InviteeID: req.InviteeID,
	}); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

func (h *GroupHandler) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	groupID := mustUUID(r, "groupId")

	if err := h.groupService.AcceptInvitation(r.Context(), groupID, userID); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

func (h *GroupHandler) DeclineInvitation(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	groupID := mustUUID(r, "groupId")

	if err := h.groupService.DeclineInvitation(r.Context(), groupID, userID); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

func (h *GroupHandler) Leave(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	groupID := mustUUID(r, "groupId")

	if err := h.groupService.Leave(r.Context(), groupID, userID); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

func mustUserID(r *http.Request) uuid.UUID {
	id, _ := middleware.UserIDFromContext(r.Context())
	return id
}

func mustUUID(r *http.Request, param string) uuid.UUID {
	id, err := uuid.Parse(chi.URLParam(r, param))
	if err != nil {
		return uuid.Nil
	}
	return id
}

// formatOptionalTime converts *time.Time to a *string for the DTO.
func formatOptionalTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := fmt.Sprintf("%s", t.Format(time.RFC3339))
	return &s
}

func (h *GroupHandler) GetPendingInvitations(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	invitations, err := h.groupRepo.FindPendingInvitationsWithDetails(r.Context(), userID)
	if err != nil {
		response.Error(w, apperror.Internal("fetching invitations", err))
		return
	}
	response.OK(w, invitations)
}
