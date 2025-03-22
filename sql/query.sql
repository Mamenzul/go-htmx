-- name: CreateUser :execresult
INSERT INTO users (username, password) VALUES (?, ?);

-- name: GetUser :one
SELECT * FROM users WHERE id = ?;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = ?;

-- name: CreateSession :execresult
INSERT INTO sessions (session_id, user_id, expires_at) VALUES (?, ?, ?);

-- name: GetSession :one
SELECT * FROM sessions WHERE session_id = ?;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE session_id = ?;