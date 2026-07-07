// Package storage defines the object-storage contract. Keys follow the
// per-tenant layout: {tenant_id}/{logos|products|employees|receipts|attachments}/...
package storage

import "context"

// Folder names inside a tenant's prefix.
const (
	FolderLogos       = "logos"
	FolderProducts    = "products"
	FolderEmployees   = "employees"
	FolderReceipts    = "receipts"
	FolderAttachments = "attachments"
)

type ObjectStorage interface {
	// Put stores an object and returns its key.
	Put(ctx context.Context, key string, data []byte, contentType string) error
	Delete(ctx context.Context, key string) error
	// PublicURL returns the browser-facing URL for a stored key.
	PublicURL(key string) string
}

// TenantKey builds a namespaced object key.
func TenantKey(tenantID, folder, filename string) string {
	return tenantID + "/" + folder + "/" + filename
}
