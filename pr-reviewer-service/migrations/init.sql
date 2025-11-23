
CREATE TABLE teams (
	team_name VARCHAR(255) PRIMARY KEY
);

CREATE TABLE users (
	user_id VARCHAR(255) PRIMARY KEY,
	username VARCHAR(255) NOT NULL,
	team_name VARCHAR(255) NOT NULL,
	is_active BOOLEAN NOT NULL DEFAULT true,
	FOREIGN KEY (team_name) REFERENCES teams(team_name) ON DELETE RESTRICT
);

CREATE TABLE pull_requests (
	pull_request_id VARCHAR(255) PRIMARY KEY,
	pull_request_name VARCHAR(255) NOT NULL,
	author_id VARCHAR(255) NOT NULL,
	status VARCHAR(20) NOT NULL DEFAULT 'OPEN',
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	merged_at TIMESTAMP,
	FOREIGN KEY (author_id) REFERENCES users(user_id) ON DELETE RESTRICT,
	CHECK (status IN ('OPEN', 'MERGED'))
);

CREATE TABLE pr_reviewers (
	pull_request_id VARCHAR(255) NOT NULL,
	user_id VARCHAR(255) NOT NULL,
	PRIMARY KEY (pull_request_id, user_id),
	FOREIGN KEY (pull_request_id) REFERENCES pull_requests(pull_request_id) ON DELETE CASCADE,
	FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE RESTRICT
);

CREATE INDEX idx_users_team_name ON users(team_name);
CREATE INDEX idx_pull_requests_author_id ON pull_requests(author_id);
CREATE INDEX idx_pr_reviewers_user_id ON pr_reviewers(user_id);