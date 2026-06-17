//go:build integration

package integration

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/kurt4ins/vk-segmentation/internal/domain"
)

func ctx(t *testing.T) context.Context {
	t.Helper()
	c, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return c
}

func countHistory(t *testing.T, op domain.Operation) int {
	t.Helper()
	var n int
	err := testPool.QueryRow(context.Background(),
		`SELECT count(*) FROM segment_history WHERE operation = $1`, string(op)).Scan(&n)
	require.NoError(t, err)
	return n
}

// add/remove is atomic with the matching history rows.
func TestAddRemoveAtomicityWithHistory(t *testing.T) {
	e := newEnv(t)
	c := ctx(t)

	_, err := e.segmentSvc.Create(c, "MAIL_GPT", nil)
	require.NoError(t, err)
	_, err = e.segmentSvc.Create(c, "VOICE_MSG", nil)
	require.NoError(t, err)

	user, err := e.userSvc.Register(c)
	require.NoError(t, err)

	active, err := e.membershipSvc.UpdateSegments(c, user.ID, []string{"MAIL_GPT", "VOICE_MSG"}, nil, nil)
	require.NoError(t, err)
	require.Len(t, active, 2)
	require.Equal(t, 2, countHistory(t, domain.OpAdd))

	// remove one; a single remove history row is written.
	active, err = e.membershipSvc.UpdateSegments(c, user.ID, nil, []string{"VOICE_MSG"}, nil)
	require.NoError(t, err)
	require.Len(t, active, 1)
	require.Equal(t, "MAIL_GPT", active[0].Slug)
	require.Equal(t, 1, countHistory(t, domain.OpRemove))

	// idempotent re-add writes no new history.
	_, err = e.membershipSvc.UpdateSegments(c, user.ID, []string{"MAIL_GPT"}, nil, nil)
	require.NoError(t, err)
	require.Equal(t, 2, countHistory(t, domain.OpAdd))
}

// TTL is enforced at read time regardless of the cleaner worker.
func TestTTLReadTimeFilter(t *testing.T) {
	e := newEnv(t)
	c := ctx(t)

	_, err := e.segmentSvc.Create(c, "TEMP_SEG", nil)
	require.NoError(t, err)
	user, err := e.userSvc.Register(c)
	require.NoError(t, err)

	ttl := 500 * time.Millisecond
	active, err := e.membershipSvc.UpdateSegments(c, user.ID, []string{"TEMP_SEG"}, nil, &ttl)
	require.NoError(t, err)
	require.Len(t, active, 1)

	// still physically present, but read filter hides it after expiry.
	time.Sleep(700 * time.Millisecond)
	active, err = e.membershipSvc.ListActive(c, user.ID)
	require.NoError(t, err)
	require.Empty(t, active)

	// the cleaner reclaims the row and writes a remove history entry.
	removed, err := e.membershipSvc.CleanExpired(c)
	require.NoError(t, err)
	require.Equal(t, 1, removed)
	require.Equal(t, 1, countHistory(t, domain.OpRemove))
}

// rollout materializes ~round(N*P/100) memberships.
func TestPercentRolloutCount(t *testing.T) {
	e := newEnv(t)
	c := ctx(t)

	const n = 50
	const percent = 40
	for i := 0; i < n; i++ {
		_, err := e.userSvc.Register(c)
		require.NoError(t, err)
	}

	seg, err := e.segmentSvc.Create(c, "AB_TEST", intp(percent))
	require.NoError(t, err)

	require.NoError(t, e.rolloutSvc.Apply(c, seg))

	var members int
	err = testPool.QueryRow(c,
		`SELECT count(*) FROM user_segments WHERE segment_id = $1`, seg.ID).Scan(&members)
	require.NoError(t, err)

	want := int(math.Round(float64(n) * percent / 100))
	require.Equal(t, want, members)

	// status flipped to applied and history matches members.
	got, err := e.segments.GetBySlug(c, "AB_TEST")
	require.NoError(t, err)
	require.Equal(t, domain.StatusApplied, got.Status)
	require.Equal(t, want, countHistory(t, domain.OpAdd))
}

// soft-deleting a segment cascades: memberships removed, remove history written.
func TestSoftDeleteCascade(t *testing.T) {
	e := newEnv(t)
	c := ctx(t)

	_, err := e.segmentSvc.Create(c, "MAIL_GPT", nil)
	require.NoError(t, err)

	const n = 5
	for i := 0; i < n; i++ {
		u, err := e.userSvc.Register(c)
		require.NoError(t, err)
		_, err = e.membershipSvc.UpdateSegments(c, u.ID, []string{"MAIL_GPT"}, nil, nil)
		require.NoError(t, err)
	}

	require.NoError(t, e.segmentSvc.Delete(c, "MAIL_GPT"))

	// segment gone from the list and from every user.
	_, err = e.segments.GetBySlug(c, "MAIL_GPT")
	require.ErrorIs(t, err, domain.ErrSegmentNotFound)

	var remaining int
	err = testPool.QueryRow(c, `SELECT count(*) FROM user_segments`).Scan(&remaining)
	require.NoError(t, err)
	require.Zero(t, remaining)

	require.Equal(t, n, countHistory(t, domain.OpAdd))
	require.Equal(t, n, countHistory(t, domain.OpRemove))
}

func intp(i int) *int { return &i }
