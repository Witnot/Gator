-- name: GetPostsForUser :many
SELECT
    posts.id,
    posts.created_at,
    posts.updated_at,
    posts.title,
    posts.url,
    posts.description,
    posts.published_at,
    posts.feed_id,
    feeds.name AS feed_name,
    users.name AS user_name
FROM posts
INNER JOIN feeds ON posts.feed_id = feeds.id
INNER JOIN users ON feeds.user_id = users.id
WHERE users.id = $1
ORDER BY posts.published_at DESC NULLS LAST
LIMIT $2;
