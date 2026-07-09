package dto

type InviteMemberRequest struct {
	FullName string `json:"full_name" binding:"required,min=2,max=120"`
	Email    string `json:"email" binding:"required,email"`
	Role     string `json:"role" binding:"required,oneof=manager cashier kitchen employee"`
}

type UpdateMemberRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=manager cashier kitchen employee"`
}

type AdminCreateTenantRequest struct {
	BusinessName  string `json:"business_name" binding:"required,min=2,max=120"`
	BusinessSlug  string `json:"business_slug" binding:"required,min=2,max=60"`
	OwnerFullName string `json:"owner_full_name" binding:"required,min=2,max=120"`
	OwnerEmail    string `json:"owner_email" binding:"required,email"`
}
