//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/kurt4ins/vk-segmentation/internal/repository/postgres"
	"github.com/kurt4ins/vk-segmentation/internal/service"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()

	ctr, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("segmentation"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		fmt.Println("start container:", err)
		os.Exit(1)
	}

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Println("connection string:", err)
		os.Exit(1)
	}

	if err := postgres.Migrate(dsn); err != nil {
		fmt.Println("migrate:", err)
		os.Exit(1)
	}

	testPool, err = postgres.NewPool(ctx, dsn)
	if err != nil {
		fmt.Println("pool:", err)
		os.Exit(1)
	}

	code := m.Run()

	testPool.Close()
	_ = testcontainers.TerminateContainer(ctr)
	os.Exit(code)
}

// env bundles the real repositories and services wired against the test pool.
type env struct {
	transactor  *postgres.Transactor
	segments    *postgres.SegmentRepo
	users       *postgres.UserRepo
	memberships *postgres.MembershipRepo
	history     *postgres.HistoryRepo

	segmentSvc    *service.SegmentService
	userSvc       *service.UserService
	membershipSvc *service.MembershipService
	rolloutSvc    *service.RolloutService
}

func newEnv(t *testing.T) *env {
	t.Helper()
	resetDB(t)

	tx := postgres.NewTransactor(testPool)
	seg := postgres.NewSegmentRepo(testPool)
	usr := postgres.NewUserRepo(testPool)
	mem := postgres.NewMembershipRepo(testPool)
	hist := postgres.NewHistoryRepo(testPool)

	return &env{
		transactor:  tx,
		segments:    seg,
		users:       usr,
		memberships: mem,
		history:     hist,

		segmentSvc:    service.NewSegmentService(seg, hist, tx, nil),
		userSvc:       service.NewUserService(usr, seg, mem, hist, tx),
		membershipSvc: service.NewMembershipService(usr, seg, mem, hist, tx),
		rolloutSvc:    service.NewRolloutService(usr, mem, hist, seg, tx, 100),
	}
}

func resetDB(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := testPool.Exec(ctx, `TRUNCATE user_segments, segment_history, users; DELETE FROM segments;`)
	if err != nil {
		t.Fatalf("reset db: %v", err)
	}
}
