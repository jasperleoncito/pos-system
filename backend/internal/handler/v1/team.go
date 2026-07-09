package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/jasperleoncito/pos-system/backend/internal/dto"
	"github.com/jasperleoncito/pos-system/backend/internal/middleware"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/response"
	"github.com/jasperleoncito/pos-system/backend/internal/service"
)

type TeamHandler struct {
	team *service.TeamService
}

func NewTeamHandler(team *service.TeamService) *TeamHandler {
	return &TeamHandler{team: team}
}

// ListMembers godoc
//
//	@Summary	List team members with account details
//	@Tags		team
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	response.Envelope
//	@Router		/team [get]
func (h *TeamHandler) ListMembers(c *gin.Context) {
	members, err := h.team.ListMembers(c.Request.Context(), c.GetString(middleware.CtxTenantID))
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", members)
}

// InviteMember godoc
//
//	@Summary	Invite or create a staff account (emails a set-password link)
//	@Tags		team
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.InviteMemberRequest	true	"Member details"
//	@Success	200		{object}	response.Envelope
//	@Failure	409		{object}	response.ErrorEnvelope
//	@Router		/team [post]
func (h *TeamHandler) InviteMember(c *gin.Context) {
	var req dto.InviteMemberRequest
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.team.InviteMember(c.Request.Context(),
		c.GetString(middleware.CtxTenantID), c.GetString(middleware.CtxUserID),
		req.FullName, req.Email, req.Role)
	if err != nil {
		respondError(c, err)
		return
	}
	msg := "member added — they can sign in with their existing account"
	if result.UserCreated {
		msg = "invitation sent — they'll receive an email to set their password"
	}
	response.OK(c, msg, result)
}

// UpdateMemberRole godoc
//
//	@Summary	Change a team member's role
//	@Tags		team
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		userId	path		string						true	"User ID"
//	@Param		payload	body		dto.UpdateMemberRoleRequest	true	"New role"
//	@Success	200		{object}	response.Envelope
//	@Router		/team/{userId}/role [patch]
func (h *TeamHandler) UpdateMemberRole(c *gin.Context) {
	var req dto.UpdateMemberRoleRequest
	if !bindJSON(c, &req) {
		return
	}
	err := h.team.UpdateMemberRole(c.Request.Context(),
		c.GetString(middleware.CtxTenantID), c.GetString(middleware.CtxUserID),
		c.Param("userId"), req.Role)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "role updated", nil)
}

// RemoveMember godoc
//
//	@Summary	Remove a member from the team
//	@Tags		team
//	@Security	BearerAuth
//	@Produce	json
//	@Param		userId	path		string	true	"User ID"
//	@Success	200		{object}	response.Envelope
//	@Router		/team/{userId} [delete]
func (h *TeamHandler) RemoveMember(c *gin.Context) {
	err := h.team.RemoveMember(c.Request.Context(),
		c.GetString(middleware.CtxTenantID), c.GetString(middleware.CtxUserID),
		c.Param("userId"))
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "member removed", nil)
}

// ResendInvite godoc
//
//	@Summary	Resend the set-password invite email to a member
//	@Tags		team
//	@Security	BearerAuth
//	@Produce	json
//	@Param		userId	path		string	true	"User ID"
//	@Success	200		{object}	response.Envelope
//	@Router		/team/{userId}/resend-invite [post]
func (h *TeamHandler) ResendInvite(c *gin.Context) {
	err := h.team.ResendInvite(c.Request.Context(),
		c.GetString(middleware.CtxTenantID), c.GetString(middleware.CtxUserID),
		c.Param("userId"))
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "invite email sent", nil)
}

// AdminCreateTenant godoc
//
//	@Summary	Create a business with its owner account (super admin)
//	@Tags		admin
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.AdminCreateTenantRequest	true	"Business + owner"
//	@Success	200		{object}	response.Envelope
//	@Failure	409		{object}	response.ErrorEnvelope
//	@Router		/admin/tenants [post]
func (h *TeamHandler) AdminCreateTenant(c *gin.Context) {
	var req dto.AdminCreateTenantRequest
	if !bindJSON(c, &req) {
		return
	}
	t, err := h.team.AdminCreateBusiness(c.Request.Context(), c.GetString(middleware.CtxUserID),
		req.BusinessName, req.BusinessSlug, req.OwnerFullName, req.OwnerEmail)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "business created — the owner has been emailed", t)
}
