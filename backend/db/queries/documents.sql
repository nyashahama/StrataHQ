-- name: CreateSchemeDocument :one
INSERT INTO scheme_documents (
    scheme_id, name, storage_key, file_type, category, size_bytes, uploaded_by_user_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetSchemeDocument :one
SELECT * FROM scheme_documents
WHERE id = $1
LIMIT 1;

-- name: ListSchemeDocuments :many
SELECT * FROM scheme_documents
WHERE scheme_id = $1
ORDER BY created_at DESC;

-- name: ListSchemeDocumentsByCategory :many
SELECT * FROM scheme_documents
WHERE scheme_id = $1 AND category = $2
ORDER BY created_at DESC;

-- name: DeleteSchemeDocument :exec
DELETE FROM scheme_documents
WHERE id = $1;
