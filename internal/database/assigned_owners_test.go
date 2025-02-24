package database

import (
	"context"
	"sort"
	"testing"

	"github.com/sourcegraph/log/logtest"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssignedOwnersStore_ListAssignedOwnersForRepo(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()
	logger := logtest.Scoped(t)
	db := NewDB(logger, dbtest.NewDB(logger, t))
	ctx := context.Background()

	// Creating 2 users.
	user1, err := db.Users().Create(ctx, NewUser{Username: "user1"})
	require.NoError(t, err)
	user2, err := db.Users().Create(ctx, NewUser{Username: "user2"})
	require.NoError(t, err)

	// Creating 2 repos.
	err = db.Repos().Create(ctx, &types.Repo{ID: 1, Name: "github.com/sourcegraph/sourcegraph"})
	require.NoError(t, err)
	err = db.Repos().Create(ctx, &types.Repo{ID: 2, Name: "github.com/sourcegraph/sourcegraph2"})
	require.NoError(t, err)

	// Creating repo paths.
	_, err = db.QueryContext(ctx, "INSERT INTO repo_paths (repo_id, absolute_path, parent_id) VALUES (1, '', NULL), (1, 'src', 1), (1, 'src/abc', 2), (1, 'src/def', 2)")
	require.NoError(t, err)

	// Inserting assigned owners.
	store := AssignedOwnersStoreWith(db, logger)
	err = store.Insert(ctx, user1.ID, 1, "src", user2.ID)
	require.NoError(t, err)
	err = store.Insert(ctx, user2.ID, 1, "src/abc", user1.ID)
	require.NoError(t, err)
	err = store.Insert(ctx, user2.ID, 1, "src/def", user1.ID)
	require.NoError(t, err)

	// Getting assigned owners for a non-existent repo.
	owners, err := store.ListAssignedOwnersForRepo(ctx, 1337)
	require.NoError(t, err)
	assert.Empty(t, owners)

	// Getting assigned owners for a repo without owners.
	owners, err = store.ListAssignedOwnersForRepo(ctx, 2)
	require.NoError(t, err)
	assert.Empty(t, owners)

	// Getting assigned owners for a given repo.
	owners, err = store.ListAssignedOwnersForRepo(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, owners, 3)
	sort.Slice(owners, func(i, j int) bool {
		return owners[i].FilePath < owners[j].FilePath
	})
	// We are checking everything except timestamps, non-zero check is sufficient for them.
	assert.Equal(t, owners[0], &AssignedOwnerSummary{OwnerUserID: 1, RepoID: 1, FilePath: "src", WhoAssignedUserID: 2, AssignedAt: owners[0].AssignedAt})
	assert.NotZero(t, owners[0].AssignedAt)
	assert.Equal(t, owners[1], &AssignedOwnerSummary{OwnerUserID: 2, RepoID: 1, FilePath: "src/abc", WhoAssignedUserID: 1, AssignedAt: owners[1].AssignedAt})
	assert.NotZero(t, owners[1].AssignedAt)
	assert.Equal(t, owners[2], &AssignedOwnerSummary{OwnerUserID: 2, RepoID: 1, FilePath: "src/def", WhoAssignedUserID: 1, AssignedAt: owners[2].AssignedAt})
	assert.NotZero(t, owners[2].AssignedAt)
}

func TestAssignedOwnersStore_Insert(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()
	logger := logtest.Scoped(t)
	db := NewDB(logger, dbtest.NewDB(logger, t))
	ctx := context.Background()

	// Creating a user.
	user1, err := db.Users().Create(ctx, NewUser{Username: "user1"})
	require.NoError(t, err)

	// Creating a repo.
	err = db.Repos().Create(ctx, &types.Repo{ID: 1, Name: "github.com/sourcegraph/sourcegraph"})
	require.NoError(t, err)

	// Creating repo paths.
	_, err = db.QueryContext(ctx, "INSERT INTO repo_paths (repo_id, absolute_path, parent_id) VALUES (1, '', NULL), (1, 'src', 1)")
	require.NoError(t, err)

	store := AssignedOwnersStoreWith(db, logger)

	// Inserting assigned owner for non-existing path.
	err = store.Insert(ctx, user1.ID, 1, "no/way", user1.ID)
	assert.EqualError(t, err, `cannot find "no/way" path for repo with ID=1`)

	// Inserting assigned owner for non-existing repo.
	err = store.Insert(ctx, user1.ID, 1337, "src", user1.ID)
	assert.EqualError(t, err, `cannot find "src" path for repo with ID=1337`)

	// Successfully inserting assigned owner.
	err = store.Insert(ctx, user1.ID, 1, "src", user1.ID)
	require.NoError(t, err)
}
