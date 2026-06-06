package test

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func connectPostgres(ctx context.Context, databaseURL string) (*pgx.Conn, error) {
	db, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return nil, err
	}

	err = dropReplication(ctx, db)
	if err != nil {
		return nil, err
	}

	err = truncateAll(ctx, db)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// dropReplication drops all publications and logical replication slots,
// terminating any backends holding a slot first.
func dropReplication(ctx context.Context, conn *pgx.Conn) error {
	_, err := conn.Exec(ctx, `
DO $$
DECLARE pub record;
BEGIN
    FOR pub IN SELECT pubname FROM pg_publication LOOP
        EXECUTE format('DROP PUBLICATION IF EXISTS %I', pub.pubname);
    END LOOP;
END $$;`)
	if err != nil {
		return err
	}

	_, err = conn.Exec(ctx, `
DO $$
DECLARE s record;
BEGIN
    FOR s IN SELECT slot_name, active, active_pid FROM pg_replication_slots
             WHERE slot_type = 'logical' LOOP
        IF s.active THEN PERFORM pg_terminate_backend(s.active_pid); END IF;
        PERFORM pg_drop_replication_slot(s.slot_name);
    END LOOP;
END $$;`)
	return err
}

// --- users ---

func createUser(ctx context.Context, conn *pgx.Conn, email, name string) (int64, error) {
	var id int64
	err := conn.QueryRow(ctx,
		"INSERT INTO users (email, name) VALUES ($1, $2) RETURNING id",
		email, name).Scan(&id)
	return id, err
}

func updateUser(ctx context.Context, conn *pgx.Conn, id int64, name string) error {
	_, err := conn.Exec(ctx,
		"UPDATE users SET name = $1, version = version + 1 WHERE id = $2", name, id)
	return err
}

func deleteUser(ctx context.Context, conn *pgx.Conn, id int64) error {
	_, err := conn.Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	return err
}

func truncateUsers(ctx context.Context, conn *pgx.Conn) error {
	_, err := conn.Exec(ctx, "TRUNCATE public.users RESTART IDENTITY CASCADE")
	return err
}

// --- posts ---

func createPost(ctx context.Context, conn *pgx.Conn, authorID int64, title, body string) (string, error) {
	var id string
	err := conn.QueryRow(ctx,
		"INSERT INTO posts (author_id, title, body) VALUES ($1, $2, $3) RETURNING id::text",
		authorID, title, body).Scan(&id)
	return id, err
}

func updatePost(ctx context.Context, conn *pgx.Conn, id, title string) error {
	_, err := conn.Exec(ctx,
		"UPDATE posts SET title = $1, version = version + 1 WHERE id = $2::uuid", title, id)
	return err
}

func deletePost(ctx context.Context, conn *pgx.Conn, id string) error {
	_, err := conn.Exec(ctx, "DELETE FROM posts WHERE id = $1::uuid", id)
	return err
}

func truncatePosts(ctx context.Context, conn *pgx.Conn) error {
	_, err := conn.Exec(ctx, "TRUNCATE public.posts RESTART IDENTITY CASCADE")
	return err
}

// --- comments (composite PK: id, author_id) ---

func createComment(ctx context.Context, conn *pgx.Conn, postID string, authorID int64, body string) (int64, error) {
	var id int64
	err := conn.QueryRow(ctx,
		"INSERT INTO comments (post_id, author_id, body) VALUES ($1::uuid, $2, $3) RETURNING id",
		postID, authorID, body).Scan(&id)
	return id, err
}

func updateComment(ctx context.Context, conn *pgx.Conn, id, authorID int64, body string) error {
	_, err := conn.Exec(ctx,
		"UPDATE comments SET body = $1, version = version + 1 WHERE id = $2 AND author_id = $3",
		body, id, authorID)
	return err
}

func deleteComment(ctx context.Context, conn *pgx.Conn, id, authorID int64) error {
	_, err := conn.Exec(ctx,
		"DELETE FROM comments WHERE id = $1 AND author_id = $2", id, authorID)
	return err
}

func truncateComments(ctx context.Context, conn *pgx.Conn) error {
	_, err := conn.Exec(ctx, "TRUNCATE public.comments RESTART IDENTITY CASCADE")
	return err
}

// --- events (composite PK: id, time; payload jsonb) ---

func createEvent(ctx context.Context, conn *pgx.Conn, userID int64, kind, payload string) (int64, error) {
	var id int64
	err := conn.QueryRow(ctx,
		"INSERT INTO events (user_id, kind, payload) VALUES ($1, $2, $3::jsonb) RETURNING id",
		userID, kind, payload).Scan(&id)
	return id, err
}

func updateEvent(ctx context.Context, conn *pgx.Conn, id int64, payload string) error {
	_, err := conn.Exec(ctx,
		"UPDATE events SET payload = $1::jsonb, version = version + 1 WHERE id = $2", payload, id)
	return err
}

func deleteEvent(ctx context.Context, conn *pgx.Conn, id int64) error {
	_, err := conn.Exec(ctx, "DELETE FROM events WHERE id = $1", id)
	return err
}

func truncateEvents(ctx context.Context, conn *pgx.Conn) error {
	_, err := conn.Exec(ctx, "TRUNCATE public.events RESTART IDENTITY CASCADE")
	return err
}

// --- combined ---

func truncateAll(ctx context.Context, conn *pgx.Conn) error {
	_, err := conn.Exec(ctx,
		"TRUNCATE public.users, public.posts, public.comments, public.events RESTART IDENTITY CASCADE")
	return err
}
