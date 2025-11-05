package mcp

// Default values for MCP tools
const (
	DefaultSearchLimit      = 5
	DefaultContextTruncate  = 200
)

// Error messages
const (
	ErrQueryRequired         = "query parameter is required and must be a string"
	ErrFilePathRequired      = "file_path parameter is required and must be a string"
	ErrDirectoryPathRequired = "directory_path parameter is required and must be a string"
	ErrInvalidPath           = "Invalid path: %v"
	ErrFailedToLoad          = "Failed to load document: %v"
	ErrFailedToIndex         = "Failed to index document: %v"
	ErrFailedToReindex       = "Failed to reindex document: %v"
	ErrFailedToSearch        = "Failed to search documents: %v"
	ErrFailedToList          = "Failed to list documents: %v"
	ErrFailedToDelete        = "Failed to delete document: %v"
	ErrFailedToGetStats      = "Failed to get stats: %v"
	ErrFailedToGetInfo       = "Failed to get document info: %v"
)

// Success messages
const (
	MsgNoDocumentsFound      = "No relevant documents found."
	MsgNoDocumentsInDir      = "No supported documents found in the directory."
	MsgNoDocumentsIndexed    = "No documents indexed yet."
	MsgDocumentDeleted       = "Successfully deleted document: %s"
	MsgDocumentIndexed       = "Successfully indexed document: %s (%d chunks created)"
	MsgDocumentReindexed     = "Successfully reindexed document: %s (%d chunks)"
)
