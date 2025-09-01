-- name: GetFeeds :many
SELECT f.id, f.created_at, f.updated_at, f.name, f.url, u.name AS user_name
FROM feeds f
JOIN users u ON f.user_id = u.id;
